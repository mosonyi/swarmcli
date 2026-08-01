// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOwnerRefRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		name string
		ref  OwnerRef
		want string
	}{
		{"apply", OwnerRef{ID: "apply/prod-swarm", Kind: OwnerKindRelease, Name: "whoami"}, "apply/prod-swarm:release/whoami"},
		{"controller", OwnerRef{ID: "cd/edge", Kind: OwnerKindRelease, Name: "traefik"}, "cd/edge:release/traefik"},
		// Release names admit '.', '-' and '_', none of which are separators.
		{"punctuated name", OwnerRef{ID: "apply/a", Kind: OwnerKindRelease, Name: "a.b-c_d"}, "apply/a:release/a.b-c_d"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, tc.ref.String())
			got, err := ParseOwner(tc.want)
			require.NoError(t, err)
			require.Equal(t, tc.ref, got)
		})
	}
}

// The id carries a '/' by convention ("apply/<name>"), so the split has to cut
// the id at the first ':' rather than the first '/'.
func TestParseOwnerKeepsSlashesInTheID(t *testing.T) {
	got, err := ParseOwner("apply/deeply/nested:release/hello")
	require.NoError(t, err)
	require.Equal(t, OwnerRef{ID: "apply/deeply/nested", Kind: OwnerKindRelease, Name: "hello"}, got)
}

func TestParseOwnerRejectsMalformedStamps(t *testing.T) {
	for _, s := range []string{
		"",
		"apply/prod",            // no resource half
		"apply/prod:release",    // no name
		":release/hello",        // no id
		"apply/prod:/hello",     // no kind
		"apply/prod:release/",   // empty name
		"apply/prod release/hi", // wrong separator
	} {
		_, err := ParseOwner(s)
		require.Error(t, err, "stamp %q must not parse", s)
	}
}

func TestValidateOwnerID(t *testing.T) {
	require.NoError(t, validateOwnerID("apply/prod-swarm"))
	require.Error(t, validateOwnerID(""))
	// ':' separates the id from the resource half; an id carrying one would
	// decode as a different owner than it was written as.
	require.Error(t, validateOwnerID("apply:prod"))
}

// --- stamping ---

// A stamped install records the owner in both places it has to be: the Config
// label, so a prune can find candidates without decoding every payload, and the
// payload itself, so the record stays self-describing.
func TestInstallStampsOwnerOnLabelAndPayload(t *testing.T) {
	fb := newFakeBackend()
	e := NewEngineWith(fb)
	ctx := context.Background()

	rel, err := e.Install(ctx, "hello", ReleaseChart{Name: "demo", Version: "0.1.0"}, nil,
		"services: {}\n", InstallOptions{Owner: "apply/prod-swarm"})
	require.NoError(t, err)
	require.Equal(t, "apply/prod-swarm:release/hello", rel.Owner)

	cfg := fb.configs["swarmcli.release.hello.v1"]
	require.Equal(t, "apply/prod-swarm:release/hello", cfg.labels[LabelOwner])

	stored, err := e.GetRevision(ctx, "hello", 1)
	require.NoError(t, err)
	require.Equal(t, "apply/prod-swarm:release/hello", stored.Owner)
}

// An unowned install is the default, and it must leave the label off entirely
// rather than blank: a prune filtering on the label's presence would otherwise
// match every release ever installed by hand.
func TestInstallWithoutOwnerLeavesNoLabel(t *testing.T) {
	fb := newFakeBackend()
	e := NewEngineWith(fb)

	rel, err := e.Install(context.Background(), "hello", ReleaseChart{Name: "demo", Version: "0.1.0"}, nil,
		"services: {}\n", InstallOptions{})
	require.NoError(t, err)
	require.Empty(t, rel.Owner)
	require.NotContains(t, fb.configs["swarmcli.release.hello.v1"].labels, LabelOwner)
}

