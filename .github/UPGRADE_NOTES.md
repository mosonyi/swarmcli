### Every package, formula and image dropped the `-be`

The Business Edition artefacts were renamed on 2026-08-07. The build is
unchanged and the executable on disk was always `swarmcli`; what changed is what
you ask the package manager for:

```bash
brew install Eldara-Tech/tap/swarmcli   # was swarmcli-be
scoop install swarmcli                  # was swarmcli-be
docker pull eldaratech/swarmcli         # was eldaratech/swarmcli-be
```

Release archives changed with them: `swarmcli_Linux_x86_64.tar.gz` for this
build and `swarmcli_Linux_x86_64_oss.tar.gz` for the wholly Apache-2.0 one, with
`checksums-merged.txt` and `checksums-oss.txt` beside them. There is no
unqualified `checksums.txt`, because a release carries two sets of artefacts and
one file would not say which set it verified.

The old names keep receiving the same build for a deprecation window. Homebrew
warns on install *and* upgrade — `deprecate!`, not a caveat, because someone who
only ever runs `brew upgrade` never reads caveats — and Scoop prints a rename
notice. Neither tool can express this as a rename, which is why it is here.

**One consequence is worth naming.** Anyone who was already on the `swarmcli`
cask or Scoop manifest — the Community Edition names — moves onto the merged
build with this release. Nothing they run changes: the licensed code is inert
until a licence verifies, and the binary behaves exactly as the Community
Edition did. If you need an artefact that provably contains no proprietary code,
that is `swarmcli-oss`, and it ships from every release — see
[docs/editions.md](../docs/editions.md).

### There is a permanent free tier

A licence need not be a paid one. The free tier is a signed key like any other,
verified offline, granting the same thirteen entitlements a paid `be` key grants
— including the five consumed by swarmcli-cd. It is not a reduced feature set
under a different name.

What bounds it is around the key rather than in it: **three nodes**, recorded as
`max_nodes`, and a **365-day term** that is rolled forward for you inside its
last 30 days. Nothing switches off on the node count; declining to roll the term
is the only lever, and that is also why the expiry is mandatory (below).

