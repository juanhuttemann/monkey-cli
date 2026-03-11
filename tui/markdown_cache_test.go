package tui

import (
	"testing"
)

func TestMarkdownRenderer_CachePopulatedOnFirstCall(t *testing.T) {
	clearMarkdownCache()
	t.Cleanup(clearMarkdownCache)

	if got := markdownCacheLen(); got != 0 {
		t.Fatalf("expected empty cache before first call, got %d", got)
	}
	RenderMarkdown("hello", 80)
	if got := markdownCacheLen(); got != 1 {
		t.Fatalf("expected cache size 1 after first call, got %d", got)
	}
}

func TestMarkdownRenderer_CacheDoesNotGrowForSameWidth(t *testing.T) {
	clearMarkdownCache()
	t.Cleanup(clearMarkdownCache)

	RenderMarkdown("hello", 80)
	RenderMarkdown("world", 80)
	RenderMarkdown("**bold**", 80)
	if got := markdownCacheLen(); got != 1 {
		t.Fatalf("expected cache size 1 for repeated same-width calls, got %d", got)
	}
}

func TestMarkdownRenderer_CacheNeverExceedsOneEntry(t *testing.T) {
	clearMarkdownCache()
	t.Cleanup(clearMarkdownCache)

	RenderMarkdown("hello", 80)
	RenderMarkdown("hello", 100)
	RenderMarkdown("hello", 120)
	if got := markdownCacheLen(); got > 1 {
		t.Fatalf("cache should never exceed 1 entry, got %d (memory leak)", got)
	}
}

func TestMarkdownRenderer_CachedRendererProducesIdenticalOutput(t *testing.T) {
	clearMarkdownCache()
	t.Cleanup(clearMarkdownCache)

	content := "# Heading\n\nSome **bold** text with `code`.\n\n```go\nfmt.Println(\"hello\")\n```"
	first := RenderMarkdown(content, 80)
	second := RenderMarkdown(content, 80)
	if first != second {
		t.Fatalf("cached renderer produced different output:\nfirst:  %q\nsecond: %q", first, second)
	}
}

func TestMarkdownRenderer_DifferentWidthsProduceDifferentWrapping(t *testing.T) {
	clearMarkdownCache()
	t.Cleanup(clearMarkdownCache)

	// Narrow width should produce different wrapping than wide width.
	narrow := stripANSI(RenderMarkdown("This is a reasonably long line of text that should wrap at a narrow width.", 40))
	wide := stripANSI(RenderMarkdown("This is a reasonably long line of text that should wrap at a narrow width.", 200))
	if narrow == wide {
		t.Fatal("expected different output for different widths")
	}
}

func BenchmarkRenderMarkdown_Cached(b *testing.B) {
	clearMarkdownCache()
	b.Cleanup(clearMarkdownCache)
	content := "# Heading\n\nSome **bold** text.\n\n```go\nfmt.Println(\"hello\")\n```"
	b.ResetTimer()
	for range b.N {
		RenderMarkdown(content, 80)
	}
}
