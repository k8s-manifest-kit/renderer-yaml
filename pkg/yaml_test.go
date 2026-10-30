package yaml_test

import (
	"testing"
	"testing/fstest"

	"github.com/k8s-manifest-kit/engine/pkg/filter/meta/gvk"
	"github.com/k8s-manifest-kit/engine/pkg/transformer/meta/labels"
	"github.com/k8s-manifest-kit/engine/pkg/types"
	"github.com/k8s-manifest-kit/pkg/util/cache"
	jqmatcher "github.com/lburgazzoli/gomega-matchers/pkg/matchers/jq"

	corev1 "k8s.io/api/core/v1"

	yaml "github.com/k8s-manifest-kit/renderer-yaml/pkg"

	. "github.com/onsi/gomega"
)

const podYAML = `
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  labels:
    app: test-app
    component: frontend
spec:
  containers:
  - name: nginx
    image: nginx:latest
`

const configMapYAML = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config
  labels:
    app: test-app
    component: backend
data:
  config.yaml: "port: 8080"
`

const multiDocYAML = `
apiVersion: v1
kind: Service
metadata:
  name: test-service
spec:
  ports:
  - port: 80
---
apiVersion: v1
kind: Secret
metadata:
  name: test-secret
type: Opaque
data:
  password: cGFzc3dvcmQ=
