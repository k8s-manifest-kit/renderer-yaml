package yaml_test

import (
	"testing"
	"testing/fstest"

	yaml "github.com/k8s-manifest-kit/renderer-yaml/pkg"
)

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
