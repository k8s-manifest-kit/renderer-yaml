# YAML Renderer Development

## Prerequisites and commands

Use Go 1.26.8 and run:

```bash
make test
make test/race
make fmt
make lint
make lint/fix
make check
```

## Testing

Cover filesystem errors, extension filtering, glob ordering, multi-document
ordering, empty documents, malformed YAML, source selectors, cache behavior,
and content-hash/source-annotation options. Use `testing/fstest.MapFS` for
portable filesystem fixtures where possible.

Do not assume render-time values are interpolated into YAML; that behavior
belongs to the Go template renderer.

`make test/race` re-runs the unit suite under the race detector. The
concurrency tests render overlapping requests against a single shared
renderer with same-path sources, so run it whenever changing cache handling,
source isolation, or output cloning.

See [`design.md`](design.md) and [`../AGENTS.md`](../AGENTS.md).
