// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package view

import "strings"

// refSep packs multiple fields into a single Action argument. The unit
// separator can't appear in Docker resource names or swarm node IDs, so it is
// an unambiguous delimiter.
const refSep = "\x1f"

// EncodeRef packs fields into one Action argument string. It is the shared
// contract for actions that need more than a bare resource name — e.g. volume
// actions need the swarm node ID plus the volume name, because a volume name
// is only unique per node. Decode with DecodeRef.
func EncodeRef(parts ...string) string { return strings.Join(parts, refSep) }

// DecodeRef unpacks an argument produced by EncodeRef.
func DecodeRef(s string) []string { return strings.Split(s, refSep) }
