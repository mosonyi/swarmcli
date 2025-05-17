package main

import (
	"github.com/charmbracelet/lipgloss"
	"regexp"
)

func highlightMatches(text, searchTerm string) string {
	if searchTerm == "" {
		return text
	}
	re, err := regexp.Compile("(?i)" + regexp.QuoteMeta(searchTerm)) // case-insensitive
	if err != nil {
		return text // fail silently
	}
	highlighted := re.ReplaceAllStringFunc(text, func(match string) string {
		return lipgloss.NewStyle().Background(lipgloss.Color("238")).Foreground(lipgloss.Color("229")).Render(match)
	})
	return highlighted
}
