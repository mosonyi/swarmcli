// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	chartfileName    = "Chart.yaml"
	valuesName       = "values.yaml"
	schemaName       = "values.schema.json"
	readmeName       = "README.md"
	requirementsName = "requirements.yaml"
	templatesDir     = "templates"
)

// maxChartFileSize bounds any single file read from a chart archive to guard
// against decompression bombs in downloaded tarballs.
const maxChartFileSize = 10 << 20 // 10 MiB

// maxChartFiles and maxChartTotalSize bound the whole archive (entry count and
// cumulative decompressed size) so a hostile tarball cannot exhaust memory with
// many large or many tiny entries despite the per-entry cap.
const (
	maxChartFiles     = 4096
	maxChartTotalSize = 64 << 20 // 64 MiB
)

// LoadChartDir loads a chart from a directory on disk.
func LoadChartDir(dir string) (*Chart, error) {
	ch := &Chart{Templates: map[string]string{}}

	cf, err := os.ReadFile(filepath.Join(dir, chartfileName))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", chartfileName, err)
	}
	if err := yaml.Unmarshal(cf, &ch.Metadata); err != nil {
		return nil, fmt.Errorf("parse %s: %w", chartfileName, err)
	}

	if v, err := os.ReadFile(filepath.Join(dir, valuesName)); err == nil {
		if err := yaml.Unmarshal(v, &ch.Values); err != nil {
			return nil, fmt.Errorf("parse %s: %w", valuesName, err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read %s: %w", valuesName, err)
	}
	if ch.Values == nil {
		ch.Values = map[string]any{}
	}

	if s, err := os.ReadFile(filepath.Join(dir, schemaName)); err == nil {
		ch.Schema = s
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read %s: %w", schemaName, err)
	}

	if rq, err := os.ReadFile(filepath.Join(dir, requirementsName)); err == nil {
		if ch.Requirements, err = parseRequirements(rq); err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read %s: %w", requirementsName, err)
	}

	if r, err := os.ReadFile(filepath.Join(dir, readmeName)); err == nil {
		ch.Readme = string(r)
	}

	tdir := filepath.Join(dir, templatesDir)
	entries, err := os.ReadDir(tdir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("chart %q has no %s/ directory", dir, templatesDir)
		}
		return nil, fmt.Errorf("read %s/: %w", templatesDir, err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		body, err := os.ReadFile(filepath.Join(tdir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("read template %s: %w", e.Name(), err)
		}
		ch.Templates[templatesDir+"/"+e.Name()] = string(body)
	}

	if err := validateChart(ch); err != nil {
		return nil, err
	}
	return ch, nil
}

// LoadChartArchive loads a chart from a gzipped tar (.tgz) stream. The archive
// is expected to contain a single top-level directory (the chart), as produced
// by chart packaging; the leading directory component is stripped.
func LoadChartArchive(r io.Reader) (*Chart, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("open gzip: %w", err)
	}
	defer func() { _ = gz.Close() }()

	ch := &Chart{Templates: map[string]string{}, Values: map[string]any{}}
	tr := tar.NewReader(gz)
	var files int
	var total int64
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read archive: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		if files++; files > maxChartFiles {
			return nil, fmt.Errorf("archive has too many files (limit %d)", maxChartFiles)
		}
		rel := stripLeadingDir(path.Clean(hdr.Name))
		if rel == "" || strings.HasPrefix(rel, "..") {
			continue
		}
		body, err := io.ReadAll(io.LimitReader(tr, maxChartFileSize+1))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", hdr.Name, err)
		}
		if len(body) > maxChartFileSize {
			return nil, fmt.Errorf("archive entry %s exceeds %d bytes", rel, maxChartFileSize)
		}
		if total += int64(len(body)); total > maxChartTotalSize {
			return nil, fmt.Errorf("archive decompresses to more than %d bytes", maxChartTotalSize)
		}

		switch {
		case rel == chartfileName:
			if err := yaml.Unmarshal(body, &ch.Metadata); err != nil {
				return nil, fmt.Errorf("parse %s: %w", chartfileName, err)
			}
		case rel == valuesName:
			if err := yaml.Unmarshal(body, &ch.Values); err != nil {
				return nil, fmt.Errorf("parse %s: %w", valuesName, err)
			}
		case rel == schemaName:
			ch.Schema = body
		case rel == requirementsName:
			if ch.Requirements, err = parseRequirements(body); err != nil {
				return nil, err
			}
		case rel == readmeName:
			ch.Readme = string(body)
		case strings.HasPrefix(rel, templatesDir+"/") && !strings.ContainsRune(rel[len(templatesDir)+1:], '/'):
			ch.Templates[rel] = string(body)
		}
	}
	if ch.Values == nil {
		ch.Values = map[string]any{}
	}
	if err := validateChart(ch); err != nil {
		return nil, err
	}
	return ch, nil
}

// stripLeadingDir removes the first path component ("traefik/Chart.yaml" ->
// "Chart.yaml"). Packaged charts wrap their contents in a name directory.
func stripLeadingDir(p string) string {
	if i := strings.IndexByte(p, '/'); i >= 0 {
		return p[i+1:]
	}
	return p
}

func validateChart(ch *Chart) error {
	if ch.Metadata.Name == "" {
		return fmt.Errorf("Chart.yaml: name is required")
	}
	if ch.Metadata.Version == "" {
		return fmt.Errorf("Chart.yaml: version is required")
	}
	if len(ch.Templates) == 0 {
		return fmt.Errorf("chart %q has no templates", ch.Metadata.Name)
	}
	return nil
}