// A malformed owner has to fail before anything is deployed: recording it would
// produce a stamp that decodes as a different owner than the caller passed.
func TestInstallRejectsMalformedOwnerBeforeDeploying(t *testing.T) {
	fb := newFakeBackend()
	e := NewEngineWith(fb)

	_, err := e.Install(context.Background(), "hello", ReleaseChart{Name: "demo", Version: "0.1.0"}, nil,
		"services: {}\n", InstallOptions{Owner: "apply:prod"})
	require.ErrorContains(t, err, "must not contain ':'")
	require.Empty(t, fb.deployed)
	require.Empty(t, fb.configs)
}

// The stamp is computed before the dry-run return, so a plan shows the ownership
// it would record without touching the swarm.
func TestDryRunReportsTheOwnerItWouldRecord(t *testing.T) {
	fb := newFakeBackend()
	e := NewEngineWith(fb)

	rel, err := e.Install(context.Background(), "hello", ReleaseChart{Name: "demo", Version: "0.1.0"}, nil,
		"services: {}\n", InstallOptions{Owner: "cd/edge", DryRun: true})
	require.NoError(t, err)
	require.Equal(t, "cd/edge:release/hello", rel.Owner)
	require.Empty(t, fb.configs)
}

// Every revision is stamped with whoever wrote it. A second manifest upgrading a
// release takes it over, and the history says exactly where the handover
// happened rather than silently keeping the first owner's claim alive.
func TestUpgradeByAnotherOwnerTakesTheReleaseOver(t *testing.T) {
	e := NewEngineWith(newFakeBackend())
	ctx := context.Background()
	chart := ReleaseChart{Name: "demo", Version: "0.1.0"}

	_, err := e.Install(ctx, "hello", chart, nil, "services: {}\n", InstallOptions{Owner: "apply/a"})
	require.NoError(t, err)
	_, err = e.Upgrade(ctx, "hello", chart, nil, "services: {b: {}}\n", InstallOptions{Owner: "apply/b"})
	require.NoError(t, err)

	hist, err := e.History(ctx, "hello")
	require.NoError(t, err)
	require.Len(t, hist, 2)
	require.Equal(t, "apply/a:release/hello", hist[0].Owner)
	require.Equal(t, "apply/b:release/hello", hist[1].Owner)
}

// A rollback is a new revision written by whoever ran it, so it carries the
// current owner rather than resurrecting the one the target revision recorded.
func TestRollbackStampsTheCurrentOwnerNotTheTargets(t *testing.T) {
	e := NewEngineWith(newFakeBackend())
	ctx := context.Background()
	chart := ReleaseChart{Name: "demo", Version: "0.1.0"}

	_, err := e.Install(ctx, "hello", chart, nil, "services: {}\n", InstallOptions{Owner: "apply/a"})
	require.NoError(t, err)
	_, err = e.Upgrade(ctx, "hello", chart, nil, "services: {b: {}}\n", InstallOptions{Owner: "apply/b"})
	require.NoError(t, err)

	rel, err := e.Rollback(ctx, "hello", 1, InstallOptions{Owner: "apply/b"})
	require.NoError(t, err)
	require.Equal(t, "apply/b:release/hello", rel.Owner)
	require.Equal(t, "services: {}\n", rel.Manifest, "rollback still restores the target's content")
}

