# AI Assistant Guide for k8s-manifest-kit/renderer-yaml

## Quick Reference

This is the **YAML renderer** for the k8s-manifest-kit ecosystem. It provides programmatic loading of Kubernetes YAML manifest files with features like glob pattern matching, caching, filtering, transformation, and source tracking.

### Repository Structure
- `pkg/` - Main renderer implementation
- `config/test/manifests/` - Test fixtures (sample YAML files)
- `docs/` - Architecture and development documentation

### Key Files
- `pkg/yaml.go` - Main renderer (`New()`, `Process()`)
- `pkg/yaml_option.go` - Functional options (`WithCache()`, `WithFilter()`, etc.)
- `pkg/yaml_cache.go` - Cache key generation (`YAMLSpec`, `CacheKeyFunc`)
- `pkg/yaml_support.go` - Helper functions and validation
- `pkg/engine.go` - Convenience function (`NewEngine()`)

### Related Repositories
- `github.com/k8s-manifest-kit/engine` - Core engine and types
- `github.com/k8s-manifest-kit/pkg` - Shared utilities (cache, errors, k8s utilities)

## Common Tasks

### Understanding the Code

**Q: How does the renderer work?**
1. Sources specify filesystem and glob pattern for YAML files
2. Glob pattern matches files (e.g., `*.yaml`, `manifests/**/*.yml`)
3. Each matched file is loaded and parsed (supports multi-document YAML)
4. Results are filtered/transformed per pipeline configuration
5. Objects are cached based on path

**Q: What's the difference between `New()` and `NewEngine()`?**
- `New()` creates a `Renderer` implementing `types.Renderer`
- `NewEngine()` creates an `engine.Engine` with a single YAML renderer (convenience)

**Q: Does the YAML renderer support templates or dynamic values?**
No, the YAML renderer is for static YAML files only. For templating, use:
- `renderer-gotemplate` for Go templates
- `renderer-helm` for Helm charts

### Making Changes

**Adding a renderer option:**
1. Add field to `RendererOptions` struct
2. Create `WithXxx()` function returning `RendererOption`
3. Add test coverage
4. Update documentation

**Adding a source option:**
1. Add field to `Source` struct
2. Update `Validate()` if needed
3. Handle in renderer processing logic
4. Add test coverage

**Modifying caching:**
- Cache logic is in `pkg/yaml.go` `renderSingle()`
- Cache key generation is in `pkg/yaml_cache.go`
- Cache key: customizable via `CacheKeyFunc` (default uses reflection-based hashing)
- Uses `github.com/k8s-manifest-kit/pkg/util/cache`

### Testing

**Run tests:**
```bash
make test
```

**Test structure:**
- Unit tests in `pkg/*_test.go`
- Test fixtures use `testing/fstest.MapFS` for in-memory filesystems
- Uses Gomega assertions (dot import)

**Key test files:**
- `yaml_test.go` - Main renderer tests
- `engine_test.go` - NewEngine tests

### Debugging

**Common issues:**
1. **Glob patterns**: Use filesystem-specific patterns (no `**` on some FS types)
2. **File extensions**: Only `.yaml` and `.yml` files are processed
3. **Multi-document YAML**: Separated by `---`, all parsed automatically
4. **Import paths**: Must use `github.com/k8s-manifest-kit/*`

**Useful debugging:**
```bash
# Run specific test
go test -v ./pkg -run TestRenderer

# Run with verbose output
go test -v ./...

# Check a specific glob pattern
go test -v ./pkg -run "TestRenderer/should_load_multiple"
```

## Architecture Notes

### Thread Safety
The renderer is thread-safe:
- Configuration is immutable after creation
- Filesystem access is read-only
- Cache has built-in concurrency support

### Filesystem Support
Uses Go's `fs.FS` interface:
- `os.DirFS()` for local directories
- `embed.FS` for embedded files
- `testing/fstest.MapFS` for tests
- Any custom `fs.FS` implementation

### Multi-Document YAML
Files with multiple documents (separated by `---`) are automatically split into individual objects using `k8s.DecodeYAML()` from the shared pkg utilities.

### Pipeline Integration
The renderer integrates with the three-level pipeline:
1. **Renderer-specific** (via `New()` options)
2. **Engine-level** (via `engine.New()` options)
3. **Render-time** (via `engine.Render()` options)

## Development Tips

1. **Follow established patterns** from kustomize/helm renderers
2. **Use functional options** for all configuration
3. **Document non-obvious behavior** in comments
4. **Test with realistic manifests** in config/test/
5. **Check the linter** (`make lint`) - it's aggressive
6. **Keep imports organized** per `.golangci.yml` rules

## Code Review Checklist

When reviewing changes:
- [ ] Tests added for new functionality
- [ ] Error messages are clear and actionable
- [ ] Documentation updated (design.md, development.md)
- [ ] Follows Go conventions (parameter types, etc.)
- [ ] Thread safety considered
- [ ] Linter passes
- [ ] Imports use new k8s-manifest-kit paths

## Common Patterns

### Creating a renderer:
```go
r, err := yaml.New(
    []yaml.Source{{
        FS:   os.DirFS("/path/to/manifests"),
        Path: "*.yaml",
    }},
    yaml.WithCache(cache.WithTTL(5*time.Minute)),
    yaml.WithCacheKeyFunc(yaml.FastCacheKey()), // Optional: customize cache key
)
```

### Using NewEngine:
```go
e, err := yaml.NewEngine(
    yaml.Source{
        FS:   os.DirFS("/path/to/manifests"),
        Path: "deployments/*.yaml",
    },
    yaml.WithSourceAnnotations(true),
)
```

### With embedded files:
```go
//go:embed manifests
var manifestsFS embed.FS

e, err := yaml.NewEngine(
    yaml.Source{
        FS:   manifestsFS,
        Path: "manifests/*.yaml",
    },
)
```

### Multiple sources:
```go
r, err := yaml.New(
    []yaml.Source{
        {FS: os.DirFS("/path1"), Path: "*.yaml"},
        {FS: os.DirFS("/path2"), Path: "*.yml"},
    },
)
```

### Custom cache key function:
```go
// Use path-only caching (recommended for YAML)
r, err := yaml.New(
    []yaml.Source{{FS: fs, Path: "*.yaml"}},
    yaml.WithCache(),
    yaml.WithCacheKeyFunc(yaml.FastCacheKey()),
)

// Or use default reflection-based caching
r, err := yaml.New(
    []yaml.Source{{FS: fs, Path: "*.yaml"}},
    yaml.WithCache(),
    yaml.WithCacheKeyFunc(yaml.DefaultCacheKey()),
)
```

## Key Differences from Other Renderers

**vs. Kustomize:**
- YAML renderer: Simple file loading, no patching or overlays
- Kustomize renderer: Full kustomization processing with overlays and patches

**vs. Helm:**
- YAML renderer: Static files only, no templating
- Helm renderer: Full Helm chart processing with values and templates

**vs. GoTemplate:**
- YAML renderer: No template processing
- GoTemplate renderer: Go template evaluation with dynamic values

## Questions?

Check:
1. `docs/design.md` - Architecture and design decisions
2. `docs/development.md` - Development workflow
3. `pkg/*_test.go` - Usage examples
4. Parent repository documentation at github.com/k8s-manifest-kit

