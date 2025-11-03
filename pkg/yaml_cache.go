package yaml

import (
	"k8s.io/apimachinery/pkg/util/dump"
)

// YAMLSpec contains the data used to generate cache keys for rendered YAML files.
//
//nolint:revive // Name matches pattern from other renderers (KustomizationSpec, TemplateSpec, ChartSpec)
type YAMLSpec struct {
	Path string
}

// CacheKeyFunc generates a cache key from YAML specification.
type CacheKeyFunc func(YAMLSpec) string

// DefaultCacheKey returns a CacheKeyFunc that uses reflection-based hashing of the
// YAML specification fields. For YAML renderer, this is functionally equivalent to
// FastCacheKey since YAML files are static (no dynamic values).
//
// Security Considerations:
// Cache keys are generated from the file path pattern. Unlike templated renderers,
// YAML files are static and don't contain dynamic values, so there are minimal security
// concerns with cache keys. The path pattern itself should not contain sensitive information.
//
// Example:
//
//	renderer := yaml.New(sources, yaml.WithCacheKeyFunc(yaml.DefaultCacheKey()))
func DefaultCacheKey() CacheKeyFunc {
	return func(spec YAMLSpec) string {
		return dump.ForHash(spec)
	}
}

// FastCacheKey returns a CacheKeyFunc that generates keys based only on the path pattern.
// For the YAML renderer, this is the recommended approach since YAML files are static and
// have no dynamic values to consider.
//
// This function is provided for API consistency with other renderers (kustomize, gotemplate, helm).
func FastCacheKey() CacheKeyFunc {
	return func(spec YAMLSpec) string {
		return spec.Path
	}
}

// PathOnlyCacheKey returns a CacheKeyFunc that generates keys based only on the path pattern.
// This is an alias for FastCacheKey provided for clarity and API consistency with other renderers.
func PathOnlyCacheKey() CacheKeyFunc {
	return func(spec YAMLSpec) string {
		return spec.Path
	}
}
