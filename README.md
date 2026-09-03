# YAML Renderer

`renderer-yaml` loads YAML manifests from an `io/fs` filesystem and renders
single- or multi-document YAML files. Paths may be individual files or glob
patterns.

## Installation

```bash
go get github.com/k8s-manifest-kit/renderer-yaml
```

## Quick start

```go
e, err := yaml.NewEngine(yaml.Source{
    FS:   os.DirFS("."),
    Path: "manifests/*.yaml",
})
if err != nil {
    return err
}

objects, err := e.Render(ctx)
```

YAML documents are rendered in sorted path and document order. Only `.yaml`
and `.yml` files are accepted. Source selectors, post-renderers, source
annotations, content hashes, and the shared cache interface are supported;
render values do not modify YAML content.

See [`docs/design.md`](docs/design.md), [`docs/development.md`](docs/development.md),
and [`AGENTS.md`](AGENTS.md).

## License

Apache License 2.0. See [LICENSE](LICENSE).
