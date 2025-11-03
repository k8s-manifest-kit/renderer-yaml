package yaml

import (
	"github.com/k8s-manifest-kit/engine/pkg/types"
	"github.com/k8s-manifest-kit/pkg/util"
	"github.com/k8s-manifest-kit/pkg/util/cache"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// RendererOption is a generic option for RendererOptions.
type RendererOption = util.Option[RendererOptions]

// RendererOptions is a struct-based option that can set multiple renderer options at once.
type RendererOptions struct {
	// Filters are renderer-specific filters applied during Process().
	Filters []types.Filter

	// Transformers are post-processing transformers applied after YAML rendering.
	Transformers []types.Transformer

	// Cache is a custom cache implementation for render results.
	Cache cache.Interface[[]unstructured.Unstructured]

	// SourceAnnotations enables automatic addition of source tracking annotations.
	SourceAnnotations bool

	// CacheKeyFunc customizes how cache keys are generated from YAML specifications.
	// If nil, DefaultCacheKey is used.
	CacheKeyFunc CacheKeyFunc
}

// ApplyTo applies the renderer options to the target configuration.
func (opts RendererOptions) ApplyTo(target *RendererOptions) {
	target.Filters = opts.Filters
	target.Transformers = opts.Transformers
	target.SourceAnnotations = opts.SourceAnnotations

	if opts.Cache != nil {
		target.Cache = opts.Cache
	}

	if opts.CacheKeyFunc != nil {
		target.CacheKeyFunc = opts.CacheKeyFunc
	}
}

// WithFilter adds a renderer-specific filter to this YAML renderer's processing chain.
// Renderer-specific filters are applied during Process(), before results are returned to the engine.
// For engine-level filtering applied to all renderers, use engine.WithFilter.
func WithFilter(filter types.Filter) RendererOption {
	return util.FunctionalOption[RendererOptions](func(opts *RendererOptions) {
		opts.Filters = append(opts.Filters, filter)
	})
}

// WithTransformer adds a renderer-specific transformer to this YAML renderer's processing chain.
// Renderer-specific transformers are applied during Process(), before results are returned to the engine.
// For engine-level transformation applied to all renderers, use engine.WithTransformer.
func WithTransformer(transformer types.Transformer) RendererOption {
	return util.FunctionalOption[RendererOptions](func(opts *RendererOptions) {
		opts.Transformers = append(opts.Transformers, transformer)
	})
}

// WithCache enables render result caching with the specified options.
// If no options are provided, uses default TTL of 5 minutes.
// By default, caching is NOT enabled.
func WithCache(opts ...cache.Option) RendererOption {
	return util.FunctionalOption[RendererOptions](func(rendererOpts *RendererOptions) {
		rendererOpts.Cache = cache.NewRenderCache(opts...)
	})
}

// WithSourceAnnotations enables or disables automatic addition of source tracking annotations.
// When enabled, the renderer adds metadata annotations to track the source type and file path.
// Annotations added: k8s-manifest-kit.io/source.type, source.file.
// Default: false (disabled).
func WithSourceAnnotations(enabled bool) RendererOption {
	return util.FunctionalOption[RendererOptions](func(opts *RendererOptions) {
		opts.SourceAnnotations = enabled
	})
}

// WithCacheKeyFunc sets a custom cache key generation function.
// Built-in options: DefaultCacheKey (default), FastCacheKey, PathOnlyCacheKey.
//
// For YAML renderer, DefaultCacheKey and FastCacheKey are functionally equivalent
// since YAML files are static (no dynamic values). FastCacheKey is recommended for simplicity.
//
// Example:
//
//	yaml.WithCacheKeyFunc(yaml.FastCacheKey())
func WithCacheKeyFunc(fn CacheKeyFunc) RendererOption {
	return util.FunctionalOption[RendererOptions](func(opts *RendererOptions) {
		opts.CacheKeyFunc = fn
	})
}
