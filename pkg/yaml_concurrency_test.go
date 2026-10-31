package yaml_test

import (
	"testing"

	"github.com/k8s-manifest-kit/engine/pkg/types"
	"github.com/k8s-manifest-kit/pkg/util/cache"

	yaml "github.com/k8s-manifest-kit/renderer-yaml/pkg"

	. "github.com/onsi/gomega"
)

// TestConcurrentRendering verifies that overlapping successful Process calls
// on one shared renderer keep same-path sources isolated and cached outputs
// independently owned under the race detector.
func TestConcurrentRendering(t *testing.T) {

	t.Run("cache disabled", func(t *testing.T) {
		g := NewWithT(t)

		renderer, err := yaml.New(
			newConcurrentSources(),
			yaml.WithSourceAnnotations(true),
		)
		g.Expect(err).ToNot(HaveOccurred())

		runConcurrentScenario(t.Context(), g, renderer, types.RenderOriginLive)
	})

	t.Run("cache enabled with default keys", func(t *testing.T) {
		g := NewWithT(t)

		renderer, err := yaml.New(
			newConcurrentSources(),
			yaml.WithCache(),
			yaml.WithSourceAnnotations(true),
		)
		g.Expect(err).ToNot(HaveOccurred())

		runConcurrentScenario(t.Context(), g, renderer, types.RenderOriginCache)
	})

	t.Run("cache enabled with constant custom keys", func(t *testing.T) {
		g := NewWithT(t)

		renderer, err := yaml.New(
			newConcurrentSources(),
			yaml.WithCache(cache.WithKeyFunc(constantCacheKey)),
			yaml.WithSourceAnnotations(true),
		)
		g.Expect(err).ToNot(HaveOccurred())

		runConcurrentScenario(t.Context(), g, renderer, types.RenderOriginCache)
	})
}
