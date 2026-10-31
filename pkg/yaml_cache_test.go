package yaml_test

import (
	"testing"
	"testing/fstest"

	"github.com/k8s-manifest-kit/engine/pkg/transformer/meta/labels"
	"github.com/k8s-manifest-kit/engine/pkg/types"
	"github.com/k8s-manifest-kit/pkg/util/cache"

	yaml "github.com/k8s-manifest-kit/renderer-yaml/pkg"

	. "github.com/onsi/gomega"
)

func TestCacheIntegration(t *testing.T) {

	t.Run("should cache identical renders", func(t *testing.T) {
		g := NewWithT(t)
		testFS := fstest.MapFS{
			"pod.yaml":       &fstest.MapFile{Data: []byte(podYAML)},
			"configmap.yaml": &fstest.MapFile{Data: []byte(configMapYAML)},
		}

		renderer, err := yaml.New([]yaml.Source{
			{FS: testFS, Path: "*.yaml"},
		},
			yaml.WithCache(),
			yaml.WithSourceAnnotations(true),
		)
		g.Expect(err).ToNot(HaveOccurred())

		// First render - cache miss
		result1, err := renderer.Process(t.Context(), nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result1).ToNot(BeEmpty())
		for _, obj := range result1 {
			g.Expect(obj.GetAnnotations()).Should(HaveKeyWithValue(
				types.AnnotationRenderOrigin,
				types.RenderOriginLive,
			))
		}

		// Second render - cache hit
		result2, err := renderer.Process(t.Context(), nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result2).To(HaveLen(len(result1)))

		for _, obj := range result2 {
			g.Expect(obj.GetAnnotations()).Should(HaveKeyWithValue(
				types.AnnotationRenderOrigin,
				types.RenderOriginCache,
			))
		}
	})

	t.Run("should miss cache on different paths", func(t *testing.T) {
		g := NewWithT(t)
		testFS := fstest.MapFS{
			"pod.yaml":       &fstest.MapFile{Data: []byte(podYAML)},
			"configmap.yaml": &fstest.MapFile{Data: []byte(configMapYAML)},
		}

		renderer, err := yaml.New([]yaml.Source{
			{FS: testFS, Path: "pod.yaml"},
		},
			yaml.WithCache(),
		)
		g.Expect(err).ToNot(HaveOccurred())

		// First render with pod.yaml
		result1, err := renderer.Process(t.Context(), nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result1).To(HaveLen(1))
		g.Expect(result1[0].GetKind()).To(Equal("Pod"))

		// Create new renderer with different path - cache miss
		renderer2, err := yaml.New([]yaml.Source{
			{FS: testFS, Path: "configmap.yaml"},
		},
			yaml.WithCache(),
		)
		g.Expect(err).ToNot(HaveOccurred())

		result2, err := renderer2.Process(t.Context(), nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result2).To(HaveLen(1))
		g.Expect(result2[0].GetKind()).To(Equal("ConfigMap"))
	})

	t.Run("should work with cache disabled", func(t *testing.T) {
		g := NewWithT(t)
		testFS := fstest.MapFS{
			"pod.yaml": &fstest.MapFile{Data: []byte(podYAML)},
		}

		renderer, err := yaml.New(
			[]yaml.Source{
				{FS: testFS, Path: "*.yaml"},
			},
		)
		g.Expect(err).ToNot(HaveOccurred())

		// First render
		result1, err := renderer.Process(t.Context(), nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result1).ToNot(BeEmpty())

		// Second render - should work even without cache
		result2, err := renderer.Process(t.Context(), nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result2).To(HaveLen(len(result1)))
	})

	t.Run("should return clones from cache", func(t *testing.T) {
		g := NewWithT(t)
		testFS := fstest.MapFS{
			"pod.yaml": &fstest.MapFile{Data: []byte(podYAML)},
		}

		renderer, err := yaml.New([]yaml.Source{
			{FS: testFS, Path: "*.yaml"},
		},
			yaml.WithCache(),
		)
		g.Expect(err).ToNot(HaveOccurred())

		// First render
		result1, err := renderer.Process(t.Context(), nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result1).ToNot(BeEmpty())

		// Modify first result
		if len(result1) > 0 {
			result1[0].SetName("modified-name")
		}

		// Second render - should not be affected by modification
		result2, err := renderer.Process(t.Context(), nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result2).ToNot(BeEmpty())

		if len(result2) > 0 {
			g.Expect(result2[0].GetName()).ToNot(Equal("modified-name"))
		}
	})
}

