// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package helpview

import (
	"strings"

	"github.com/Eldara-Tech/swarmcli/v2/views/helpbar"
)

// FromKeys builds a help screen for a view that does not carry one of its own.
//
// It is deliberately derived rather than written: ShortHelpItems is part of the
// View contract, so every view — including one added tomorrow — has something
// to say here, and it is the same list the operator can already see in the help
// bar. A view with more to explain than its keys implements HelpContent instead;
// this is the floor, not the ceiling.
func FromKeys(title string, viewKeys, globalKeys []helpbar.HelpEntry) []HelpCategory {
	var categories []HelpCategory
	if items := toItems(viewKeys); len(items) > 0 {
		categories = append(categories, HelpCategory{Title: viewTitle(title), Items: items})
	}
	if items := toItems(globalKeys); len(items) > 0 {
		categories = append(categories, HelpCategory{Title: "Everywhere", Items: items})
	}
	return categories
}

// viewTitle turns a frame title into a category heading. Frame titles carry
// what the view is looking at ("Stats · web.1 · node-1"), which is noise in a
// heading; the first segment is the view.
func viewTitle(title string) string {
	name := strings.TrimSpace(strings.Split(title, "·")[0])
	if name == "" {
		return "Keys"
	}
	return name + " keys"
}

// toItems renders help-bar entries as help-screen items. Entries disabled right
// now are kept: the bar dims them rather than dropping them, and a help screen
// that lists only what happens to apply to the current row would answer "what
// can this view do" with a moving target.
func toItems(entries []helpbar.HelpEntry) []HelpItem {
	items := make([]HelpItem, 0, len(entries))
	for _, e := range entries {
		if e.Key == "" {
			continue
		}
		items = append(items, HelpItem{Keys: "<" + e.Key + ">", Description: e.Desc})
	}
	return items
}
