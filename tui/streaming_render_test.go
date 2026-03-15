package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// TestStreaming_RenderedPriorValid_InitiallyFalse verifies that the rendered-prior
// cache starts in an invalid (uncomputed) state.
func TestStreaming_RenderedPriorValid_InitiallyFalse(t *testing.T) {
	m := NewModel(nil)
	if m.renderedPriorValid {
		t.Error("renderedPriorValid should be false on a new model")
	}
}

// TestStreaming_RenderedPriorValid_SetOnFirstToken verifies that the cache is
// populated (valid) after the very first streaming token creates a new
// assistant message.
func TestStreaming_RenderedPriorValid_SetOnFirstToken(t *testing.T) {
	m := NewModel(nil)
	m.messages = []Message{{Role: "user", Content: "hello", Timestamp: time.Now()}}
	m.printedCount = 0
	m.streaming = true

	updated, _ := m.Update(PartialResponseMsg{Token: "Hi"})
	result := updated.(Model)

	if !result.renderedPriorValid {
		t.Error("renderedPriorValid should be true after first streaming token")
	}
}

// TestStreaming_RenderedPriorValid_NotRecomputedOnSubsequentTokens verifies
// that the cache value is stable across multiple tokens (same string pointer
// content – we only verify it is non-empty and unchanged between tokens).
func TestStreaming_RenderedPriorValid_NotRecomputedOnSubsequentTokens(t *testing.T) {
	m := NewModel(nil)
	m.messages = []Message{{Role: "user", Content: "question", Timestamp: time.Now()}}
	m.printedCount = 0
	m.streaming = true

	// First token: prior cache should be computed and non-empty (user message).
	m1, _ := m.Update(PartialResponseMsg{Token: "A"})
	prior1 := m1.(Model).renderedPrior

	// Second token: prior cache should remain the same.
	m2, _ := m1.(Model).Update(PartialResponseMsg{Token: "B"})
	prior2 := m2.(Model).renderedPrior

	if prior1 != prior2 {
		t.Error("renderedPrior should not change between subsequent streaming tokens")
	}
	if prior1 == "" {
		t.Error("renderedPrior should be non-empty when there are prior messages")
	}
}

// TestStreaming_RenderedPriorValid_ClearedOnWindowResize verifies that a
// window-resize event invalidates the cache so the next token re-renders prior
// messages at the new width.
func TestStreaming_RenderedPriorValid_ClearedOnWindowResize(t *testing.T) {
	m := NewModel(nil)
	m.messages = []Message{{Role: "user", Content: "hello", Timestamp: time.Now()}}
	m.printedCount = 0
	m.streaming = true

	// Populate the cache.
	m1, _ := m.Update(PartialResponseMsg{Token: "Hi"})
	if !m1.(Model).renderedPriorValid {
		t.Fatal("precondition: cache should be valid before resize")
	}

	// Resize the window.
	m2, _ := m1.(Model).Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if m2.(Model).renderedPriorValid {
		t.Error("renderedPriorValid should be false after window resize")
	}
}

// TestStreaming_RenderedPriorValid_ClearedOnPromptResponse verifies that the
// cache is cleaned up when streaming finishes.
func TestStreaming_RenderedPriorValid_ClearedOnPromptResponse(t *testing.T) {
	m := NewModel(nil)
	m.messages = []Message{{Role: "user", Content: "hello", Timestamp: time.Now()}}
	m.printedCount = 0
	m.streaming = true

	m1, _ := m.Update(PartialResponseMsg{Token: "Hi"})
	if !m1.(Model).renderedPriorValid {
		t.Fatal("precondition: cache should be valid mid-stream")
	}

	m2, _ := m1.(Model).Update(PromptResponseMsg{Response: "Hi there"})
	if m2.(Model).renderedPriorValid {
		t.Error("renderedPriorValid should be false after PromptResponseMsg")
	}
}

// TestStreaming_RenderedPriorValid_ClearedOnCancel verifies that the cache is
// cleaned up when streaming is cancelled.
func TestStreaming_RenderedPriorValid_ClearedOnCancel(t *testing.T) {
	m := NewModel(nil)
	m.messages = []Message{{Role: "user", Content: "hello", Timestamp: time.Now()}}
	m.printedCount = 0
	m.streaming = true

	m1, _ := m.Update(PartialResponseMsg{Token: "Hi"})
	if !m1.(Model).renderedPriorValid {
		t.Fatal("precondition: cache should be valid mid-stream")
	}

	m.streaming = true
	m1m := m1.(Model)
	m1m.state = StateLoading
	m2, _ := m1m.Update(PromptCancelledMsg{})
	if m2.(Model).renderedPriorValid {
		t.Error("renderedPriorValid should be false after PromptCancelledMsg")
	}
}

// TestStreaming_ManyTokens_ContentAccumulatesCorrectly verifies that using the
// new strings.Builder path still produces the correct accumulated content.
func TestStreaming_ManyTokens_ContentAccumulatesCorrectly(t *testing.T) {
	m := NewModel(nil)
	m.streaming = true

	tokens := strings.Fields("The quick brown fox jumps over the lazy dog")
	current := tea.Model(m)
	for i, tok := range tokens {
		if i > 0 {
			// Re-add space since Fields splits on whitespace.
			current, _ = current.(Model).Update(PartialResponseMsg{Token: " "})
		}
		current, _ = current.(Model).Update(PartialResponseMsg{Token: tok})
	}

	msgs := current.(Model).GetHistory()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	want := "The quick brown fox jumps over the lazy dog"
	if msgs[0].Content != want {
		t.Errorf("content = %q, want %q", msgs[0].Content, want)
	}
}