Get one with `swarmcli license activate`, or Register Cluster at
[swarmcli.io/licenses](https://swarmcli.io/licenses). One per account, bound to
one swarm, `bind: static` — so it has no lease and nothing to renew by hand.

**An older swarmcli cannot read a free key.** It verifies the signature, finds a
tier it has no entitlements for, and reports **Invalid**. The key is fine;
upgrade rather than ask for a different one.

### Trial *and free* licences must now carry an expiry

A `trial` or `free` licence with no expiry date no longer verifies. It is
reported as **invalid** rather than expired — nothing has lapsed, so the remedy
is a reissued key and not a renewal. Get a new one at
[swarmcli.io/be](https://swarmcli.io/be) and install it as before.

The two tiers fail the same way for different reasons. A trial's expiry is the
whole of the difference between it and a paid licence. A free licence's expiry
is the only thing that can ever *end* it, because the term is what the issuer
rolls forward — unexpiring, a free key would be a perpetual grant on every swarm
it reached.

`be` (paid Business Edition) licences are unaffected and may still be perpetual.

**This most likely reaches nobody.** Keys handed out by the issuing side have
always carried an expiry, so one without is not believed to exist. Confirm in a
second if you would rather be sure — a licence that reports valid on v1.14.0
reports valid on v2.0.0:

```bash
swarmcli license status
```

Upgrade the pieces you run to the same version. Mixed versions can leave one of
them honouring a key another refuses, which is awkward to diagnose rather than
dangerous.

### A paid licence is now signed `managed`, and activation is a second step

Binding used to be a property of the key alone: the swarm was named in it at
issuance and that was the end of it. A **managed** licence names the swarm the
same way and is then *activated* for it by a short-lived signed **lease**.
Installing the key is the first step; the lease is the second.

Paid licences issued from now on are signed `bind: managed`. Free-tier keys are
signed `bind: static` and have no lease. Keys issued before managed binding
existed are unaffected and keep working exactly as they did.

On a swarm that can reach the licence service, the second step takes care of
itself — the renewal check runs on a timer and again whenever you open
`:license`, so activation follows the key install on its own. Until a lease is
in place, `:license` reports **Not activated for this swarm** and Business
Edition features stay off. That is deliberate: with a managed licence the lease
*is* the binding, so its absence is an answer rather than an unknown.

Air-gapped and policy-restricted swarms take a lease as a file instead:

```bash
swarmcli license lease install ~/swarm.lease   # or :license lease install
```

**The client half of this ships in v2.0.0; lease issuance turns on separately
and is not live in production yet.** Nothing you run needs to change for it, and
`swarmcli license status` is the check that says which state a swarm is in.

Two consequences worth knowing before you meet them:

- A managed licence moves between swarms **at most once per lease window** — a
  lease that has been issued cannot be recalled. Unbind Cluster in the dashboard
  names the date the next move becomes possible. A licence bound at issuance has
  no such cap.
- `swarmcli license install` exits `0` on a managed key that grants nothing yet,
  and that is correct: the key is stored, and a managed key is not an activated
  licence until a lease arrives. `swarmcli license status` is the verb that
  answers "are the features on".

### `swarmcli license activate` — a licence in one command

The whole licence lifecycle is now reachable without a person at a TUI, which
matters for controllers, CI runners and any swarm nobody opens swarmcli
against:

```bash
swarmcli license activate                # get a licence for this swarm
swarmcli license install <file>          # store a key you already hold
swarmcli license lease install <file>    # activate from a lease file
swarmcli license status  [--json]        # what this swarm holds
swarmcli license sync    [--json]        # bring it up to date now
swarmcli license id                      # the id support asks for
```

`activate` opens an activation, prints a short code and a link, waits while you
confirm in a browser, and installs what comes back. It is the free tier's
primary path as well as a paid one's, and it makes no outbound call at all when
`SWARMCLI_DISABLE_LICENSE_RENEWAL` is set.

`:bootstrap` now deploys a fourth service, `licence-renewer`, which runs
`license sync` on a timer so a swarm nobody logs into keeps its licence current.
It is the only service in the stack with a route off the cluster, on a network
of its own, and `:bootstrap --no-renewer` omits it. A swarm bootstrapped before
the service existed cannot gain one from `--upgrade`; `:license` says so on an
`Auto-renewal:` line — see [docs/bootstrap.md](../docs/bootstrap.md).

`SWARMCLI_DISABLE_LICENSE_RENEWAL` switches off **both** licensing requests, on
every licence type — not only a managed licence's lease renewal. The other is a
daily token refresh, and it is what carries a renewal, a plan change or a rolled
free-tier term onto a swarm.

### Importing `swarmcli` as a Go module? The path gained a `/v2`

Go carries the major version in the module path from v2 on, so:

```go
import "github.com/Eldara-Tech/swarmcli/v2/charts"   // was .../swarmcli/charts
```

```bash
go get github.com/Eldara-Tech/swarmcli/v2@v2.0.0
```

Nothing about the packages themselves changed — same names, same signatures, so
the rewrite is the import lines and the `require`. Existing builds are not
affected until they choose to move: a `require github.com/Eldara-Tech/swarmcli
v1.14.0` goes on resolving v1.14.0 exactly as before.

### Everything else

There is no other breaking change. This repository carries none of its own in
v2.0.0 — the major is shared across the SwarmCLI components and this release
takes it for the licence changes above. The Apache-2.0 build (`swarmcli_*_oss`,
`eldaratech/swarmcli:<version>-oss`) contains no licensed code and is untouched
by any of them; upgrading that one from v1.14.0 needs nothing at all.
