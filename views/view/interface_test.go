// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package view

import (
	"testing"

	"swarmcli/views/helpbar"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

// framedStub is a view that does not implement Chromeless.
type framedStub struct{}

func (v *framedStub) Update(tea.Msg) tea.Cmd              { return nil }
func (v *framedStub) View() string                        { return "" }
func (v *framedStub) Init() tea.Cmd                       { return nil }
func (v *framedStub) Name() string                        { return "framed" }
func (v *framedStub) ShortHelpItems() []helpbar.HelpEntry { return nil }
func (v *framedStub) OnEnter() tea.Cmd                    { return nil }
func (v *framedStub) OnExit() tea.Cmd                     { return nil }
func (v *framedStub) HasErrors() bool                     { return false }
func (v *framedStub) FrameTitle() string                  { return "" }
func (v *framedStub) FrameHeader() string                 { return "" }
func (v *framedStub) FrameFooter() string                 { return "" }
func (v *framedStub) FrameContent() string                { return "" }

// chromelessStub opts in, with a configurable answer.
type chromelessStub struct {
	framedStub
	chromeless bool
}

func (v *chromelessStub) Chromeless() bool { return v.chromeless }

func TestIsChromeless_NotImplemented(t *testing.T) {
	require.False(t, IsChromeless(&framedStub{}))
}

func TestIsChromeless_OptedIn(t *testing.T) {
	require.True(t, IsChromeless(&chromelessStub{chromeless: true}))
}

func TestIsChromeless_OptedOut(t *testing.T) {
	require.False(t, IsChromeless(&chromelessStub{chromeless: false}))
}
