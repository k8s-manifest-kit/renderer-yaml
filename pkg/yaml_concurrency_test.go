package yaml_test

import (
	"context"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/k8s-manifest-kit/engine/pkg/transformer/meta/labels"
	"github.com/k8s-manifest-kit/engine/pkg/types"
	"github.com/k8s-manifest-kit/pkg/util/cache"
	jqmatcher "github.com/lburgazzoli/gomega-matchers/pkg/matchers/jq"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	yaml "github.com/k8s-manifest-kit/renderer-yaml/pkg"

	. "github.com/onsi/gomega"
)

// concurrentRenders is the number of overlapping Process calls issued against
// one shared renderer during each concurrent phase.
const concurrentRenders = 8

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

// constantCacheKey is a concurrency-safe custom key function returning the
// same key for every source. Per-source cache instances must still keep the
// sources' outputs separated even with this degenerate key.
func constantCacheKey(_ any) string {
	return "constant-key"
}

// newConcurrentSources returns two sources sharing the same path pattern on
// separate immutable in-memory filesystems, each with its own post-renderer
// so cached outputs must be re-owned and re-processed per source.
func newConcurrentSources() []yaml.Source {
	return []yaml.Source{
		{
			FS:   fstest.MapFS{"manifests/pod.yaml": &fstest.MapFile{Data: []byte(podYAML)}},
			Path: "manifests/*.yaml",
			PostRenderers: []types.PostRenderer{
				types.TransformerAsPostRenderer(labels.Set(map[string]string{"source": "one"})),
			},
		},
		{
			FS:   fstest.MapFS{"manifests/configmap.yaml": &fstest.MapFile{Data: []byte(configMapYAML)}},
			Path: "manifests/*.yaml",
			PostRenderers: []types.PostRenderer{
				types.TransformerAsPostRenderer(labels.Set(map[string]string{"source": "two"})),
			},
		},
	}
}

// runConcurrentScenario exercises one shared renderer with barrier-started
// overlapping renders: a cold phase on freshly created state, a nested
// mutation of one cold result after the workers join, and a warm phase whose
// results must stay independent of that mutation. warmOrigin is the render
// origin every warm render must report: cache hits when caching is enabled,
// live loads otherwise.
func runConcurrentScenario(
	ctx context.Context,
	g *WithT,
	renderer *yaml.Renderer,
	warmOrigin string,
) {
	// Cold phase: caches start empty, so workers race between live loads and
	// hits on freshly populated per-source entries.
	coldResults, coldErrs := renderConcurrently(ctx, renderer, concurrentRenders)
	for id := range coldResults {
		g.Expect(coldErrs[id]).ToNot(HaveOccurred())
		expectRenderedObjects(g, coldResults[id], types.RenderOriginLive, types.RenderOriginCache)
	}
	expectLiveLoadPerSource(g, coldResults)

	// Mutate one worker's nested content after the workers joined: neither
	// another worker's cold result...
	mutateNestedSpec(coldResults[0])
	expectUnmutatedPodSpec(g, coldResults[1][0])

	// ...nor any later warm result may observe the mutation.
	warmResults, warmErrs := renderConcurrently(ctx, renderer, concurrentRenders)
	for id := range warmResults {
		g.Expect(warmErrs[id]).ToNot(HaveOccurred())
		expectRenderedObjects(g, warmResults[id], warmOrigin)
	}
}

// renderConcurrently issues count overlapping Process calls against the shared
// renderer, started together by a barrier, and returns each worker's result
// and error indexed by worker id. Workers only write to their own slots; all
// assertions happen in the test goroutine after the workers join.
func renderConcurrently(
	ctx context.Context,
	renderer *yaml.Renderer,
	count int,
) ([][]unstructured.Unstructured, []error) {
	results := make([][]unstructured.Unstructured, count)
	errs := make([]error, count)

	start := make(chan struct{})
	var done sync.WaitGroup

	for id := range count {
		done.Go(func() {
			<-start

			objects, err := renderer.Process(ctx, nil)
			results[id] = objects
			errs[id] = err
		})
	}

	close(start)
	done.Wait()

	return results, errs
}

// expectRenderedObjects asserts that one render returned exactly the two
// source-isolated objects — the pod from source one and the config map from
// source two — with their per-source post-renderer labels, source
// annotations, content hashes, and an allowed render origin.
func expectRenderedObjects(
	g *WithT,
	objects []unstructured.Unstructured,
	allowedOrigins ...string,
) {
	g.Expect(objects).To(HaveLen(2))

	pod := objects[0]
	g.Expect(pod.GetKind()).To(Equal("Pod"))
	g.Expect(pod.GetName()).To(Equal("test-pod"))
	g.Expect(pod.GetLabels()).To(HaveKeyWithValue("source", "one"))
	expectUnmutatedPodSpec(g, pod)
	expectSourceAnnotations(g, pod, "manifests/pod.yaml", allowedOrigins...)

	configMap := objects[1]
	g.Expect(configMap.GetKind()).To(Equal("ConfigMap"))
	g.Expect(configMap.GetName()).To(Equal("test-config"))
	g.Expect(configMap.GetLabels()).To(HaveKeyWithValue("source", "two"))
	g.Expect(configMap.Object).To(jqmatcher.Match(`.data["config.yaml"] == "port: 8080"`))
	expectSourceAnnotations(g, configMap, "manifests/configmap.yaml", allowedOrigins...)
}

// expectSourceAnnotations asserts the renderer type, source file, content
// hash, and an allowed render origin on one rendered object.
func expectSourceAnnotations(
	g *WithT,
	object unstructured.Unstructured,
	sourceFile string,
	allowedOrigins ...string,
) {
	annotations := object.GetAnnotations()

	g.Expect(annotations).To(HaveKeyWithValue(types.AnnotationSourceType, "yaml"))
	g.Expect(annotations).To(HaveKeyWithValue(types.AnnotationSourceFile, sourceFile))
	g.Expect(annotations).To(HaveKey(types.AnnotationContentHash))

	// The render origin must be present and one of the allowed origins.
	g.Expect(allowedOrigins).To(ContainElement(annotations[types.AnnotationRenderOrigin]))
}

// expectLiveLoadPerSource asserts that the cold phase exercised the live path
// for both sources: caches start empty, so at least one worker must have
// loaded each source live even though other racing workers may already have
// hit the freshly populated per-source entries.
func expectLiveLoadPerSource(g *WithT, results [][]unstructured.Unstructured) {
	liveSources := make(map[string]bool)

	for _, objects := range results {
		for i := range objects {
			annotations := objects[i].GetAnnotations()
			if annotations[types.AnnotationRenderOrigin] == types.RenderOriginLive {
				liveSources[annotations[types.AnnotationSourceFile]] = true
			}
		}
	}

	g.Expect(liveSources).To(HaveKey("manifests/pod.yaml"))
	g.Expect(liveSources).To(HaveKey("manifests/configmap.yaml"))
}

// mutateNestedSpec replaces the nested spec of the first object of one
// worker's result to verify that later renders stay independent of caller
// mutations.
func mutateNestedSpec(objects []unstructured.Unstructured) {
	objects[0].Object["spec"] = map[string]any{"mutated": true}
}

// expectUnmutatedPodSpec asserts that a pod still carries its original nested
// spec, untouched by any mutation applied to another render's result.
func expectUnmutatedPodSpec(g *WithT, pod unstructured.Unstructured) {
	g.Expect(pod.Object).To(And(
		jqmatcher.Match(`.spec.containers[0].name == "nginx"`),
		jqmatcher.Match(`.spec.containers[0].image == "nginx:latest"`),
	))
}
