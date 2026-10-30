package yaml

import (
	"fmt"
	"strings"

	"github.com/k8s-manifest-kit/pkg/util/cache"
	"github.com/k8s-manifest-kit/pkg/util/errors"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// sourceHolder wraps a Source with internal state for consistency with other renderers.
type sourceHolder struct {
	Source

	// cache is the source-private render cache. Each registered source gets its
	// own instance so sources that share a path (or a custom cache key) never
	// return each other's cached objects.
	cache cache.Interface[[]unstructured.Unstructured]
}

// Validate checks if the Source configuration is valid.
func (h *sourceHolder) Validate() error {
	if h.FS == nil {
		return fmt.Errorf("filesystem is required: %w", errors.ErrFsRequired)
	}
	if len(strings.TrimSpace(h.Path)) == 0 {
		return fmt.Errorf("path is required: %w", errors.ErrPathEmpty)
	}

	return nil
}
