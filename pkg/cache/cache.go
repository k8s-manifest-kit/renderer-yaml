package cache

import (
	utilcache "github.com/k8s-manifest-kit/pkg/util/cache"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// YAMLSpec contains the data used to generate cache keys for rendered YAML files.
type YAMLSpec struct {
	Path string
}

// New creates a cache instance with YAML-specific default KeyFunc.
func New(opts *utilcache.Options) utilcache.Interface[[]unstructured.Unstructured] {
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

			return utilcache.DefaultKeyFunc(key)
		}
	}

	return utilcache.NewRenderCache(co)
}
