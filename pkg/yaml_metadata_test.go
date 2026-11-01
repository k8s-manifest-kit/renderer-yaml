package yaml_test

import (
	"testing"
	"testing/fstest"

	"github.com/k8s-manifest-kit/engine/pkg/types"
	"github.com/k8s-manifest-kit/pkg/util/k8s"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	yaml "github.com/k8s-manifest-kit/renderer-yaml/pkg"

	. "github.com/onsi/gomega"
)

// annotatedYAML carries its own annotations, including stale kit-managed
// keys, to characterize how renderer annotation phases interact with
// pre-existing metadata.
const annotatedYAML = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: annotated-config
  annotations:
    existing: annotation
    manifests.k8s-manifests-kit/content.hash: sha256:stale
    manifests.k8s-manifests-kit/render.origin: stale
data:
  key: value
`

// renderYAMLSource renders the given in-memory files with the renderer
// options and returns the bare result.
func renderYAMLSource(
	t *testing.T,
	files map[string]string,
	opts ...yaml.RendererOption,
) []unstructured.Unstructured {
	t.Helper()

	g := NewWithT(t)

	testFS := make(fstest.MapFS, len(files))
	for name, data := range files {
		testFS[name] = &fstest.MapFile{Data: []byte(data)}
	}

	renderer, err := yaml.New([]yaml.Source{{FS: testFS, Path: "*.yaml"}}, opts...)
	g.Expect(err).ToNot(HaveOccurred())

	objects, err := renderer.Process(t.Context(), nil)
	g.Expect(err).ToNot(HaveOccurred())

	return objects
}

// TestAnnotationApplication characterizes how the YAML renderer applies
// file provenance, content hashes, and render origins.
func TestAnnotationApplication(t *testing.T) {

	t.Run("exact annotation map with hash over source metadata before origin", func(t *testing.T) {
		g := NewWithT(t)

		files := map[string]string{"pod.yaml": podYAML}

		bare := renderYAMLSource(t, files, yaml.WithContentHash(false))
		annotated := renderYAMLSource(t, files, yaml.WithSourceAnnotations(true))

		g.Expect(bare).To(HaveLen(1))
		g.Expect(annotated).To(HaveLen(1))

		expected := bare[0].DeepCopy()
		expected.SetAnnotations(map[string]string{
			types.AnnotationSourceType: "yaml",
			types.AnnotationSourceFile: "pod.yaml",
		})

		g.Expect(annotated[0].GetAnnotations()).To(Equal(map[string]string{
			types.AnnotationSourceType:   "yaml",
			types.AnnotationSourceFile:   "pod.yaml",
			types.AnnotationContentHash:  k8s.ContentHash(expected),
			types.AnnotationRenderOrigin: types.RenderOriginLive,
		}))
	})

	t.Run("pre-existing annotations are preserved and included in the hash input", func(t *testing.T) {
		g := NewWithT(t)

		files := map[string]string{"annotated.yaml": annotatedYAML}

		bare := renderYAMLSource(t, files, yaml.WithContentHash(false))
		annotated := renderYAMLSource(t, files, yaml.WithSourceAnnotations(true))

		g.Expect(bare).To(HaveLen(1))
		g.Expect(annotated).To(HaveLen(1))

		expected := bare[0].DeepCopy()
		expected.SetAnnotations(map[string]string{
			"existing":                   "annotation",
			types.AnnotationContentHash:  "sha256:stale",
			types.AnnotationRenderOrigin: "stale",
			types.AnnotationSourceType:   "yaml",
			types.AnnotationSourceFile:   "annotated.yaml",
		})

		annotations := annotated[0].GetAnnotations()
		g.Expect(annotations).To(HaveKeyWithValue("existing", "annotation"))
		g.Expect(annotations).To(HaveKeyWithValue(
			types.AnnotationContentHash,
			k8s.ContentHash(expected),
		))
		g.Expect(annotations).NotTo(HaveKeyWithValue(types.AnnotationContentHash, "sha256:stale"))
		g.Expect(annotations).To(HaveKeyWithValue(types.AnnotationRenderOrigin, types.RenderOriginLive))
		g.Expect(annotations).NotTo(HaveKey(types.AnnotationSourcePath))
	})

	t.Run("source annotations without content hash leave no hash key and no path", func(t *testing.T) {
		g := NewWithT(t)

		annotated := renderYAMLSource(t,
			map[string]string{"pod.yaml": podYAML},
			yaml.WithSourceAnnotations(true),
			yaml.WithContentHash(false),
		)

		g.Expect(annotated).To(HaveLen(1))

		annotations := annotated[0].GetAnnotations()
		g.Expect(annotations).To(HaveLen(3))
		g.Expect(annotations).To(HaveKeyWithValue(types.AnnotationSourceType, "yaml"))
		g.Expect(annotations).To(HaveKeyWithValue(types.AnnotationSourceFile, "pod.yaml"))
		g.Expect(annotations).To(HaveKeyWithValue(types.AnnotationRenderOrigin, types.RenderOriginLive))
		g.Expect(annotations).NotTo(HaveKey(types.AnnotationContentHash))
		g.Expect(annotations).NotTo(HaveKey(types.AnnotationSourcePath))
	})

	t.Run("multi-document files annotate every document with the same file", func(t *testing.T) {
		g := NewWithT(t)

		annotated := renderYAMLSource(t,
			map[string]string{"multi.yaml": multiDocYAML},
			yaml.WithSourceAnnotations(true),
		)

		g.Expect(annotated).To(HaveLen(2))

		for i := range annotated {
			annotations := annotated[i].GetAnnotations()
			g.Expect(annotations).To(HaveKeyWithValue(types.AnnotationSourceType, "yaml"))
			g.Expect(annotations).To(HaveKeyWithValue(types.AnnotationSourceFile, "multi.yaml"))
			g.Expect(annotations).To(HaveKeyWithValue(types.AnnotationRenderOrigin, types.RenderOriginLive))
			g.Expect(annotations).To(HaveKey(types.AnnotationContentHash))
			g.Expect(annotations).NotTo(HaveKey(types.AnnotationSourcePath))
		}
	})
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
