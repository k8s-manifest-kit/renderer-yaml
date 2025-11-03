package yaml

import (
	"github.com/k8s-manifest-kit/pkg/util/cache"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// YAMLSpec contains the data used to generate cache keys for rendered YAML files.
//
//nolint:revive // Name matches pattern from other renderers (KustomizationSpec, TemplateSpec, ChartSpec)
type YAMLSpec struct {
	Path string
}

// newCache creates a cache instance with YAML-specific default KeyFunc.
func newCache(opts *cache.Options) cache.Interface[[]unstructured.Unstructured] {
	if opts == nil {
		return nil
	}

	co := *opts

	// Inject path-only KeyFunc as default for YAML
	if co.KeyFunc == nil {
		co.KeyFunc = func(key any) string {
			if spec, ok := key.(YAMLSpec); ok {
				return spec.Path
			}

			return cache.DefaultKeyFunc(key)
		}
	}

	return cache.NewRenderCache(co)
}
