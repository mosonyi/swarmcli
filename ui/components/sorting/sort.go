// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package sorting

import (
	"sort"
)

// SortOrder represents the sort direction
type SortOrder int

const (
	Ascending SortOrder = iota
	Descending
)

// SortConfig holds the current sort configuration
type SortConfig struct {
	Field SortOrder
}

// SortArrow returns the visual indicator for sort direction
func SortArrow(order SortOrder) string {
	if order == Ascending {
		return "▲"
	}
	return "▼"
}

// SortStringField sorts a slice of items by a string field
func SortStringField[T any](items []T, ascending bool, getField func(T) string) {
	if ascending {
		sort.Slice(items, func(i, j int) bool {
			return getField(items[i]) < getField(items[j])
		})
	} else {
		sort.Slice(items, func(i, j int) bool {
			return getField(items[i]) > getField(items[j])
		})
	}
}

// SortIntField sorts a slice of items by an int field
func SortIntField[T any](items []T, ascending bool, getField func(T) int) {
	if ascending {
		sort.Slice(items, func(i, j int) bool {
			return getField(items[i]) < getField(items[j])
		})
	} else {
		sort.Slice(items, func(i, j int) bool {
			return getField(items[i]) > getField(items[j])
		})
	}
}
