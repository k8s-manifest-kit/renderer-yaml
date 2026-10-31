package yaml_test

import (
	"testing"
	"testing/fstest"

	"github.com/k8s-manifest-kit/engine/pkg/filter/meta/gvk"
	"github.com/k8s-manifest-kit/engine/pkg/transformer/meta/labels"
	jqmatcher "github.com/lburgazzoli/gomega-matchers/pkg/matchers/jq"

	corev1 "k8s.io/api/core/v1"

	yaml "github.com/k8s-manifest-kit/renderer-yaml/pkg"

	. "github.com/onsi/gomega"
)

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
