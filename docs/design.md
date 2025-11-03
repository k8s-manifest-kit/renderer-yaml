# YAML Renderer Design

## Overview

The YAML renderer provides programmatic loading and processing of Kubernetes YAML manifest files. It is the simplest renderer in the k8s-manifest-kit ecosystem, focusing on straightforward file loading without templating or overlay logic.

## Architecture

### Core Components

1. **Renderer** (`pkg/yaml.go`)
   - Main entry point implementing `types.Renderer`
   - Manages file discovery via glob patterns
   - Handles multi-document YAML parsing
   - Supports optional caching
   - Thread-safe for concurrent operations

2. **Source** (`pkg/yaml.go`)
   - Defines filesystem and glob pattern for file discovery
   - Minimal configuration (FS + Path)
   - No dynamic values (static files only)

3. **Options** (`pkg/yaml_option.go`)
   - Functional options pattern for renderer configuration
   - Supports filters, transformers, caching, source annotations, cache key customization

4. **Cache Keys** (`pkg/yaml_cache.go`)
   - `YAMLSpec`: Struct containing data for cache key generation
   - `CacheKeyFunc`: Type for custom cache key functions
   - Built-in functions: `DefaultCacheKey()`, `FastCacheKey()`, `PathOnlyCacheKey()`

5. **Engine Convenience** (`pkg/engine.go`)
   - `NewEngine()` function for simple single-source scenarios
   - Wraps renderer creation with engine setup

## Key Design Decisions

### 1. Filesystem Abstraction

The renderer uses Go's standard `fs.FS` interface:
- Supports any filesystem implementation
- Common use cases: `os.DirFS()`, `embed.FS`, `testing/fstest.MapFS`
- Enables testing without real files
- Works with embedded resources

```go
// Local filesystem
yaml.Source{
    FS:   os.DirFS("/path/to/manifests"),
    Path: "*.yaml",
}

// Embedded filesystem
//go:embed manifests
var manifestsFS embed.FS

yaml.Source{
    FS:   manifestsFS,
    Path: "manifests/*.yaml",
}

// Test filesystem
yaml.Source{
    FS: fstest.MapFS{
        "pod.yaml": &fstest.MapFile{Data: []byte(podYAML)},
    },
    Path: "*.yaml",
}
```

### 2. Glob Pattern Matching

Uses standard `fs.Glob()` for file discovery:
- Supports patterns like `*.yaml`, `*.yml`, `manifests/*.yaml`
- Pattern syntax depends on filesystem implementation
- Only `.yaml` and `.yml` files are processed
- Non-matching files are silently skipped

### 3. Multi-Document YAML Support

Automatically handles files with multiple documents:
- Documents separated by `---` (YAML standard)
- Each document becomes a separate `unstructured.Unstructured` object
- Uses `k8s.DecodeYAML()` from shared utilities
- Maintains document order within files

### 4. Caching Strategy

Path-based caching with customizable key generation:
- Cache key generation: customizable via `CacheKeyFunc`
- Default: reflection-based hashing of `YAMLSpec` (path)
- Alternative: `FastCacheKey()` / `PathOnlyCacheKey()` (just returns path)
- TTL-based expiration
- Deep cloning for cached results
- Transparent to caller

**Cache key functions:**
- `DefaultCacheKey()`: Uses `dump.ForHash()` on `YAMLSpec` (safest, slower)
- `FastCacheKey()`: Returns path directly (recommended for YAML, fastest)
- `PathOnlyCacheKey()`: Alias for `FastCacheKey()` for clarity

**Cache behavior:**
```go
// Same path = cache hit
r1.Process() // miss - loads files
r1.Process() // hit  - returns cached

// Different path = cache miss
r2 := yaml.New([]yaml.Source{{FS: fs, Path: "other/*.yaml"}})
r2.Process() // miss - different pattern

// Custom cache key function
r3 := yaml.New(
    []yaml.Source{{FS: fs, Path: "*.yaml"}},
    yaml.WithCache(),
    yaml.WithCacheKeyFunc(yaml.FastCacheKey()),
)
```

**Note:** For YAML renderer, `DefaultCacheKey()` and `FastCacheKey()` are functionally equivalent since YAML files are static (no dynamic values). `FastCacheKey()` is recommended for simplicity and performance.

