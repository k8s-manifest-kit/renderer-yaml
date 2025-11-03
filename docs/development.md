# YAML Renderer Development Guide

## Setup

### Prerequisites

- Go 1.24 or later
- Make
- golangci-lint

### Getting Started

```bash
# Clone and navigate
cd /path/to/k8s-manifest-kit/renderer-yaml

# Install dependencies
go mod download

# Run tests
make test

# Run linter
make lint
```

## Project Structure

```
renderer-yaml/
├── pkg/
│   ├── yaml.go              # Main renderer implementation
│   ├── yaml_option.go       # Functional options
│   ├── yaml_support.go      # Helper functions
│   ├── yaml_test.go         # Tests
│   ├── engine.go            # NewEngine convenience
│   └── engine_test.go       # NewEngine tests
├── config/test/
│   └── manifests/           # Test fixtures (sample YAML files)
├── docs/
│   ├── design.md           # Architecture documentation
│   └── development.md      # This file
├── .golangci.yml           # Linter configuration
├── Makefile                # Build automation
├── go.mod                  # Go module definition
└── README.md               # Project overview
```

## Coding Conventions

### Go Style

Follow standard Go conventions plus:
- Each function parameter has its own type declaration
- Use multiline formatting for functions with 3+ parameters
- Prefer explicit types when they aid readability

### Error Handling

- Return errors as the last parameter
- Use `fmt.Errorf` with `%w` verb for wrapping
- Handle errors at appropriate abstraction level
- Provide clear, actionable error messages

Example:
```go
func (r *Renderer) loadYAMLFile(fsys fs.FS, path string) ([]unstructured.Unstructured, error) {
    info, err := fs.Stat(fsys, path)
    if err != nil {
        return nil, fmt.Errorf("failed to stat %s: %w", path, err)
    }
    // ...
}
```

### Testing with Gomega

Use vanilla Gomega assertions:

```go
import . "github.com/onsi/gomega"

func TestExample(t *testing.T) {
    g := NewWithT(t)
    result, err := someFunction()
    
    g.Expect(err).ShouldNot(HaveOccurred())
    g.Expect(result).Should(HaveLen(3))
}
```

### Documentation

- Comments explain *why*, not *what*
- Focus on non-obvious behavior and edge cases
- Skip boilerplate docstrings unless they add value
- Document public APIs thoroughly

## Development Workflow

### Making Changes

1. **Write Tests First**: Add test cases for new functionality
2. **Implement**: Make minimal changes to fulfill requirements
3. **Run Tests**: `make test`
4. **Run Linter**: `make lint`
5. **Fix Issues**: Address any linter warnings

### Adding New Features

#### Adding a Renderer Option

1. Add field to `RendererOptions` struct in `yaml_option.go`
2. Create `WithXxx()` function
3. Add test coverage
4. Update documentation

Example:
```go
// In yaml_option.go
type RendererOptions struct {
    // ... existing fields ...
    NewFeature bool
}

func WithNewFeature(enabled bool) RendererOption {
    return option.FunctionalOption[RendererOptions](func(opts *RendererOptions) {
        opts.NewFeature = enabled
    })
}
```

#### Adding a Source Option

1. Add field to `Source` struct in `yaml.go`
2. Update `Validate()` in `yaml_support.go` if needed
3. Handle in renderer processing logic
4. Add test coverage

#### Modifying File Loading

File loading logic is in `yaml.go`:
- `renderSingle()`: Handles glob matching and caching
- `loadYAMLFile()`: Loads and parses individual files

Cache key is the glob pattern (Source.Path).

## Testing

### Running Tests

```bash
# Run all tests
make test

# Run with verbose output
go test -v ./...

# Run specific test
go test -v ./pkg -run TestRenderer

# Run specific sub-test
go test -v ./pkg -run "TestRenderer/should_load_single"

# Run benchmarks
make bench
```

### Test Structure

Tests use `testing/fstest.MapFS` for in-memory filesystems:

```go
func TestExample(t *testing.T) {
    g := NewWithT(t)
    testFS := fstest.MapFS{
        "pod.yaml": &fstest.MapFile{Data: []byte(podYAML)},
    }
    
    renderer, err := yaml.New([]yaml.Source{
        {FS: testFS, Path: "*.yaml"},
    })
    g.Expect(err).ToNot(HaveOccurred())
    
    objects, err := renderer.Process(t.Context(), nil)
    g.Expect(err).ToNot(HaveOccurred())
    g.Expect(objects).To(HaveLen(1))
}
```

### Test Coverage

Key test files:
- `yaml_test.go`: Main renderer tests
  - Basic file loading
  - Glob pattern matching
  - Multi-document YAML
  - Filters and transformers
  - Error cases
  - Cache integration
  - Source annotations
  - Benchmarks
- `engine_test.go`: NewEngine convenience function tests

### Writing Good Tests

1. **Test behavior, not implementation**
2. **Use descriptive test names**: `"should load multiple YAML files with glob"`
3. **One assertion focus per test**
4. **Use table-driven tests for similar cases**
5. **Test error paths, not just happy paths**