`

func TestRenderer(t *testing.T) {
	ctx := t.Context()

	t.Run("should load single YAML file", func(t *testing.T) {
		g := NewWithT(t)
		testFS := fstest.MapFS{
			"pod.yaml": &fstest.MapFile{Data: []byte(podYAML)},
		}

		renderer, err := yaml.New([]yaml.Source{
			{FS: testFS, Path: "pod.yaml"},
		})
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := renderer.Process(ctx, nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(1))
		g.Expect(objects[0].Object).To(And(
			jqmatcher.Match(`.kind == "Pod"`),
			jqmatcher.Match(`.metadata.name == "test-pod"`),
			jqmatcher.Match(`.metadata.labels["app"] == "test-app"`),
		))
	})

	t.Run("should load multiple YAML files with glob", func(t *testing.T) {
		g := NewWithT(t)
		testFS := fstest.MapFS{
			"pod.yaml":       &fstest.MapFile{Data: []byte(podYAML)},
			"configmap.yaml": &fstest.MapFile{Data: []byte(configMapYAML)},
		}

		renderer, err := yaml.New([]yaml.Source{
			{FS: testFS, Path: "*.yaml"},
		})
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := renderer.Process(ctx, nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(2))
	})

	t.Run("should load multi-document YAML", func(t *testing.T) {
		g := NewWithT(t)
		testFS := fstest.MapFS{
			"resources.yaml": &fstest.MapFile{Data: []byte(multiDocYAML)},
		}

		renderer, err := yaml.New([]yaml.Source{
			{FS: testFS, Path: "resources.yaml"},
		})
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := renderer.Process(ctx, nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(2))
		g.Expect(objects[0].GetKind()).To(Equal("Service"))
		g.Expect(objects[1].GetKind()).To(Equal("Secret"))
	})

	t.Run("should apply filters", func(t *testing.T) {
		g := NewWithT(t)
		testFS := fstest.MapFS{
			"pod.yaml":       &fstest.MapFile{Data: []byte(podYAML)},
			"configmap.yaml": &fstest.MapFile{Data: []byte(configMapYAML)},
		}

		renderer, err := yaml.New(
			[]yaml.Source{{FS: testFS, Path: "*.yaml"}},
			yaml.WithFilter(gvk.Filter(corev1.SchemeGroupVersion.WithKind("Pod"))),
		)
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := renderer.Process(ctx, nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(1))
		g.Expect(objects[0].GetKind()).To(Equal("Pod"))
	})

	t.Run("should apply transformers", func(t *testing.T) {
		g := NewWithT(t)
		testFS := fstest.MapFS{
			"pod.yaml": &fstest.MapFile{Data: []byte(podYAML)},
		}

		renderer, err := yaml.New(
			[]yaml.Source{{FS: testFS, Path: "*.yaml"}},
			yaml.WithTransformer(labels.Set(map[string]string{
				"managed-by": "yaml-renderer",
				"env":        "test",
			})),
		)
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := renderer.Process(ctx, nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(1))
		g.Expect(objects[0].Object).To(And(
			jqmatcher.Match(`.metadata.labels["managed-by"] == "yaml-renderer"`),
			jqmatcher.Match(`.metadata.labels["env"] == "test"`),
			jqmatcher.Match(`.metadata.labels["app"] == "test-app"`),
		))
	})

	t.Run("should handle .yml extension", func(t *testing.T) {
		g := NewWithT(t)
		testFS := fstest.MapFS{
			"pod.yml": &fstest.MapFile{Data: []byte(podYAML)},
		}

		renderer, err := yaml.New([]yaml.Source{
			{FS: testFS, Path: "pod.yml"},
		})
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := renderer.Process(ctx, nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(1))
	})

	t.Run("should return error for non-existent pattern", func(t *testing.T) {
		g := NewWithT(t)
		testFS := fstest.MapFS{
			"pod.yaml": &fstest.MapFile{Data: []byte(podYAML)},
		}

		renderer, err := yaml.New([]yaml.Source{
			{FS: testFS, Path: "nonexistent.yaml"},
		})
		g.Expect(err).ToNot(HaveOccurred())

		_, err = renderer.Process(ctx, nil)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("no files matched pattern"))
	})

	t.Run("should process multiple inputs", func(t *testing.T) {
		g := NewWithT(t)
		testFS1 := fstest.MapFS{
			"pod.yaml": &fstest.MapFile{Data: []byte(podYAML)},
		}
		testFS2 := fstest.MapFS{
			"configmap.yaml": &fstest.MapFile{Data: []byte(configMapYAML)},
		}

		renderer, err := yaml.New([]yaml.Source{
			{FS: testFS1, Path: "*.yaml"},
			{FS: testFS2, Path: "*.yaml"},
		})
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := renderer.Process(ctx, nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(2))
	})
}

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

func BenchmarkYamlRenderWithoutCache(b *testing.B) {
	testFS := fstest.MapFS{
		"pod.yaml":       &fstest.MapFile{Data: []byte(podYAML)},
		"configmap.yaml": &fstest.MapFile{Data: []byte(configMapYAML)},
		"multi.yaml":     &fstest.MapFile{Data: []byte(multiDocYAML)},
	}

	renderer, err := yaml.New([]yaml.Source{
		{FS: testFS, Path: "*.yaml"},
	})
	if err != nil {
		b.Fatalf("failed to create renderer: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_, err := renderer.Process(b.Context(), nil)
		if err != nil {
			b.Fatalf("failed to render: %v", err)
		}
	}
}

func BenchmarkYamlRenderWithCache(b *testing.B) {
	testFS := fstest.MapFS{
		"pod.yaml":       &fstest.MapFile{Data: []byte(podYAML)},
		"configmap.yaml": &fstest.MapFile{Data: []byte(configMapYAML)},
		"multi.yaml":     &fstest.MapFile{Data: []byte(multiDocYAML)},
	}

	renderer, err := yaml.New(
		[]yaml.Source{
			{FS: testFS, Path: "*.yaml"},
		},
		yaml.WithCache(),
	)
	if err != nil {
		b.Fatalf("failed to create renderer: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_, err := renderer.Process(b.Context(), nil)
		if err != nil {
			b.Fatalf("failed to render: %v", err)
		}
	}
}

func BenchmarkYamlRenderCacheMiss(b *testing.B) {
	testFS := fstest.MapFS{
		"pod.yaml":       &fstest.MapFile{Data: []byte(podYAML)},
		"configmap.yaml": &fstest.MapFile{Data: []byte(configMapYAML)},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		// Create new renderer each time to ensure cache miss
		renderer, err := yaml.New(
			[]yaml.Source{
				{FS: testFS, Path: "*.yaml"},
			},
			yaml.WithCache(),
		)
		if err != nil {
			b.Fatalf("failed to create renderer: %v", err)
		}

		_, err = renderer.Process(b.Context(), nil)
		if err != nil {
			b.Fatalf("failed to render: %v", err)
		}
	}
}

func TestSourceAnnotations(t *testing.T) {

	t.Run("should add source annotations when enabled", func(t *testing.T) {
		g := NewWithT(t)
		testFS := fstest.MapFS{
			"manifests/pod.yaml":       &fstest.MapFile{Data: []byte(podYAML)},
			"manifests/configmap.yaml": &fstest.MapFile{Data: []byte(configMapYAML)},
		}

		renderer, err := yaml.New(
			[]yaml.Source{
				{FS: testFS, Path: "manifests/*.yaml"},
			},
			yaml.WithSourceAnnotations(true),
		)
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := renderer.Process(t.Context(), nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(2))

		// Verify all objects have source annotations
		for _, obj := range objects {
			annotations := obj.GetAnnotations()
			g.Expect(annotations).Should(HaveKeyWithValue(types.AnnotationSourceType, "yaml"))
			g.Expect(annotations).Should(HaveKeyWithValue(
				types.AnnotationRenderOrigin,
				types.RenderOriginLive,
			))
			g.Expect(annotations).Should(HaveKey(types.AnnotationSourceFile))
			// File should be one of the yaml files
			g.Expect(annotations[types.AnnotationSourceFile]).Should(
				Or(
					Equal("manifests/pod.yaml"),
					Equal("manifests/configmap.yaml"),
				),
			)
			// YAML renderer should not have path annotation (only file)
			g.Expect(annotations).ShouldNot(HaveKey(types.AnnotationSourcePath))
		}
	})

	t.Run("should not add source annotations when disabled", func(t *testing.T) {
		g := NewWithT(t)
		testFS := fstest.MapFS{
			"pod.yaml": &fstest.MapFile{Data: []byte(podYAML)},
		}

		renderer, err := yaml.New([]yaml.Source{
			{FS: testFS, Path: "*.yaml"},
		})
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := renderer.Process(t.Context(), nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).ToNot(BeEmpty())

		// Verify no source annotations are present
		for _, obj := range objects {
			annotations := obj.GetAnnotations()
			g.Expect(annotations).ShouldNot(HaveKey(types.AnnotationSourceType))
			g.Expect(annotations).ShouldNot(HaveKey(types.AnnotationSourcePath))
			g.Expect(annotations).ShouldNot(HaveKey(types.AnnotationSourceFile))
		}
	})
}

func TestContentHash(t *testing.T) {

	t.Run("should add content hash annotation by default", func(t *testing.T) {
		g := NewWithT(t)
		testFS := fstest.MapFS{
			"manifests/pod.yaml":       &fstest.MapFile{Data: []byte(podYAML)},
			"manifests/configmap.yaml": &fstest.MapFile{Data: []byte(configMapYAML)},
		}

		renderer, err := yaml.New([]yaml.Source{
			{FS: testFS, Path: "manifests/*.yaml"},
		})
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := renderer.Process(t.Context(), nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(2))

		for _, obj := range objects {
			annotations := obj.GetAnnotations()
			g.Expect(annotations).Should(HaveKey(types.AnnotationContentHash))
			g.Expect(annotations[types.AnnotationContentHash]).Should(MatchRegexp("^sha256:[0-9a-f]{64}$"))
		}
	})

	t.Run("should not add content hash when disabled", func(t *testing.T) {
		g := NewWithT(t)
		testFS := fstest.MapFS{
			"pod.yaml": &fstest.MapFile{Data: []byte(podYAML)},
		}

		renderer, err := yaml.New(
			[]yaml.Source{{FS: testFS, Path: "*.yaml"}},
			yaml.WithContentHash(false),
		)
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := renderer.Process(t.Context(), nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).ToNot(BeEmpty())

		for _, obj := range objects {
			annotations := obj.GetAnnotations()
			g.Expect(annotations).ShouldNot(HaveKey(types.AnnotationContentHash))
		}
	})

	t.Run("different objects should have different hashes", func(t *testing.T) {
		g := NewWithT(t)
		testFS := fstest.MapFS{
			"pod.yaml":       &fstest.MapFile{Data: []byte(podYAML)},
			"configmap.yaml": &fstest.MapFile{Data: []byte(configMapYAML)},
		}

		renderer, err := yaml.New([]yaml.Source{
			{FS: testFS, Path: "*.yaml"},
		})
		g.Expect(err).ToNot(HaveOccurred())

		objects, err := renderer.Process(t.Context(), nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(2))

		hash0 := objects[0].GetAnnotations()[types.AnnotationContentHash]
		hash1 := objects[1].GetAnnotations()[types.AnnotationContentHash]
		g.Expect(hash0).ShouldNot(Equal(hash1))
	})

	t.Run("cached results should preserve content hash", func(t *testing.T) {
		g := NewWithT(t)
		testFS := fstest.MapFS{
			"pod.yaml":       &fstest.MapFile{Data: []byte(podYAML)},
			"configmap.yaml": &fstest.MapFile{Data: []byte(configMapYAML)},
		}

		renderer, err := yaml.New(
			[]yaml.Source{{FS: testFS, Path: "*.yaml"}},
			yaml.WithCache(),
		)
		g.Expect(err).ToNot(HaveOccurred())

		result1, err := renderer.Process(t.Context(), nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result1).To(HaveLen(2))

		result2, err := renderer.Process(t.Context(), nil)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result2).To(HaveLen(2))

		for i := range result1 {
			hash1 := result1[i].GetAnnotations()[types.AnnotationContentHash]
			hash2 := result2[i].GetAnnotations()[types.AnnotationContentHash]
			g.Expect(hash1).ShouldNot(BeEmpty())
			g.Expect(hash1).Should(Equal(hash2))
		}
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
