# renderer-yaml

YAML manifest renderer for the k8s-manifest-kit ecosystem.

Part of the [k8s-manifest-kit](https://github.com/k8s-manifest-kit) organization.

## Overview

The YAML renderer provides programmatic loading and processing of Kubernetes YAML manifest files. It supports glob pattern matching, multi-document YAML files, filtering, transformation, and caching.

## Installation

```bash
go get github.com/k8s-manifest-kit/renderer-yaml
```

## Quick Start

```go
package main

import (
    "context"
    "os"
    
    yaml "github.com/k8s-manifest-kit/renderer-yaml/pkg"
)

func main() {
    // Create engine with YAML renderer
    e, err := yaml.NewEngine(yaml.Source{
        FS:   os.DirFS("/path/to/manifests"),
        Path: "*.yaml",
    })
    if err != nil {
        panic(err)
    }
    
    // Render manifests
    objects, err := e.Render(context.Background())
    if err != nil {
        panic(err)
    }
    
    // Process objects...
}
```

## Features

- **Glob Pattern Matching**: Load multiple files using patterns like `*.yaml` or `manifests/**/*.yml`
- **Multi-Document YAML**: Automatically handles files with multiple YAML documents separated by `---`
- **Filesystem Abstraction**: Works with any `fs.FS` implementation (os.DirFS, embed.FS, testing/fstest)
- **Caching**: Optional TTL-based caching to avoid redundant file reads
- **Filtering & Transformation**: Apply filters and transformers at render time
- **Source Tracking**: Optional annotations to track which file each object came from

## Documentation

- [Design Documentation](docs/design.md) - Architecture and design decisions
- [Development Guide](docs/development.md) - Development workflow and conventions
- [CLAUDE.md](CLAUDE.md) - AI assistant reference guide

## Contributing

Contributions are welcome! Please see our [contributing guidelines](https://github.com/k8s-manifest-kit/docs/blob/main/CONTRIBUTING.md).

## License

Apache License 2.0 - See [LICENSE](LICENSE) for details.
