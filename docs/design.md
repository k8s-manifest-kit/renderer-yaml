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

## Cache

Caching is disabled unless `WithCache` is supplied; without it, every
render reads the source filesystem live. Every registered source gets its
own cache instance, so sources that share a path (or a custom cache key)
never return each other's cached objects; within one source, the default
cache key is the source path represented by the renderer's `YAMLSpec`.
Callers can customize cache behavior with the shared cache API, for example
`WithCache(cache.WithKeyFunc(fn))`; there is no renderer-specific cache-key
option. Key functions only select entries within a single source's cache:
a constant key cannot merge two sources, while within one source it
deliberately collapses renders onto a single entry.

When caching is enabled, the matched files are reread from the source
filesystem only on a cache miss. Cached entries live until the configured
TTL expires, so renders may return stale content after the underlying files
change; there is no content-sensitive invalidation.

See [`../AGENTS.md`](../AGENTS.md) and [`development.md`](development.md).