// The owner is part of the desired state an apply establishes, so a plan
// classified against one owner must not report "unchanged" for a release
// stamped by another — that was #511, and it let a departed owner's stamp
// survive for ever behind a byte-identical file.
//
// The table is the whole contract, including the two cases that must NOT force
// a revision: an absent stamp (else every release predating owner stamping
// re-deploys once on upgrade) and an absent plan owner (else an apply claiming
// nothing strips a stamp somebody else wrote).
func TestUnchangedComparesTheOwner(t *testing.T) {
	const manifest = "services: {}\n"
	chart := ReleaseChart{Name: "demo", Version: "0.1.0"}
	rel := func(owner string) *Release {
		return &Release{Name: "hello", Status: StatusDeployed, Chart: chart, Manifest: manifest, Owner: owner}
	}

	for _, tc := range []struct {
		name  string
		stamp string
		owner string
		want  bool
	}{
		{"same owner is unchanged", "apply/a:release/hello", "apply/a", true},
		{"a different owner is a handover", "apply/a:release/hello", "apply/b", false},
		{"an unstamped release is not a mismatch", "", "apply/a", true},
		{"an empty plan owner never forces", "apply/a:release/hello", "", true},
		{"neither stamped nor claimed", "", "", true},
		// A stamp naming another release is not evidence this one is owned;
		// same for one that does not parse. Both get corrected.
		{"a stamp naming another release is a mismatch", "apply/a:release/elsewhere", "apply/a", false},
		{"an unparseable stamp is healed", "not-a-stamp", "apply/a", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := unchanged(rel(tc.stamp), chart, nil, manifest, tc.owner, nil)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

// End to end: taking a release over with an otherwise identical file writes one
// revision and the stamp is corrected. Renaming the owner of an unchanged
// release used to have no effect at all.
func TestApplyTakesOverAnUnchangedReleaseUnderANewOwner(t *testing.T) {
	e, fb, src, rf := applyEnv(t, oneRelease, "0.1.0")
	ctx := context.Background()

	plan, err := e.PlanApply(ctx, rf, src, PlanOptions{Owner: "cd/ctrl/old-app"})
	require.NoError(t, err)
	require.Equal(t, ActionInstall, plan.Releases[0].Action)
	_, err = e.Apply(ctx, plan, InstallOptions{Owner: plan.Owner})
	require.NoError(t, err)

	// Same file, same chart, same values — only the owner differs.
	plan2, err := e.PlanApply(ctx, rf, src, PlanOptions{Owner: "cd/ctrl/new-app"})
	require.NoError(t, err)
	require.Equal(t, ActionUpgrade, plan2.Releases[0].Action)

	res, err := e.Apply(ctx, plan2, InstallOptions{Owner: plan2.Owner})
	require.NoError(t, err)
	require.Equal(t, 2, res[0].Revision, "the handover is one revision, like any other real change")

	cur, err := e.GetRevision(ctx, "hello", 2)
	require.NoError(t, err)
	require.Equal(t, "cd/ctrl/new-app:release/hello", cur.Owner)

	// And it settles: the new owner's next apply is a no-op again.
	before := len(fb.configs)
	plan3, err := e.PlanApply(ctx, rf, src, PlanOptions{Owner: "cd/ctrl/new-app"})
	require.NoError(t, err)
	require.Equal(t, ActionUnchanged, plan3.Releases[0].Action)
	_, err = e.Apply(ctx, plan3, InstallOptions{Owner: plan3.Owner})
	require.NoError(t, err)
	require.Len(t, fb.configs, before, "a settled handover must not keep writing revisions")
}

// THE NO-MIGRATION GUARANTEE. A release installed before owner stamping existed
// carries no stamp. Planning it against an owner must stay ActionUnchanged, or
// the first apply after upgrading re-deploys every release on the swarm — every
// spec re-pushed, and with --resolve-image always, every digest re-resolved —
// to buy safety it already had, since an unstamped release is classified
// Unmanaged rather than Orphaned. Do not delete this test.
func TestApplyDoesNotChurnAnUnstampedRelease(t *testing.T) {
	e, fb, src, rf := applyEnv(t, oneRelease, "0.1.0")
	ctx := context.Background()

	plan, err := e.PlanApply(ctx, rf, src, PlanOptions{})
	require.NoError(t, err)
	_, err = e.Apply(ctx, plan, InstallOptions{})
	require.NoError(t, err)

	cur, err := e.GetRevision(ctx, "hello", 1)
	require.NoError(t, err)
	require.Empty(t, cur.Owner, "the fixture has to be unstamped for this to test anything")

	before := len(fb.configs)
	plan2, err := e.PlanApply(ctx, rf, src, PlanOptions{Owner: "apply/prod"})
	require.NoError(t, err)
	require.Equal(t, ActionUnchanged, plan2.Releases[0].Action)

	_, err = e.Apply(ctx, plan2, InstallOptions{Owner: plan2.Owner})
	require.NoError(t, err)
	require.Len(t, fb.configs, before, "an unstamped release must not be re-deployed to acquire a stamp")
}