## Benchmarking

Benchmark key operations:

```bash
# Run all benchmarks
make bench

# Run specific benchmark
go test -bench=BenchmarkYamlRenderWithCache -benchmem ./pkg

# Compare with and without cache
go test -bench=BenchmarkYamlRender -benchmem ./pkg
```

Existing benchmarks:
- `BenchmarkYamlRenderWithoutCache`: Baseline performance
- `BenchmarkYamlRenderWithCache`: Cache hit performance
- `BenchmarkYamlRenderCacheMiss`: Cache initialization overhead

## Linting

The project uses an aggressive linter configuration:

```bash
# Run linter
make lint

# Auto-fix issues
make lint/fix

# Format code
make fmt
```

### Common Linter Issues

1. **Import ordering**: Use `gci` formatter (runs automatically with `make fmt`)
2. **Error wrapping**: Always use `%w` verb
3. **Variable naming**: Avoid single-letter names outside loops
4. **Dot imports**: Only for Gomega test assertions

## Debugging

### Common Issues

1. **Glob pattern not matching**
   - Check filesystem structure
   - Verify pattern syntax for your FS implementation
   - Test with simpler pattern first

2. **YAML parsing errors**
   - Validate YAML syntax externally first
   - Check for multiple documents without `---`
   - Verify Kubernetes resource format

3. **Import path errors**
   - Use `github.com/k8s-manifest-kit/*` paths
   - Run `go mod tidy` after changing imports

4. **Cache not working**
   - Verify cache is enabled with `WithCache()`
   - Check cache key (path must match exactly)
   - Look for cache.Sync() calls

### Useful Debugging Commands

```bash
# Verbose test output
go test -v ./pkg -run TestName

# Print test with race detector
go test -race ./...

# Check module dependencies
go mod graph | grep k8s-manifest-kit

# Verify imports
go list -m all
```

## Release Process

1. Update version in documentation
2. Run full test suite: `make test`
3. Run linter: `make lint`
4. Update CHANGELOG.md
5. Create git tag
6. Push tag to trigger release workflow

## Dependencies

### Core Dependencies
- `k8s.io/apimachinery` - Kubernetes types
- `github.com/k8s-manifest-kit/engine` - Core engine
- `github.com/k8s-manifest-kit/pkg` - Shared utilities

### Test Dependencies
- `github.com/onsi/gomega` - Assertions
- `github.com/lburgazzoli/gomega-matchers` - JQ matchers
- `k8s.io/api` - Kubernetes API types

### Updating Dependencies

```bash
# Update all dependencies
go get -u ./...
go mod tidy

# Update specific dependency
go get github.com/k8s-manifest-kit/engine@latest
go mod tidy
```

## Code Review Guidelines

### Before Submitting

- [ ] Tests pass locally
- [ ] Linter passes
- [ ] Documentation updated
- [ ] Error messages are clear
- [ ] Follows coding conventions

### Review Checklist

- [ ] Code follows established patterns
- [ ] Tests cover new functionality
- [ ] Error handling is appropriate
- [ ] Documentation is clear
- [ ] No unnecessary complexity
- [ ] Thread safety considered
- [ ] Performance implications considered

## Common Patterns

### Creating Test Fixtures

```go
const sampleYAML = `
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
spec:
  containers:
  - name: nginx
    image: nginx:latest
`

testFS := fstest.MapFS{
    "pod.yaml": &fstest.MapFile{Data: []byte(sampleYAML)},
}
```

### Testing Error Cases

```go
t.Run("should return error for empty path", func(t *testing.T) {
    g := NewWithT(t)
    _, err := yaml.New([]yaml.Source{
        {FS: testFS, Path: ""},
    })
    
    g.Expect(err).Should(HaveOccurred())
    g.Expect(err.Error()).To(ContainSubstring("path is required"))
})
```

### Testing with Real Filesystem

```go
func TestWithRealFiles(t *testing.T) {
    g := NewWithT(t)
    
    // Use config/test/manifests directory
    renderer, err := yaml.New([]yaml.Source{
        {FS: os.DirFS("../config/test"), Path: "manifests/*.yaml"},
    })
    g.Expect(err).ToNot(HaveOccurred())
    
    objects, err := renderer.Process(t.Context(), nil)
    g.Expect(err).ToNot(HaveOccurred())
    g.Expect(objects).ToNot(BeEmpty())
}
```

## Resources

- [Go fs.FS documentation](https://pkg.go.dev/io/fs)
- [YAML 1.2 specification](https://yaml.org/spec/1.2/spec.html)
- [Kubernetes API reference](https://kubernetes.io/docs/reference/kubernetes-api/)
- [Gomega documentation](https://onsi.github.io/gomega/)

## Questions?

Check:
1. [CLAUDE.md](../CLAUDE.md) - Quick reference for common tasks
2. [Design documentation](design.md) - Architecture details
3. Test files - Usage examples
4. Parent organization at github.com/k8s-manifest-kit