### 5. Source Annotations

When enabled, adds tracking metadata:
- `k8s-manifest-kit.io/renderer`: `"yaml"`
- `k8s-manifest-kit.io/source.file`: File path within filesystem

**Note:** Unlike other renderers, YAML renderer only adds `source.file` (not `source.path`) since the path is just a glob pattern.

### 6. Thread Safety

Designed for concurrent use:
- Immutable configuration after creation
- Read-only filesystem access
- Cache with built-in concurrency support
- No shared mutable state

### 7. No Template Support

**Important:** The YAML renderer does NOT support:
- Template variables or expressions
- Dynamic value injection
- Conditional rendering
- Loops or computed values

For templating, use:
- `renderer-gotemplate` for Go templates
- `renderer-helm` for Helm charts

## Error Handling

The renderer follows Go error wrapping conventions:

```go
// Validation errors
fmt.Errorf("invalid source at index %d: %w", i, err)

// File discovery errors
fmt.Errorf("failed to match pattern %s: %w", pattern, err)

// File loading errors
fmt.Errorf("failed to load %s: %w", path, err)

// YAML parsing errors
fmt.Errorf("failed to decode YAML: %w", err)

// Pipeline errors
fmt.Errorf("error applying filters/transformers to YAML pattern %s: %w", path, err)
```

**Specific error types:**
- `ErrNoFilesMatched`: No files match the glob pattern
- `ErrPathIsDirectory`: Path points to a directory, not a file
- `ErrFsRequired`: Source.FS is nil
- `ErrPathEmpty`: Source.Path is empty or whitespace

## Comparison with Other Renderers

### YAML vs. Kustomize

| Feature | YAML Renderer | Kustomize Renderer |
|---------|--------------|-------------------|
| File loading | ✓ Simple glob | ✓ kustomization.yaml |
| Overlays | ✗ None | ✓ Full support |
| Patches | ✗ None | ✓ Strategic/JSON |
| ConfigMaps | ✗ None | ✓ Generated |
| Complexity | Low | Medium-High |

### YAML vs. Helm

| Feature | YAML Renderer | Helm Renderer |
|---------|--------------|---------------|
| File loading | ✓ Simple glob | ✓ Chart structure |
| Templates | ✗ None | ✓ Go templates |
| Values | ✗ Static only | ✓ Dynamic injection |
| Dependencies | ✗ None | ✓ Chart deps |
| Complexity | Low | High |

### YAML vs. GoTemplate

| Feature | YAML Renderer | GoTemplate Renderer |
|---------|--------------|---------------------|
| File loading | ✓ Simple glob | ✓ Template files |
| Templates | ✗ None | ✓ Go templates |
| Values | ✗ Static only | ✓ Dynamic injection |
| Functions | ✗ None | ✓ Custom funcs |
| Complexity | Low | Low-Medium |

## Use Cases

### Best For:
- Static Kubernetes manifests
- Pre-rendered YAML files
- Simple deployments without templating
- Testing with fixed manifests
- Embedded manifests in Go binaries
- CI/CD with pre-generated manifests

### Not Suitable For:
- Dynamic configuration (use GoTemplate or Helm)
- Environment-specific deployments (use Kustomize overlays)
- Complex multi-environment setups (use Helm)
- ConfigMap/Secret generation (use Kustomize)

## Performance Characteristics

### Time Complexity
- File discovery: O(n) where n = number of files in filesystem
- YAML parsing: O(m) where m = total size of matching files
- Caching: O(1) lookup

### Memory Usage
- Loads entire files into memory
- Parses all matching files at once
- Cache stores full object copies (deep clones)

### Optimization Strategies
1. Use specific glob patterns to limit file discovery
2. Enable caching for repeated renders
3. Split large manifest sets across multiple sources
4. Consider file size when embedding manifests

## Future Enhancements

Potential improvements (not currently implemented):
- Streaming YAML parsing for large files
- Parallel file loading
- Incremental cache updates
- File watcher integration for hot reload
- Validation against Kubernetes schemas

## Related Documentation

- [Development Guide](development.md) - How to work with the codebase
- [CLAUDE.md](../CLAUDE.md) - AI assistant reference
- [Engine Documentation](https://github.com/k8s-manifest-kit/engine) - Core engine patterns

