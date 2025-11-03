package yaml

import (
	"fmt"
	"strings"

	"github.com/k8s-manifest-kit/pkg/util/errors"
)

// sourceHolder wraps a Source with internal state for consistency with other renderers.
type sourceHolder struct {
	Source
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
