package yaml_test

import (
	"testing"
	"testing/fstest"

	yaml "github.com/k8s-manifest-kit/renderer-yaml/pkg"

	. "github.com/onsi/gomega"
)

func TestNewEngine(t *testing.T) {

	t.Run("should create engine with YAML renderer", func(t *testing.T) {
		g := NewWithT(t)
		testFS := fstest.MapFS{
			"pod.yaml": &fstest.MapFile{Data: []byte(podYAML)},
		}

		e, err := yaml.NewEngine(yaml.Source{
			FS:   testFS,
			Path: "*.yaml",
		})

		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(e).ShouldNot(BeNil())

		// Verify it can render
		objects, err := e.Render(t.Context())
		g.Expect(err).ShouldNot(HaveOccurred())
		g.Expect(objects).To(HaveLen(1))
	})

	t.Run("should return error for invalid source", func(t *testing.T) {
		g := NewWithT(t)
		e, err := yaml.NewEngine(yaml.Source{
			FS:   nil, // Missing FS
			Path: "*.yaml",
		})

		g.Expect(err).Should(HaveOccurred())
		g.Expect(e).Should(BeNil())
	})

	t.Run("should return error for empty path", func(t *testing.T) {
		g := NewWithT(t)
		testFS := fstest.MapFS{
			"pod.yaml": &fstest.MapFile{Data: []byte(podYAML)},
		}

		e, err := yaml.NewEngine(yaml.Source{
			FS:   testFS,
			Path: "", // Missing path
		})

		g.Expect(err).Should(HaveOccurred())
		g.Expect(e).Should(BeNil())
	})
}
