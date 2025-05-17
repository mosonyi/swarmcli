package utils

import (
	"github.com/charmbracelet/lipgloss"
	"strings"
)

//func HighlightMatches(text, searchTerm string) string {
//	if searchTerm == "" {
//		return text
//	}
//	re, err := regexp.Compile("(?i)" + regexp.QuoteMeta(searchTerm)) // case-insensitive
//	if err != nil {
//		return text // fail silently
//	}
//	highlighted := re.ReplaceAllStringFunc(text, func(match string) string {
//		return lipgloss.NewStyle().Background(lipgloss.Color("238")).Foreground(lipgloss.Color("229")).Render(match)
//	})
//	return highlighted
//}

var highlightStyle = lipgloss.NewStyle().Background(lipgloss.Color("205")).Foreground(lipgloss.Color("0"))

func HighlightMatches(text, term string, matches []int) string {
	if term == "" || len(matches) == 0 {
		return text
	}

	var b strings.Builder
	last := 0
	for _, idx := range matches {
		if idx < last || idx >= len(text) {
			continue
		}
		b.WriteString(text[last:idx])
		b.WriteString(highlightStyle.Render(text[idx : idx+len(term)]))
		last = idx + len(term)
	}
	b.WriteString(text[last:])
	return b.String()
}

func FindAllMatches(text, term string) []int {
	var matches []int
	textLower := strings.ToLower(text)
	termLower := strings.ToLower(term)
	idx := 0
	for {
		i := strings.Index(textLower[idx:], termLower)
		if i == -1 {
			break
		}
		matches = append(matches, idx+i)
		idx += i + len(term)
	}
	return matches
}
