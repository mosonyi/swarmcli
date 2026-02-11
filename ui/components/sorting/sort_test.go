package sorting

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSortArrow_Ascending(t *testing.T) {
	require.Equal(t, "▲", SortArrow(Ascending))
}

func TestSortArrow_Descending(t *testing.T) {
	require.Equal(t, "▼", SortArrow(Descending))
}

type item struct {
	Name  string
	Value int
}

func TestSortStringField_Ascending(t *testing.T) {
	items := []item{{Name: "charlie"}, {Name: "alice"}, {Name: "bob"}}
	SortStringField(items, true, func(i item) string { return i.Name })
	require.Equal(t, "alice", items[0].Name)
	require.Equal(t, "bob", items[1].Name)
	require.Equal(t, "charlie", items[2].Name)
}

func TestSortStringField_Descending(t *testing.T) {
	items := []item{{Name: "alice"}, {Name: "charlie"}, {Name: "bob"}}
	SortStringField(items, false, func(i item) string { return i.Name })
	require.Equal(t, "charlie", items[0].Name)
	require.Equal(t, "bob", items[1].Name)
	require.Equal(t, "alice", items[2].Name)
}

func TestSortStringField_Empty(t *testing.T) {
	var items []item
	SortStringField(items, true, func(i item) string { return i.Name })
	require.Empty(t, items)
}

func TestSortIntField_Ascending(t *testing.T) {
	items := []item{{Value: 3}, {Value: 1}, {Value: 2}}
	SortIntField(items, true, func(i item) int { return i.Value })
	require.Equal(t, 1, items[0].Value)
	require.Equal(t, 2, items[1].Value)
	require.Equal(t, 3, items[2].Value)
}

func TestSortIntField_Descending(t *testing.T) {
	items := []item{{Value: 1}, {Value: 3}, {Value: 2}}
	SortIntField(items, false, func(i item) int { return i.Value })
	require.Equal(t, 3, items[0].Value)
	require.Equal(t, 2, items[1].Value)
	require.Equal(t, 1, items[2].Value)
}
