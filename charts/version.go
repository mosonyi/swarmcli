// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"strings"

	"golang.org/x/mod/semver"
)

// compareVersions orders two chart versions by SemVer. Versions are normalized
// to the "vX.Y.Z" form semver expects. When either side is not valid SemVer the
// comparison falls back to lexical ordering so results stay deterministic.
func compareVersions(a, b string) int {
	va, vb := normalizeSemver(a), normalizeSemver(b)
	if semver.IsValid(va) && semver.IsValid(vb) {
		return semver.Compare(va, vb)
	}
	return strings.Compare(a, b)
}

func normalizeSemver(v string) string {
	if v == "" {
		return v
	}
	if v[0] != 'v' {
		return "v" + v
	}
	return v
}