func TestCacheSourceIsolation(t *testing.T) {

	t.Run("should isolate same-path sources on different filesystems", func(t *testing.T) {
		g := NewWithT(t)
		testFS1 := fstest.MapFS{
			"manifests/pod.yaml": &fstest.MapFile{Data: []byte(podYAML)},
		}
		testFS2 := fstest.MapFS{
			"manifests/pod.yaml": &fstest.MapFile{Data: []byte(configMapYAML)},
		}

		renderer, err := yaml.New(
			[]yaml.Source{
				{FS: testFS1, Path: "manifests/*.yaml"},
				{FS: testFS2, Path: "manifests/*.yaml"},
			},
			yaml.WithCache(),
			yaml.WithSourceAnnotations(true),
		)
		g.Expect(err).ToNot(HaveOccurred())

		// First render: both sources must load their own files, not share a cache entry.
		result1, err := renderer.Process(t.Context(), nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result1).To(HaveLen(2))
		g.Expect(result1[0].GetKind()).To(Equal("Pod"))
		g.Expect(result1[1].GetKind()).To(Equal("ConfigMap"))
		for _, obj := range result1 {
			g.Expect(obj.GetAnnotations()).Should(HaveKeyWithValue(
				types.AnnotationRenderOrigin,
				types.RenderOriginLive,
			))
		}

		// Second render: each source must hit its own cache entry.
		result2, err := renderer.Process(t.Context(), nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result2).To(HaveLen(2))
		g.Expect(result2[0].GetKind()).To(Equal("Pod"))
		g.Expect(result2[1].GetKind()).To(Equal("ConfigMap"))
		for _, obj := range result2 {
			g.Expect(obj.GetAnnotations()).Should(HaveKeyWithValue(
				types.AnnotationRenderOrigin,
				types.RenderOriginCache,
			))
		}
	})

	t.Run("should isolate same-path sources with a constant custom key", func(t *testing.T) {
		g := NewWithT(t)
		testFS1 := fstest.MapFS{
			"manifests/pod.yaml": &fstest.MapFile{Data: []byte(podYAML)},
		}
		testFS2 := fstest.MapFS{
			"manifests/pod.yaml": &fstest.MapFile{Data: []byte(configMapYAML)},
		}

		var keyInputs []yaml.YAMLSpec
		customKeyFunc := func(key any) string {
			if spec, ok := key.(yaml.YAMLSpec); ok {
				keyInputs = append(keyInputs, spec)
			}

			return "constant-key"
		}

		renderer, err := yaml.New(
			[]yaml.Source{
				{FS: testFS1, Path: "manifests/*.yaml"},
				{FS: testFS2, Path: "manifests/*.yaml"},
			},
			yaml.WithCache(cache.WithKeyFunc(customKeyFunc)),
			yaml.WithSourceAnnotations(true),
		)
		g.Expect(err).ToNot(HaveOccurred())

		// First render: both sources load live despite the constant key.
		result1, err := renderer.Process(t.Context(), nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result1).To(HaveLen(2))
		g.Expect(result1[0].GetKind()).To(Equal("Pod"))
		g.Expect(result1[1].GetKind()).To(Equal("ConfigMap"))
		for _, obj := range result1 {
			g.Expect(obj.GetAnnotations()).Should(HaveKeyWithValue(
				types.AnnotationRenderOrigin,
				types.RenderOriginLive,
			))
		}

		// Second render: each source independently transitions to its own cache entry.
		result2, err := renderer.Process(t.Context(), nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result2).To(HaveLen(2))
		g.Expect(result2[0].GetKind()).To(Equal("Pod"))
		g.Expect(result2[1].GetKind()).To(Equal("ConfigMap"))
		for _, obj := range result2 {
			g.Expect(obj.GetAnnotations()).Should(HaveKeyWithValue(
				types.AnnotationRenderOrigin,
				types.RenderOriginCache,
			))
		}

		// The key callback still receives YAMLSpec values carrying the source path.
		g.Expect(keyInputs).ToNot(BeEmpty())
		for _, spec := range keyInputs {
			g.Expect(spec.Path).To(Equal("manifests/*.yaml"))
		}
	})

	t.Run("should keep cached output independent and source-correct on hits", func(t *testing.T) {
		g := NewWithT(t)
		testFS1 := fstest.MapFS{
			"manifests/pod.yaml": &fstest.MapFile{Data: []byte(podYAML)},
		}
		testFS2 := fstest.MapFS{
			"manifests/configmap.yaml": &fstest.MapFile{Data: []byte(configMapYAML)},
		}

		renderer, err := yaml.New(
			[]yaml.Source{
				{
					FS:   testFS1,
					Path: "manifests/*.yaml",
					PostRenderers: []types.PostRenderer{
						types.TransformerAsPostRenderer(labels.Set(map[string]string{"source": "one"})),
					},
				},
				{
					FS:   testFS2,
					Path: "manifests/*.yaml",
					PostRenderers: []types.PostRenderer{
						types.TransformerAsPostRenderer(labels.Set(map[string]string{"source": "two"})),
					},
				},
			},
			yaml.WithCache(),
			yaml.WithSourceAnnotations(true),
		)
		g.Expect(err).ToNot(HaveOccurred())

		result1, err := renderer.Process(t.Context(), nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result1).To(HaveLen(2))

		// Mutate a nested field of the first result; later renders must not see it.
		result1[0].Object["spec"] = map[string]any{"mutated": true}

		result2, err := renderer.Process(t.Context(), nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result2).To(HaveLen(2))

		// The pod comes from source one and the config map from source two, both as cache hits.
		g.Expect(result2[0].GetKind()).To(Equal("Pod"))
		g.Expect(result2[1].GetKind()).To(Equal("ConfigMap"))
		for _, obj := range result2 {
			g.Expect(obj.GetAnnotations()).Should(HaveKeyWithValue(
				types.AnnotationRenderOrigin,
				types.RenderOriginCache,
			))
		}

		// Each cached object keeps its own source file and re-receives its own post-renderer.
		g.Expect(result2[0].GetAnnotations()).Should(HaveKeyWithValue(
			types.AnnotationSourceFile,
			"manifests/pod.yaml",
		))
		g.Expect(result2[0].GetLabels()["source"]).To(Equal("one"))
		g.Expect(result2[1].GetAnnotations()).Should(HaveKeyWithValue(
			types.AnnotationSourceFile,
			"manifests/configmap.yaml",
		))
		g.Expect(result2[1].GetLabels()["source"]).To(Equal("two"))

		// The mutation of result1 did not leak into the cached pod.
		g.Expect(result2[0].Object).ToNot(HaveKey("mutated"))
		g.Expect(result2[0].Object["spec"]).ToNot(HaveKey("mutated"))
	})
}

