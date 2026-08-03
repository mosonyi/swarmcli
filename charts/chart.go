// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
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
	filesDir         = "files"
)

// chartAPIVersionV1 is the only Chart.yaml schema this build understands. An
// absent apiVersion means v1: charts predate the field, and none in the wild set
// it. Validating it is what reserves a future value for a format break — an
// older swarmcli must refuse a chart it cannot read, not load half of it.
const chartAPIVersionV1 = "v1"

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

// dirLoader holds the state one LoadChartDir call shares across every file it
// reads.
//
// The counters are per-CHART, not per-directory, because that is what the
// limits describe: a chart with 2048 templates and 2048 files under files/ is
// over a 4096-file limit even though neither walk is, and a per-walk counter
// would let it through while looking like it enforced something. Every read on
// every path — the top-level members, templates/ and files/ — therefore goes
// through readFile.
type dirLoader struct {
	root  string
	count int
	total int64
}

// refusal marks a failure that is this loader's decision rather than the
// filesystem's accident: a symlink, an irregular file, or a limit. It exists
// because README.md's read is deliberately best-effort, and "this chart is over
// the limit" must not be swallowed by the branch that swallows "there is no
// README".
type refusal struct{ error }

func isRefusal(err error) bool {
	var r refusal
	return errors.As(err, &r)
}

// readFile reads one chart member, named by its slash-separated path relative
// to the chart root, and is the only place LoadChartDir touches a file.
//
// It refuses a symlink instead of following it. os.DirEntry.IsDir and os.Stat
// both answer about a link's TARGET, so a templates/x.yaml -> /etc/passwd used
// to read as an ordinary template and its content became a template source;
// files/ would have made that a second and easier way to put arbitrary
// operator-side content into a swarm config. LoadChartArchive has the property
// for free — it skips every entry that is not tar.TypeReg — so this is the
// directory loader catching up, deliberately including templates/.
//
// A missing file returns an error satisfying os.IsNotExist, which is how the
// optional members keep tolerating their own absence.
func (l *dirLoader) readFile(rel string) ([]byte, error) {
	full := filepath.Join(l.root, filepath.FromSlash(rel))
	fi, err := os.Lstat(full)
	if err != nil {
		return nil, err
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return nil, refusal{fmt.Errorf("chart entry %s is a symlink", rel)}
	}
	if !fi.Mode().IsRegular() {
		return nil, refusal{fmt.Errorf("chart entry %s is not a regular file", rel)}
	}
	if l.count++; l.count > maxChartFiles {
		return nil, refusal{fmt.Errorf("chart has too many files (limit %d)", maxChartFiles)}
	}
	// Sized before reading rather than after: the point of the per-file cap is
	// not to hold 10 MiB+1 in memory to discover it was too big.
	if fi.Size() > maxChartFileSize {
		return nil, refusal{fmt.Errorf("chart entry %s exceeds %d bytes", rel, maxChartFileSize)}
	}
	body, err := os.ReadFile(full)
	if err != nil {
		return nil, err
	}
	if l.total += int64(len(body)); l.total > maxChartTotalSize {
		return nil, refusal{fmt.Errorf("chart contains more than %d bytes", maxChartTotalSize)}
	}
	return body, nil
}

// LoadChartDir loads a chart from a directory on disk.
func LoadChartDir(dir string) (*Chart, error) {
	ch := &Chart{Templates: map[string]string{}}
	l := &dirLoader{root: dir}

	cf, err := l.readFile(chartfileName)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", chartfileName, err)
	}
	if err := yaml.Unmarshal(cf, &ch.Metadata); err != nil {
		return nil, fmt.Errorf("parse %s: %w", chartfileName, err)
	}

	if v, err := l.readFile(valuesName); err == nil {
		if err := yaml.Unmarshal(v, &ch.Values); err != nil {
			return nil, fmt.Errorf("parse %s: %w", valuesName, err)
		}
		ch.ValuesRaw = v
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read %s: %w", valuesName, err)
	}
	if ch.Values == nil {
		ch.Values = map[string]any{}
	}

	if s, err := l.readFile(schemaName); err == nil {
		ch.Schema = s
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read %s: %w", schemaName, err)
	}

	if rq, err := l.readFile(requirementsName); err == nil {
		ch.RequirementsRaw = rq
		if err := ch.loadRequirements(rq); err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read %s: %w", requirementsName, err)
	}

	// A README this loader cannot read has never been fatal, and this does not
	// change that — but a refusal is not "cannot read", it is "will not", and
	// swallowing one would leave a symlinked README followed by nothing and a
	// blown limit reported nowhere.
	switch r, err := l.readFile(readmeName); {
	case err == nil:
		ch.Readme = string(r)
	case isRefusal(err):
		return nil, err
	}

	if err := l.loadTemplates(ch); err != nil {
		return nil, err
	}
	if err := l.loadFiles(ch); err != nil {
		return nil, err
	}

	if err := validateChart(ch); err != nil {
		return nil, err
	}
	return ch, nil
}

