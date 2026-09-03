# YAML Renderer Development

## Prerequisites and commands

Use Go 1.26.8 and run:

```bash
make test
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

See [`design.md`](design.md) and [`../AGENTS.md`](../AGENTS.md).
