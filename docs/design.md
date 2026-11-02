# YAML Renderer Design

## Source model

`yaml.Source` contains an `io/fs.FS` and a path. The path may identify one file
or a glob. Source-specific post-renderers are supported, and source selectors
can exclude sources before loading.

The renderer accepts only `.yaml` and `.yml` paths. Matching files are sorted,
then YAML documents within each file are emitted in document order. Empty YAML
documents are ignored; malformed YAML returns an error with file context.

## Pipeline and values

The renderer loads and decodes each source, applies source post-renderers,
combines selected results, and applies renderer-level filters, transformers,
and post-renderers. Source post-renderers run on every render, including
renders served from cache. Render-time values are accepted by the shared API
but do not modify YAML file content.

Source annotations are opt-in. Content hashes use
`manifests.k8s-manifests-kit/content.hash` and are enabled by default.

### Option composition

Renderer options apply left to right. Functional options touch a single
field — `WithFilter`, `WithTransformer`, `WithPostRenderer`, and
`WithSourceSelector` append, and the boolean setters overwrite only their
own field — so they compose with anything around them. A
`RendererOptions` struct is stronger: it replaces the hook, selector, and
boolean fields outright (including with nil and false), while a nil
`CacheOptions` preserves what earlier options configured.

A hook before a struct survives only if the struct carries it; a hook after
a struct is kept:

```go
// The struct clears the filter added before it and disables hashing:
// its zero-valued Filters and ContentHash replace the previous state.
renderer, _ := yaml.New(sources,
    yaml.WithFilter(keepPods),
    yaml.RendererOptions{SourceAnnotations: true},
    yaml.WithContentHash(true), // re-enables what the struct turned off
)

// Hook after struct: the post-renderer is appended to the struct's state.
renderer, _ := yaml.New(sources,
    yaml.RendererOptions{SourceAnnotations: true, ContentHash: true},
    yaml.WithPostRenderer(label),
)
```

A struct that wants hashing must set `ContentHash: true` explicitly; a
`WithContentHash(true)` after the struct achieves the same. These rules
govern the renderer's own options only — the engine's additive options
(`engine.WithFilter` and friends) follow the engine's rules, not these.

### Document decoding

Files are decoded with the shared permissive helper
(`k8s.DecodeYAML` in `github.com/k8s-manifest-kit/pkg`): empty documents
and documents without nonempty string `kind` and `apiVersion` are skipped,
while YAML syntax, decode, and other conversion failures return an error
naming the zero-based document index, wrapped with the file's path. This is
document metadata checking, not Kubernetes schema validation, and there is
no strict decoding option; a skipped document may be an intended resource
with misspelled metadata. The full skip and error rules live in the shared
decoder's GoDoc.

### Hash stages

Objects decoded from a file are annotated in one step after decoding and
before the result is stored in the source's cache: source annotations are
applied first (when enabled), then the content hash. The render-origin
stamp — live or cache — follows cache storage and retrieval, and the
source, renderer, and engine hooks run last.

Consequences callers can rely on: the hash covers the complete object at
hashing time, including any metadata already present — previously applied
source annotations participate, as does any pre-existing annotation. It is
not a hash of the file's source bytes, and it is not idempotent. The
render-origin stamp and cache retrieval never rehash, and mutations made by
later hooks need not match the stored digest. The full annotation phase
contract is the engine's
[annotation contract](../../engine/docs/design.md#annotations); this
renderer follows it at the points described above.

## Cache

Caching is disabled unless `WithCache` is supplied; without it, every
render reads the source filesystem live. Every registered source gets its
own cache instance, so sources that share a path (or a custom cache key)
never return each other's cached objects; within one source, the default
cache key is the source path represented by `cache.YAMLSpec` from
`github.com/k8s-manifest-kit/renderer-yaml/pkg/cache` (constructed with
`cache.New`).
Callers can customize cache behavior with the shared cache API, for example
`WithCache(utilcache.WithKeyFunc(fn))` (importing
`github.com/k8s-manifest-kit/pkg/util/cache` as `utilcache`); there is no
renderer-specific cache-key
option. Key functions only select entries within a single source's cache:
a constant key cannot merge two sources, while within one source it
deliberately collapses renders onto a single entry.

Breaking change: `YAMLSpec` and the cache constructor moved from the root
`yaml` package to `github.com/k8s-manifest-kit/renderer-yaml/pkg/cache`
(`cache.YAMLSpec`, `cache.New`). The old root symbols are gone; key functions
that type-asserted `yaml.YAMLSpec` must assert `cache.YAMLSpec` instead.

When caching is enabled, the matched files are reread from the source
filesystem only on a cache miss. Cached entries live until the configured
TTL expires, so renders may return stale content after the underlying files
change; there is no content-sensitive invalidation.

See [`../AGENTS.md`](../AGENTS.md) and [`development.md`](development.md).
