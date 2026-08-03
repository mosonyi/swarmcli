// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"fmt"
	"strings"
)

// OwnerKindRelease is the resource kind of a release-history record, the only
// thing an owner stamp names today.
const OwnerKindRelease = "release"

// OwnerRef identifies both who manages a resource and which resource the stamp
// was written for. It is encoded as "<id>:<kind>/<name>", e.g.
//
//	apply/prod-swarm:release/whoami
//
// The resource half is what makes the stamp verifiable, and it is there
// deliberately. A bare owner string cannot tell a resource this tool created
// from a copy of one: ArgoCD shipped exactly that as the
// app.kubernetes.io/instance label, and replaced it in 3.0 with an annotation
// naming the resource too. Reading a stamp back therefore means checking that
// it names the resource carrying it — one that does not is not evidence of
// ownership, so it is treated as unowned rather than trusted.
//
// Docker label values have no length limit, which is the other half of what
// ArgoCD had to work around (its label truncated at 63 bytes).
type OwnerRef struct {
	ID   string // the manifest or controller that manages the resource
	Kind string // OwnerKindRelease
	Name string // the resource's name on the swarm
}

// String encodes the reference for storage in a label or a release payload.
func (o OwnerRef) String() string { return o.ID + ":" + o.Kind + "/" + o.Name }

// ParseOwner decodes a stamp written by OwnerRef.String.
//
// The id may itself contain "/" (the "apply/<name>" convention), so the id is
// cut at the first ":" and only the remainder is split into kind and name.
func ParseOwner(s string) (OwnerRef, error) {
	id, rest, ok := strings.Cut(s, ":")
	if !ok {
		return OwnerRef{}, fmt.Errorf("owner stamp '%s' is not <id>:<kind>/<name>", s)
	}
	kind, name, ok := strings.Cut(rest, "/")
	if !ok {
		return OwnerRef{}, fmt.Errorf("owner stamp '%s' is not <id>:<kind>/<name>", s)
	}
	if id == "" || kind == "" || name == "" {
		return OwnerRef{}, fmt.Errorf("owner stamp '%s' has an empty id, kind or name", s)
	}
	return OwnerRef{ID: id, Kind: kind, Name: name}, nil
}

// validateOwnerID rejects an owner id that would not survive the encoding: ":"
// separates the id from the resource half, so an id containing one would decode
// as a different owner than it was written as.
func validateOwnerID(id string) error {
	switch {
	case id == "":
		return fmt.Errorf("owner is empty")
	case strings.Contains(id, ":"):
		return fmt.Errorf("invalid owner '%s': must not contain ':'", id)
	}
	return nil
}