// loadTemplates reads templates/, which is deliberately flat: a subdirectory is
// skipped, exactly as LoadChartArchive skips a template path with a second
// slash in it.
func (l *dirLoader) loadTemplates(ch *Chart) error {
	tdir := filepath.Join(l.root, templatesDir)
	// Lstat first, so a templates -> /etc symlink is refused rather than
	// turning every file in the target into a template.
	if fi, err := os.Lstat(tdir); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		return refusal{fmt.Errorf("chart entry %s is a symlink", templatesDir)}
	}
	entries, err := os.ReadDir(tdir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("chart '%s' has no %s/ directory", l.root, templatesDir)
		}
		return fmt.Errorf("read %s/: %w", templatesDir, err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		rel := templatesDir + "/" + e.Name()
		body, err := l.readFile(rel)
		if err != nil {
			if isRefusal(err) {
				return err // already names the entry
			}
			return fmt.Errorf("read template %s: %w", e.Name(), err)
		}
		ch.Templates[rel] = string(body)
	}
	return nil
}

// loadFiles reads files/ recursively into ch.Files, keyed by the chart-relative
// slash path ("files/nginx.conf", "files/tls/ca.pem") so the key is the same on
// every platform and identical to the one LoadChartArchive produces from a tar
// header.
//
// An absent files/ is not an error and leaves ch.Files nil — every chart
// written before this existed loads exactly as it did.
func (l *dirLoader) loadFiles(ch *Chart) error {
	fdir := filepath.Join(l.root, filesDir)
	err := filepath.WalkDir(fdir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(l.root, p)
		if err != nil {
			return err
		}
		// WalkDir reports a symlink — to a file or to a directory — as a
		// non-directory entry, so readFile is what refuses both.
		key := filepath.ToSlash(rel)
		body, err := l.readFile(key)
		if err != nil {
			return err
		}
		if key == filesDir {
			// The walk root is a plain file called "files" rather than a
			// directory. It is not a chart file: every key here starts with
			// "files/", and LoadChartArchive ignores such an entry for exactly
			// that reason. Refusing the chart over it would be theatre.
			return nil
		}
		if ch.Files == nil {
			ch.Files = map[string][]byte{}
		}
		ch.Files[key] = body
		return nil
	})
	switch {
	case err == nil, os.IsNotExist(err):
		return nil
	case isRefusal(err):
		return err // already names the entry
	default:
		return fmt.Errorf("read %s/: %w", filesDir, err)
	}
}

// loadRequirements records the UNRENDERED view of requirements.yaml, shared by both
// loaders.
//
// requirements.yaml is a Go TEMPLATE: a chart may `range` over a user-supplied list
// (e.g. the extra overlays to attach to), and a template action on its own line is not
// parseable YAML. When the raw bytes do not parse we therefore keep no unrendered view
// and defer entirely to RenderRequirements — the AUTHORITATIVE parse, which renders
// with the release's values first and reports a genuinely broken file at
// template/install time.
//
// When the raw bytes DO parse we still validate eagerly, so a chart that declares
// something invalid (a nameless network) is rejected at load exactly as before.
func (ch *Chart) loadRequirements(raw []byte) error {
	req, err := unmarshalRequirements(raw)
	if err != nil {
		return nil // not YAML on its own => a template; RenderRequirements decides.
	}
	if err := validateRequirements(req); err != nil {
		return err
	}
	ch.Requirements = req
	return nil
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
			ch.ValuesRaw = body
		case rel == schemaName:
			ch.Schema = body
		case rel == requirementsName:
			ch.RequirementsRaw = body
			if err := ch.loadRequirements(body); err != nil {
				return nil, err
			}
		case rel == readmeName:
			ch.Readme = string(body)
		case strings.HasPrefix(rel, templatesDir+"/") && !strings.ContainsRune(rel[len(templatesDir)+1:], '/'):
			ch.Templates[rel] = string(body)
		case strings.HasPrefix(rel, filesDir+"/"):
			// No second-slash test, unlike templates/ above: files/ is read
			// recursively on purpose, so files/tls/ca.pem keeps its shape. The
			// count and size guards above already cover these entries, and the
			// tar.TypeReg test at the top of the loop has already dropped every
			// symlink, hardlink and device node.
			if ch.Files == nil {
				ch.Files = map[string][]byte{}
			}
			ch.Files[rel] = body
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
	switch ch.Metadata.APIVersion {
	case "", chartAPIVersionV1:
	default:
		return fmt.Errorf("Chart.yaml: unsupported apiVersion '%s' (expected '%s')",
			ch.Metadata.APIVersion, chartAPIVersionV1)
	}
	if len(ch.Templates) == 0 {
		return fmt.Errorf("chart '%s' has no templates", ch.Metadata.Name)
	}
	return nil
}
