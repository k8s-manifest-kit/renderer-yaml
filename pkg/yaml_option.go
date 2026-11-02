package yaml

import (
	"github.com/k8s-manifest-kit/engine/pkg/types"
	"github.com/k8s-manifest-kit/pkg/util/cache"
	"github.com/k8s-manifest-kit/pkg/util/option"
)

// RendererOption is a generic option for RendererOptions.
type RendererOption = option.Option[RendererOptions]

// RendererOptions is a struct-based option that can set multiple renderer options at once.
//
// Options apply left to right: each option sees the state left by the ones
// before it. When RendererOptions is applied through ApplyTo, fields divide
// into replacing and preserving groups:
//
//   - Filters, Transformers, PostRenderers, and SourceSelectors replace the
//     target's slices, including when nil — a struct clears hooks and
//     selectors that earlier options added.
//   - SourceAnnotations and ContentHash replace the target's booleans,
//     including when false.
//   - A nil CacheOptions preserves the target's cache configuration; a
//     non-nil CacheOptions applies its own field rules to the target's cache
//     options, creating them if needed.
//
// Functional options change only their own field or append to a single
// slice, so they compose with anything applied before or after. Note the
// difference from a fresh constructor: New starts from defaults with content
// hashing enabled and caching disabled (nil cache options), while applying a
// RendererOptions struct replaces the hook, selector, and boolean fields
// outright — a caller who wants content hashing after a struct must set
// ContentHash: true in the struct (or follow it with WithContentHash(true)).
type RendererOptions struct {
	// Filters are renderer-specific filters applied during Process().
	Filters []types.Filter

	// Transformers are post-processing transformers applied after YAML rendering.
	Transformers []types.Transformer

	// PostRenderers are renderer-specific post-renderers applied during Process().
	PostRenderers []types.PostRenderer

	// SourceSelectors are renderer-specific source selectors evaluated before rendering each source.
	SourceSelectors []SourceSelector

	// CacheOptions holds cache configuration. nil = caching disabled.
	// Applying a RendererOptions struct with a nil CacheOptions preserves
	// the previously configured cache (including disabled); a non-nil value
	// applies its field rules on top of the previous configuration.
	CacheOptions *cache.Options

	// SourceAnnotations enables automatic addition of source tracking annotations.
	// Applying a RendererOptions struct replaces this field, including with false.
	SourceAnnotations bool

	// ContentHash enables automatic addition of a SHA-256 content hash annotation.
	// Default: true (enabled) in a fresh constructor; applying a
	// RendererOptions struct replaces this field, including with false, so a
	// struct that wants hashing must set it to true explicitly.
	ContentHash bool
}

// ApplyTo applies the renderer options to the target configuration.
//
// Application follows the composition rules documented on RendererOptions:
// hooks, selectors, and booleans replace their target fields (including
// nil and false), and a nil CacheOptions preserves the target's cache
// configuration while a non-nil one applies its field rules on top.
func (opts RendererOptions) ApplyTo(target *RendererOptions) {
	target.Filters = opts.Filters
	target.Transformers = opts.Transformers
	target.PostRenderers = opts.PostRenderers
	target.SourceSelectors = opts.SourceSelectors
	target.SourceAnnotations = opts.SourceAnnotations
	target.ContentHash = opts.ContentHash

	if opts.CacheOptions != nil {
		if target.CacheOptions == nil {
			target.CacheOptions = &cache.Options{}
		}
		opts.CacheOptions.ApplyTo(target.CacheOptions)
	}
}

// WithFilter adds a renderer-specific filter to this YAML renderer's processing chain.
func WithFilter(filter types.Filter) RendererOption {
	return option.FunctionalOption[RendererOptions](func(opts *RendererOptions) {
		opts.Filters = append(opts.Filters, filter)
	})
}

// WithTransformer adds a renderer-specific transformer to this YAML renderer's processing chain.
func WithTransformer(transformer types.Transformer) RendererOption {
	return option.FunctionalOption[RendererOptions](func(opts *RendererOptions) {
		opts.Transformers = append(opts.Transformers, transformer)
	})
}

// WithPostRenderer adds a renderer-specific post-renderer to this YAML renderer's processing chain.
func WithPostRenderer(p types.PostRenderer) RendererOption {
	return option.FunctionalOption[RendererOptions](func(opts *RendererOptions) {
		opts.PostRenderers = append(opts.PostRenderers, p)
	})
}

// WithSourceSelector adds a source selector to this YAML renderer.
// Source selectors are evaluated before rendering each source. If any selector
// returns false, the source is skipped entirely.
func WithSourceSelector(s SourceSelector) RendererOption {
	return option.FunctionalOption[RendererOptions](func(opts *RendererOptions) {
		opts.SourceSelectors = append(opts.SourceSelectors, s)
	})
}

// WithCache enables render result caching with the specified options.
func WithCache(opts ...cache.Option) RendererOption {
	return option.FunctionalOption[RendererOptions](func(rendererOpts *RendererOptions) {
		if rendererOpts.CacheOptions == nil {
			rendererOpts.CacheOptions = &cache.Options{}
		}

		for _, opt := range opts {
			opt.ApplyTo(rendererOpts.CacheOptions)
		}
	})
}

// WithSourceAnnotations enables or disables automatic addition of source tracking annotations.
func WithSourceAnnotations(enabled bool) RendererOption {
	return option.FunctionalOption[RendererOptions](func(opts *RendererOptions) {
		opts.SourceAnnotations = enabled
	})
}

// WithContentHash enables or disables automatic addition of a SHA-256 content hash annotation.
func WithContentHash(enabled bool) RendererOption {
	return option.FunctionalOption[RendererOptions](func(opts *RendererOptions) {
		opts.ContentHash = enabled
	})
}
