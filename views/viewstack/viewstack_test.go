package viewstack

import (
	"swarmcli/views/helpbar"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

type mockView struct {
	name string
}

func (m *mockView) Update(tea.Msg) tea.Cmd              { return nil }
func (m *mockView) View() string                        { return m.name }
func (m *mockView) Init() tea.Cmd                       { return nil }
func (m *mockView) Name() string                        { return m.name }
func (m *mockView) OnEnter() tea.Cmd                    { return nil }
func (m *mockView) OnExit() tea.Cmd                     { return nil }
func (m *mockView) HasErrors() bool                     { return false }
func (m *mockView) ShortHelpItems() []helpbar.HelpEntry { return nil }

func TestPush_And_Pop(t *testing.T) {
	s := &Stack{}
	s.Push(&mockView{name: "a"})
	s.Push(&mockView{name: "b"})

	v := s.Pop()
	require.Equal(t, "b", v.Name())
	v = s.Pop()
	require.Equal(t, "a", v.Name())
}

func TestPop_Empty(t *testing.T) {
	s := &Stack{}
	require.Nil(t, s.Pop())
}

func TestViews(t *testing.T) {
	s := &Stack{}
	s.Push(&mockView{name: "a"})
	s.Push(&mockView{name: "b"})

	views := s.Views()
	require.Len(t, views, 2)
	require.Equal(t, "a", views[0].Name())
	require.Equal(t, "b", views[1].Name())

	// Modifying returned slice should not affect stack
	views[0] = &mockView{name: "x"}
	require.Equal(t, "a", s.Views()[0].Name())
}

func TestLen(t *testing.T) {
	s := &Stack{}
	require.Equal(t, 0, s.Len())
	s.Push(&mockView{name: "a"})
	require.Equal(t, 1, s.Len())
	s.Push(&mockView{name: "b"})
	require.Equal(t, 2, s.Len())
}

func TestReset(t *testing.T) {
	s := &Stack{}
	s.Push(&mockView{name: "a"})
	s.Push(&mockView{name: "b"})
	s.Reset()
	require.Equal(t, 0, s.Len())
	require.Nil(t, s.Pop())
}
