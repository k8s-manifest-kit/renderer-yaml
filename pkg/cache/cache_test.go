package cache_test

import (
	"testing"
	"time"

	utilcache "github.com/k8s-manifest-kit/pkg/util/cache"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/k8s-manifest-kit/renderer-yaml/pkg/cache"

	. "github.com/onsi/gomega"
)

func testObjects(name string) []unstructured.Unstructured {
	return []unstructured.Unstructured{
		{Object: map[string]any{
			"kind": "ConfigMap",
			"metadata": map[string]any{
				"name": name,
			},
		}},
	}
}

func TestNew(t *testing.T) {
	t.Run("should disable caching for nil options", func(t *testing.T) {
		g := NewWithT(t)

		g.Expect(cache.New(nil)).To(BeNil())
	})

	t.Run("should key default entries by the spec path only", func(t *testing.T) {
		g := NewWithT(t)

		c := cache.New(&utilcache.Options{})

		spec := cache.YAMLSpec{Path: "manifests/pod.yaml"}

		c.Set(spec, testObjects("default-key"))

		cached, found := c.Get(spec)
		g.Expect(found).To(BeTrue())
		g.Expect(cached).To(HaveLen(1))
		g.Expect(cached[0].GetName()).To(Equal("default-key"))

		// The default key is the path only: a different path misses.
		other := cache.YAMLSpec{Path: "manifests/configmap.yaml"}
		_, found = c.Get(other)
		g.Expect(found).To(BeFalse())
	})

	t.Run("should fall back to the shared default for non-spec keys", func(t *testing.T) {
		g := NewWithT(t)

		c := cache.New(&utilcache.Options{})

		// A plain string key is not a YAMLSpec; the injected default must
		// delegate to the shared DefaultKeyFunc (which returns string keys
		// verbatim) instead of failing or producing an empty key.
		c.Set("manifests/plain-key", testObjects("string-key"))

		cached, found := c.Get("manifests/plain-key")
		g.Expect(found).To(BeTrue())
		g.Expect(cached[0].GetName()).To(Equal("string-key"))

		// Non-string keys round-trip through the shared fallback as well.
		c.Set(42, testObjects("int-key"))

		cached, found = c.Get(42)
		g.Expect(found).To(BeTrue())
		g.Expect(cached[0].GetName()).To(Equal("int-key"))
	})

	t.Run("should preserve custom key functions", func(t *testing.T) {
		g := NewWithT(t)

		var received []any

		c := cache.New(&utilcache.Options{
			KeyFunc: func(key any) string {
				received = append(received, key)

				return utilcache.DefaultKeyFunc(key)
			},
		})

		spec := cache.YAMLSpec{Path: "manifests/pod.yaml"}

		c.Set(spec, testObjects("custom-key"))

		cached, found := c.Get(spec)
		g.Expect(found).To(BeTrue())
		g.Expect(cached[0].GetName()).To(Equal("custom-key"))

		// Set plus one get: every invocation received the exported spec.
		g.Expect(received).To(HaveLen(2))
		for _, key := range received {
			spec, ok := key.(cache.YAMLSpec)
			g.Expect(ok).To(BeTrue(), "key function should receive a YAMLSpec, got %T", key)
			g.Expect(spec.Path).To(Equal("manifests/pod.yaml"))
		}
	})

	t.Run("should honor the configured TTL", func(t *testing.T) {
		g := NewWithT(t)

		c := cache.New(&utilcache.Options{TTL: 100 * time.Millisecond})

		spec := cache.YAMLSpec{Path: "manifests/pod.yaml"}

		c.Set(spec, testObjects("expiring"))

		cached, found := c.Get(spec)
		g.Expect(found).To(BeTrue())
		g.Expect(cached[0].GetName()).To(Equal("expiring"))

		time.Sleep(150 * time.Millisecond)

		_, found = c.Get(spec)
		g.Expect(found).To(BeFalse())
	})

	t.Run("should not mutate the caller options", func(t *testing.T) {
		g := NewWithT(t)

		opts := &utilcache.Options{TTL: time.Minute}

		cache.New(opts)

		// The path-only default key function injection must not leak into
		// the caller's options, and the TTL must be preserved as configured.
		g.Expect(opts.KeyFunc).To(BeNil())
		g.Expect(opts.TTL).To(Equal(time.Minute))
	})

	t.Run("should not share entries between instances", func(t *testing.T) {
		g := NewWithT(t)

		first := cache.New(&utilcache.Options{})
		second := cache.New(&utilcache.Options{})

		spec := cache.YAMLSpec{Path: "manifests/pod.yaml"}

		first.Set(spec, testObjects("first"))

		_, found := second.Get(spec)
		g.Expect(found).To(BeFalse())

		second.Set(spec, testObjects("second"))

		cached, found := first.Get(spec)
		g.Expect(found).To(BeTrue())
		g.Expect(cached[0].GetName()).To(Equal("first"))

		cached, found = second.Get(spec)
		g.Expect(found).To(BeTrue())
		g.Expect(cached[0].GetName()).To(Equal("second"))
	})

	t.Run("should deep clone stored and returned objects", func(t *testing.T) {
		g := NewWithT(t)

		c := cache.New(&utilcache.Options{})

		spec := cache.YAMLSpec{Path: "manifests/pod.yaml"}

		stored := testObjects("cloned")
		c.Set(spec, stored)

		// Mutating the caller's slice after Set must not affect the cache.
		stored[0].SetName("mutated-stored")

		first, found := c.Get(spec)
		g.Expect(found).To(BeTrue())
		g.Expect(first[0].GetName()).To(Equal("cloned"))

		// Mutating a returned slice must not affect later retrievals.
		first[0].SetName("mutated-returned")

		second, found := c.Get(spec)
		g.Expect(found).To(BeTrue())
		g.Expect(second[0].GetName()).To(Equal("cloned"))
	})
}