func TestCacheKeyFunc(t *testing.T) {

	t.Run("should use default cache key function", func(t *testing.T) {
		g := NewWithT(t)
		testFS := fstest.MapFS{
			"pod.yaml": &fstest.MapFile{Data: []byte(podYAML)},
		}

		renderer, err := yaml.New(
			[]yaml.Source{{FS: testFS, Path: "*.yaml"}},
			yaml.WithCache(),
		)
		g.Expect(err).ToNot(HaveOccurred())

		// First render - cache miss
		result1, err := renderer.Process(t.Context(), nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result1).To(HaveLen(1))

		// Second render - cache hit
		result2, err := renderer.Process(t.Context(), nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result2).To(HaveLen(1))

		// Results should be equal
		g.Expect(result2[0]).To(Equal(result1[0]))
	})

	t.Run("should use custom cache key function", func(t *testing.T) {
		g := NewWithT(t)
		testFS := fstest.MapFS{
			"pod.yaml":       &fstest.MapFile{Data: []byte(podYAML)},
			"configmap.yaml": &fstest.MapFile{Data: []byte(configMapYAML)},
		}

		// Custom cache key function that always returns the same key
		customKeyFunc := func(_ any) string {
			return "constant-key"
		}

		renderer, err := yaml.New(
			[]yaml.Source{{FS: testFS, Path: "*.yaml"}},
			yaml.WithCache(cache.WithKeyFunc(customKeyFunc)),
		)
		g.Expect(err).ToNot(HaveOccurred())

		// First render
		result1, err := renderer.Process(t.Context(), nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result1).To(HaveLen(2))

		// Create second renderer with different path but same custom key function
		renderer2, err := yaml.New(
			[]yaml.Source{{FS: testFS, Path: "pod.yaml"}},
			yaml.WithCache(cache.WithKeyFunc(customKeyFunc)),
		)
		g.Expect(err).ToNot(HaveOccurred())

		// This would normally be a cache miss due to different path,
		// but with custom key function returning constant, behavior depends on cache instance
		result2, err := renderer2.Process(t.Context(), nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result2).To(HaveLen(1))
	})

	t.Run("should use custom path-only cache key function", func(t *testing.T) {
		g := NewWithT(t)
		testFS := fstest.MapFS{
			"pod.yaml": &fstest.MapFile{Data: []byte(podYAML)},
		}

		// Path-only cache key function (equivalent to old FastCacheKey)
		pathOnlyKeyFunc := func(key any) string {
			if spec, ok := key.(yaml.YAMLSpec); ok {
				return spec.Path
			}

			return ""
		}

		renderer, err := yaml.New(
			[]yaml.Source{{FS: testFS, Path: "*.yaml"}},
			yaml.WithCache(cache.WithKeyFunc(pathOnlyKeyFunc)),
		)
		g.Expect(err).ToNot(HaveOccurred())

		// First render - cache miss
		result1, err := renderer.Process(t.Context(), nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result1).To(HaveLen(1))

		// Second render - cache hit
		result2, err := renderer.Process(t.Context(), nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result2).To(HaveLen(1))

		// Results should be equal
		g.Expect(result2[0]).To(Equal(result1[0]))
	})

	t.Run("should use default cache behavior with path-based keys", func(t *testing.T) {
		g := NewWithT(t)
		testFS := fstest.MapFS{
			"pod.yaml":       &fstest.MapFile{Data: []byte(podYAML)},
			"configmap.yaml": &fstest.MapFile{Data: []byte(configMapYAML)},
		}

		// Using default key function (which defaults to path-based for YAML)
		renderer, err := yaml.New(
			[]yaml.Source{{FS: testFS, Path: "*.yaml"}},
			yaml.WithCache(),
		)
		g.Expect(err).ToNot(HaveOccurred())

		// First render - cache miss
		result1, err := renderer.Process(t.Context(), nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result1).To(HaveLen(2))

		// Second render - cache hit
		result2, err := renderer.Process(t.Context(), nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result2).To(HaveLen(2))

		// Results should be equal
		for i := range result1 {
			g.Expect(result2[i]).To(Equal(result1[i]))
		}
	})
}
