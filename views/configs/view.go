package configsview

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/docker/docker/api/types/swarm"
)

type configItem struct {
	Name string
	ID   string
}

func (i configItem) FilterValue() string { return i.Name }
func (i configItem) Title() string       { return i.Name }
func (i configItem) Description() string { return fmt.Sprintf("ID: %s", i.ID) }

type itemDelegate struct{}

func (d itemDelegate) Height() int  { return 1 }
func (d itemDelegate) Spacing() int { return 0 }

func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	cfg := item.(configItem)
	selected := index == m.Index()
	if selected {
		fmt.Fprintf(w, "> %s", cfg.Name)
	} else {
		fmt.Fprintf(w, "  %s", cfg.Name)
	}
}

func configItemFromSwarm(c swarm.Config) configItem {
	return configItem{Name: c.Spec.Name, ID: c.ID}
}
