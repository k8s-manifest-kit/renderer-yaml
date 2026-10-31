# YAML Renderer Development

## Prerequisites and commands

Use the Go version recorded in [`go.mod`](../go.mod) and run:

```bash
make test
make test/race
make fmt
make lint
make lint/fix
make check
```

`make check` runs lint and vulnerability checks only; it is not a substitute
for `make test` or `make test/race`, and formatting is a separate target.

## Code conventions

- Wrap errors with the failing operation and path context using `%w`, and
  check sentinel errors such as `ErrNoFilesMatched` and
  `ErrPathIsDirectory` with `errors.Is`.
- Give each function parameter its own explicit type.
- Render-time values are accepted by the shared API but ignored by this
  renderer; interpolation belongs to the Go template renderer, not here.
- User-supplied code — cache key functions, post-renderers, source
  selectors — runs inside `Process` and may overlap on a shared renderer.
  It must be safe for concurrent use; the renderer does not synchronize it.

### Multiline strings

- Declare every multiline string — fixtures, manifests, templates, and
  expected output — as one descriptive, standalone, package-level
  `const name =` with a raw string literal.
- Never embed multiline string literals in function or method bodies,
  and never declare them as function-local constants.
- Never group multiline constants in `const (...)`. Related single-line
  scalar constants can still be grouped, and single-line inline strings
  are unaffected.
- Keep large literal content separate from execution logic for
  readability. Preserve meaningful content indentation; the declaration
  layout must not introduce unintended bytes.
- A function may format the named template into a local variable. The
  prohibition concerns where the literal is declared, not where its
  string value is used.

```go
const fixtureYAML = `apiVersion: v1
kind: ConfigMap
`
```

### Test layout

- Colocate tests with implementation. A `test/` subdirectory is a separate Go
  package, not a cosmetic folder; use existing package boundaries.
- Organize larger suites by concern using `<component>_<concern>_test.go`;
  retain a cohesive `<component>_test.go` for core API behavior. Use
  `_metadata_test.go`, `_cache_test.go`, `_values_test.go`, and
  `_concurrency_test.go` where relevant, not as mandatory files for every
  package.
- Place benchmarks in `<component>_benchmark_test.go`, in the same package
  directory, retaining benchmark names and setup/timing behavior.
- Put shared helpers in `<component>_support_test.go` and substantial
  feature-specific fixture machinery in
  `<component>_<concern>_support_test.go`. Small test-local helpers and
  scenario assertions can remain beside their tests.
- Support files contain helper/fixture/type declarations, not test, benchmark,
  fuzz, or example entry points. `_support_test.go` is a naming convention;
  only `_test.go` has Go's test-file significance.
- Keep tests discoverable without excessive fragmentation: no hard line
  limit, mandatory one-file-per-test rule, shared cross-repository harness,
  fixture-storage migration, or automatic test-speed claim.
- Preserve integration build tags; filenames alone do not exclude a test from
  the offline suite. Offline cache/metrics interaction tests do not become
  live integration tests because of their function names.

## Testing

Cover filesystem errors, extension filtering, glob ordering, multi-document
ordering, empty documents, malformed YAML, source selectors, cache behavior,
and content-hash/source-annotation options. Use `testing/fstest.MapFS` as
immutable portable filesystem fixtures where possible, checked-in fixtures
under `config/test/manifests`, Gomega assertions, and `t.Context()`.

`make test/race` re-runs the unit suite under the race detector. The
concurrency tests render overlapping requests against a single shared
renderer with same-path sources, so run it whenever changing cache handling,
source isolation, or output cloning.

### Cache and output ownership

- Enable caching explicitly with `WithCache` and demonstrate hit/miss
  behavior through the render-origin annotation — live on the first render,
  cache on later renders — rather than callback or call counts.
- Sources are separated by cache instance, not by key: same-path sources on
  different filesystems must stay isolated even when a constant custom key
  (`cache.WithKeyFunc`) is used. Key choice only affects entry reuse within
  one source, where deliberately reusing an entry is supported.
- Prove output ownership by mutating a nested field of one returned result
  and asserting other results and later renders stay unmodified.
- Source post-renderers are reapplied on every render, including cache
  hits; warm renders must still show per-source post-renderer effects.
- Regression anchors: `TestCacheSourceIsolation` in `pkg/yaml_cache_test.go` and
  `TestConcurrentRendering` in `pkg/yaml_concurrency_test.go`.

See [`design.md`](design.md) and [`../AGENTS.md`](../AGENTS.md).
