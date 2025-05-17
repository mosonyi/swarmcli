package utils

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
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

func HighlightMatches(text, term string) string {
	matches := FindAllMatches(text, term)
	if len(matches) == 0 {
		return text
	}

	var highlighted strings.Builder
	cursor := 0
	style := termenv.String().Foreground(termenv.ANSIBrightYellow).Background(termenv.ANSIBlue)

	for _, match := range matches {
		start := match
		end := match + len(term)
		if start > cursor {
			highlighted.WriteString(text[cursor:start])
		}
		highlighted.WriteString(style.Styled(text[start:end]))
		cursor = end
	}
	if cursor < len(text) {
		highlighted.WriteString(text[cursor:])
	}

	return highlighted.String()
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
