### Chart repositories must now be served over HTTPS

A chart repository serves the tarball that *becomes* the deployed workload, so
anything on the path to it decides what runs on your swarm. The digest published
per version does not close that: it travels in the same `index.yaml`, over the
same connection, so an on-path attacker rewrites both — and an entry that
publishes no digest only warns.

Plain `http://` is therefore refused: when adding a repository, when refreshing
its index, and for a tarball URL an index points at. A repository already in
`repos.json` from an earlier version is refused too, so this bites on
`swarmcli charts repo update`, `install`, `upgrade` and `apply`, not only on
`repo add`.

If your registry is internal and you already trust the network between you and
it, opt that machine out:

```bash
export SWARMCLI_CHARTS_ALLOW_PLAINTEXT=1
```

Otherwise, re-point the repository at its HTTPS URL:

```bash
swarmcli charts repo remove <name>
swarmcli charts repo add <name> https://…
```

### Repository names are restricted to `[A-Za-z0-9._-]`

The name is a component of the file its cached index is written to, so it is now
validated the same way a release name is — in the store and in
`swarmcli-release.yaml`. Any name outside that charset (a `/`, a `..` segment)
is rejected rather than resolved into a path.
