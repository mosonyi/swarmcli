// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package stacksview

import (
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// loadFilesCmd loads files from the specified directory
func loadFilesCmd(dirPath string) tea.Cmd {
	return func() tea.Msg {
		files := []string{}

		// Expand ~ to home directory
		if strings.HasPrefix(dirPath, "~") {
			if homeDir, err := os.UserHomeDir(); err == nil {
				dirPath = strings.Replace(dirPath, "~", homeDir, 1)
			}
		}

		// Add parent directory option if not root
		if dirPath != "/" {
			files = append(files, "..")
		}

		entries, err := os.ReadDir(dirPath)
		if err != nil {
			return filesLoadedMsg{
				Path:  dirPath,
				Files: files,
				Error: err,
			}
		}

		// Separate directories and regular files
		var dirs []string
		var regFiles []string

		for _, entry := range entries {
			if entry.IsDir() {
				// Add directory with trailing slash
				dirs = append(dirs, filepath.Join(dirPath, entry.Name())+"/")
			} else {
				// Add all files (not just .tar like in contexts)
				regFiles = append(regFiles, filepath.Join(dirPath, entry.Name()))
			}
		}

		// Add directories first, then files
		files = append(files, dirs...)
		files = append(files, regFiles...)

		return filesLoadedMsg{
			Path:  dirPath,
			Files: files,
			Error: nil,
		}
	}
}
