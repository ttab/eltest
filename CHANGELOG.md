# Changelog

Everything from v0.4.0 onwards is documented here; earlier releases are not
reconstructed. The entries are derived from the release tags, and the linked
pull requests hold the detail.

## [v0.4.1] - 2026-09-06

**Behaviour change (minio image):** `NewMinio` now runs the tag it is given. It
never stored the tag on the container options, so every minio container was
started from `minio/minio:latest` no matter whether `Minio202302` or
`Minio202509` was passed — the constant only kept the containers apart in the
bootstrap map. A suite pinned to an older release has in fact been testing
against whatever `latest` pointed at that day, and will now get the release it
asked for. `NewPostgres`, `NewOpenSearch` and `NewValkey` were unaffected.

Changes:

- A failure to clean up a test-local backing service now reports the underlying
  error instead of printing `%!w(...)`.

## [v0.4.0] - 2026-09-05

**Breaking (toolchain):** the module declares Go 1.26.8, so a consumer still
building with 1.25 has to move to 1.26 before it will build.

Changes:

- New `Postgres18_6` constant, `18.6-alpine3.24`, which is how a service starts
  testing against Postgres 18: pass it to `NewPostgres` in place of
  `Postgres17_6`. The existing constants are untouched, and the round-trip and
  tern migration tests run against both 17 and 18.
- Dependency upgrades: Go to 1.26.8, tern to v2.4.3, minio-go to v7.3.0,
  go-redis to v9.22.0, and the docker, OpenTelemetry and `golang.org/x` sets.
