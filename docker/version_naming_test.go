package docker

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNextConfigVersionName_NoSuffix(t *testing.T) {
	require.Equal(t, "foo-v2", nextConfigVersionName("foo"))
}

func TestNextConfigVersionName_V1(t *testing.T) {
	require.Equal(t, "foo-v2", nextConfigVersionName("foo-v1"))
}

func TestNextConfigVersionName_V5(t *testing.T) {
	require.Equal(t, "foo-v6", nextConfigVersionName("foo-v5"))
}

func TestNextConfigVersionName_NestedDash(t *testing.T) {
	require.Equal(t, "my-config-v4", nextConfigVersionName("my-config-v3"))
}

func TestNextSecretVersionName_NoSuffix(t *testing.T) {
	require.Equal(t, "bar-v2", nextSecretVersionName("bar"))
}

func TestNextSecretVersionName_V1(t *testing.T) {
	require.Equal(t, "bar-v2", nextSecretVersionName("bar-v1"))
}

func TestNextSecretVersionName_V10(t *testing.T) {
	require.Equal(t, "bar-v11", nextSecretVersionName("bar-v10"))
}
