package docker

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStructFieldsAsStringArray_Basic(t *testing.T) {
	type sample struct {
		Name string
		Age  int
	}
	result := StructFieldsAsStringArray(sample{Name: "alice", Age: 30})
	require.Equal(t, []string{"alice", "30"}, result)
}

func TestStructFieldsAsStringArray_EmptyStruct(t *testing.T) {
	type empty struct{}
	result := StructFieldsAsStringArray(empty{})
	require.Empty(t, result)
}

func TestStructFieldsAsStringArray_NonStruct(t *testing.T) {
	result := StructFieldsAsStringArray("not a struct")
	require.Empty(t, result)
}

func TestStructFieldsAsStringArray_MixedTypes(t *testing.T) {
	type mixed struct {
		Name    string
		Count   int
		Enabled bool
	}
	result := StructFieldsAsStringArray(mixed{Name: "test", Count: 42, Enabled: true})
	require.Equal(t, []string{"test", "42", "true"}, result)
}