// TestStreaming_NoPriorMessages_RenderedPriorIsEmpty verifies that when there
// are no prior messages the cache is empty (not nil-panic).
func TestStreaming_NoPriorMessages_RenderedPriorIsEmpty(t *testing.T) {
	m := NewModel(nil)
	m.streaming = true

	m1, _ := m.Update(PartialResponseMsg{Token: "hello"})
	result := m1.(Model)

	if result.renderedPrior != "" {
		t.Errorf("renderedPrior should be empty when there are no prior messages, got %q", result.renderedPrior)
	}
	if !result.renderedPriorValid {
		t.Error("renderedPriorValid should still be true (cache computed, just empty)")
	}
}

// updateInGoroutine calls Update in a fresh goroutine so that the value
// receiver for m is placed on a different goroutine stack on every call.
// This reliably changes the memory address of m.streamBuf between calls,
// triggering the strings.Builder copy-after-write panic.
func updateInGoroutine(m tea.Model, msg tea.Msg) tea.Model {
	ch := make(chan tea.Model, 1)
	go func() {
		result, _ := m.Update(msg)
		ch <- result
	}()
	return <-ch
}

// TestStreaming_MultipleTokens_DoesNotPanic is a regression test for the
// strings.Builder copy-after-use panic. Model.Update has a value receiver, so
// m (and m.streamBuf) is copied on every call. strings.Builder panics when
// copied after first write ("illegal use of non-zero Builder copied by value").
//
// The panic is triggered when the method receiver lands at a different stack
// address on successive calls — which happens in bubbletea's event loop when
// other messages are dispatched between streaming tokens. Calling Update in a
// fresh goroutine each time guarantees a different stack address, reliably
// reproducing the panic.
func TestStreaming_MultipleTokens_DoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Update panicked on repeated streaming tokens: %v", r)
		}
	}()

	m := NewModel(nil)
	m.SetDimensions(80, 24)
	m.streaming = true

	current := tea.Model(m)
	for range 5 {
		current = updateInGoroutine(current, PartialResponseMsg{Token: "word "})
	}
}

// TestRenderMessages_Streaming_UsesRenderedPriorCache verifies that renderMessages
// uses the renderedPrior cache instead of re-rendering prior messages on every token.
// This test fails if the fast path is absent (renderMessages ignores renderedPrior).
func TestRenderMessages_Streaming_UsesRenderedPriorCache(t *testing.T) {
	m := NewModel(nil)
	m.SetDimensions(80, 40)
	m.messages = []Message{
		{Role: "user", Content: "prior question", Timestamp: time.Now()},
		{Role: "assistant", Content: "partial answer", Timestamp: time.Now()},
	}
	m.printedCount = 0
	m.streaming = true
	// Install a recognisable sentinel in place of the real prior render.
	m.renderedPrior = "CACHED_PRIOR_SENTINEL\n"
	m.renderedPriorValid = true

	rendered := stripANSI(m.renderMessages())

	if !strings.Contains(rendered, "CACHED_PRIOR_SENTINEL") {
		t.Error("renderMessages during streaming should use renderedPrior cache for prior messages")
	}
}

// TestRenderMessages_Streaming_FastPathMatchesFullRender verifies that the
// streaming fast path produces byte-identical output to the full render path.
func TestRenderMessages_Streaming_FastPathMatchesFullRender(t *testing.T) {
	m := NewModel(nil)
	m.SetDimensions(80, 40)
	m.messages = []Message{
		{Role: "user", Content: "question", Timestamp: time.Now()},
		{Role: "assistant", Content: "partial answer so far", Timestamp: time.Now()},
	}
	m.printedCount = 0
	m.streaming = true

	// Build the prior cache exactly as Update does on the first streaming token.
	sw := m.messageStyleWidth()
	m.renderedPrior = m.renderMessageEntry(sw, 0)
	m.renderedPriorValid = true

	fastOutput := m.renderMessages()

	// Full path: invalidate cache so the loop runs for all messages.
	m.renderedPriorValid = false
	fullOutput := m.renderMessages()

	if fastOutput != fullOutput {
		t.Errorf("streaming fast path differs from full render\nfast:\n%s\nfull:\n%s",
			stripANSI(fastOutput), stripANSI(fullOutput))
	}
}

// BenchmarkStreaming_ManyTokensWithPriorMessages measures the cost of
// processing N streaming tokens when prior messages exist in the viewport.
func BenchmarkStreaming_ManyTokensWithPriorMessages(b *testing.B) {
	base := NewModel(nil)
	base.SetDimensions(80, 40)
	// Add several prior messages to populate the viewport.
	for i := 0; i < 5; i++ {
		base.messages = append(base.messages, Message{
			Role:      "user",
			Content:   "This is prior user message number " + string(rune('0'+i)),
			Timestamp: time.Now(),
		})
		base.messages = append(base.messages, Message{
			Role:      "assistant",
			Content:   "This is a prior assistant reply with some **markdown** formatting.",
			Timestamp: time.Now(),
		})
	}
	base.streaming = true

	b.ResetTimer()
	for range b.N {
		m := base // copy
		m.streamBuf = m.streamBuf[:0]
		m.renderedPriorValid = false
		current := tea.Model(m)
		for range 50 {
			current, _ = current.(Model).Update(PartialResponseMsg{Token: "word "})
		}
	}
}
