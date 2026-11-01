# Agent Guide: renderer-yaml

`renderer-yaml` loads Kubernetes YAML and YML files from any `io/fs` filesystem, including OS, embedded, and in-memory filesystems.

## Documentation

- [README](README.md) — overview and quick start.
- [Design](docs/design.md) — file loading and pipeline behavior.
- [Development](docs/development.md) — workflow and tests.

## Public API

The package is imported from `github.com/k8s-manifest-kit/renderer-yaml/pkg`.

- `yaml.New([]yaml.Source{...}, opts...)` creates a renderer.
- `yaml.NewEngine(source, opts...)` creates an `engine.Engine` for one source.
- `Source` contains `FS`, `Path`, and optional source-specific `PostRenderers`.
- Options include filters, transformers, post-renderers, source selectors, caching, source annotations, and content hashes.

The renderer uses `fs.Glob`, processes only `.yaml` and `.yml` files in sorted path and document order, and reports `ErrNoFilesMatched` when nothing matches. Render-time values are ignored. Caching is disabled unless `WithCache` is supplied; every registered source gets its own cache instance, whose default key is the `cache.YAMLSpec` path (`github.com/k8s-manifest-kit/renderer-yaml/pkg/cache`), and callers can customize the underlying cache with the shared `utilcache.WithKeyFunc`. Cache misses reread the source filesystem; entries live until the configured TTL expires.

Source annotations are disabled by default and include renderer type plus source file. Content hashes are enabled by default. Use shared annotation constants from `engine/pkg/types`.

## Development

Run commands from this directory:

```bash
make test
make test/race
make fmt
make lint
make lint/fix
make check
```

`make check` runs lint and vulnerability checks only; it does not run tests
or formatting.

Use `testing/fstest.MapFS` for unit tests, checked-in fixtures under `config/test/manifests`, Gomega assertions, and `t.Context()`. Keep tests focused and simple: use the smallest filesystem fixture that exercises the loading behavior, and add helpers or extra test files only when a concrete behavior or regression requires them.
