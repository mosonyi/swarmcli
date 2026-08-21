// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package helpview

import (
	"testing"

	"github.com/Eldara-Tech/swarmcli/v2/views/helpbar"

	"github.com/stretchr/testify/require"
)

func TestFromKeys_SplitsTheViewsKeysFromTheAppsOwn(t *testing.T) {
	categories := FromKeys("Volumes",
		[]helpbar.HelpEntry{{Key: "d", Desc: "Delete"}},
		[]helpbar.HelpEntry{{Key: "ctrl+q", Desc: "Quit"}},
	)

	require.Len(t, categories, 2)
	require.Equal(t, "Volumes keys", categories[0].Title)
	require.Equal(t, []HelpItem{{Keys: "<d>", Description: "Delete"}}, categories[0].Items)
	require.Equal(t, "Everywhere", categories[1].Title)
	require.Equal(t, []HelpItem{{Keys: "<ctrl+q>", Description: "Quit"}}, categories[1].Items)
}

// TestFromKeys_DropsTheFrameTitlesSubject keeps the heading about the view
// rather than about the row it happens to be looking at.
func TestFromKeys_DropsTheFrameTitlesSubject(t *testing.T) {
	categories := FromKeys("Stats · web.1 · node-1",
		[]helpbar.HelpEntry{{Key: "w", Desc: "Window"}}, nil)

	require.Equal(t, "Stats keys", categories[0].Title)
}

func TestFromKeys_UntitledViewStillGetsAHeading(t *testing.T) {
	categories := FromKeys("  ", []helpbar.HelpEntry{{Key: "w", Desc: "Window"}}, nil)

	require.Equal(t, "Keys", categories[0].Title)
}

// TestFromKeys_KeepsWhatIsDisabledRightNow: the bar dims those entries rather
// than hiding them, and "what can this view do" should not depend on which row
// the cursor is on.
func TestFromKeys_KeepsWhatIsDisabledRightNow(t *testing.T) {
	categories := FromKeys("Stats",
		[]helpbar.HelpEntry{{Key: "n/N", Desc: "Replica", Disabled: true}}, nil)

	require.Equal(t, []HelpItem{{Keys: "<n/N>", Description: "Replica"}}, categories[0].Items)
}

func TestFromKeys_OmitsAnEmptyHalf(t *testing.T) {
	require.Empty(t, FromKeys("Loading", nil, nil))

	categories := FromKeys("Loading", nil, []helpbar.HelpEntry{{Key: "?", Desc: "Help"}})
	require.Len(t, categories, 1, "a view with no keys of its own still gets the app's")
	require.Equal(t, "Everywhere", categories[0].Title)
}

func TestFromKeys_SkipsAnEntryWithNoKey(t *testing.T) {
	categories := FromKeys("Stats", []helpbar.HelpEntry{{Desc: "orphaned"}, {Key: "w", Desc: "Window"}}, nil)

	require.Equal(t, []HelpItem{{Keys: "<w>", Description: "Window"}}, categories[0].Items)
}
