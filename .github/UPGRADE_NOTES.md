### Trial licences must now carry an expiry

A `trial` licence with no expiry date no longer verifies. It is reported as
**invalid** rather than expired — nothing has lapsed, so the remedy is a
reissued trial key and not a renewal. Get a new one at
[swarmcli.io/be](https://swarmcli.io/be) and install it as before.

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
takes it for the licence change above. The Apache-2.0 build (`swarmcli_*_oss`,
`eldaratech/swarmcli:<version>-oss`) contains no licensed code and is untouched
by it; upgrading that one from v1.14.0 needs nothing at all.
