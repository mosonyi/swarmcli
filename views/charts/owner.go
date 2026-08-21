// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package chartsview

import (
	"strings"

	"github.com/Eldara-Tech/swarmcli/v2/charts"
)

// controllerOwnerPrefix is the id namespace swarmcli-cd stamps its releases
// with, and controllerOwnerName is how it is spelled on screen.
//
// Display vocabulary and nothing else: the engine never reads this, and what is
// stored on the swarm is unchanged. It is the one prefix an operator cannot
// recognise from something they typed — "apply/" carries the `owner:` they
// wrote in a release file, while "cd/" is a product abbreviating its own name.
const (
	controllerOwnerPrefix = "cd/"
	controllerOwnerName   = "swarmcli-cd/"
)

// ownerCell renders the owner stamp of one of release's revisions.
//
// The stamp is "<id>:release/<name>", and on a row for that release the
// resource half says nothing: a stamp is only evidence of ownership when it
// names the release carrying it, so where it matches it repeats the NAME
// column — and it is the longer half of the longest cell in the table.
//
// Where it does not match, or where the stamp does not parse at all, the whole
// thing is shown verbatim. Such a stamp is not evidence of owning anything on
// screen, so rendering it as an owner would assert something the engine itself
// does not believe, and it is the one case where the exact bytes are what the
// operator came to read.
func ownerCell(stamp, release string) string {
	if stamp == "" {
		return "—"
	}
	ref, err := charts.ParseOwner(stamp)
	if err != nil || ref.Kind != charts.OwnerKindRelease || ref.Name != release {
		return stamp
	}
	if id, ok := strings.CutPrefix(ref.ID, controllerOwnerPrefix); ok && id != "" {
		return controllerOwnerName + id
	}
	return ref.ID
}
