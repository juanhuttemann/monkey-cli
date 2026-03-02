package tui

import (
	"strings"
	"testing"
)

func TestView_ShowsIntroArt_WhenNoMessages(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 24)
	model.SetIntro("MONKEY_ASCII_ART_MARKER")

	view := model.View()

	if !strings.Contains(stripANSI(view), "MONKEY_ASCII_ART_MARKER") {
		t.Error("View() should show intro art when no messages")
	}
}

func TestView_ShowsVersion_WhenNoMessages(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 24)
	model.SetIntro("ART\nv1.2.3")

	view := model.View()

	if !strings.Contains(stripANSI(view), "v1.2.3") {
		t.Error("View() should show version line in intro")
	}
}

func TestView_ShowsIntroTitle_WhenNoMessages(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 24)
	model.SetIntro("art content")
	model.SetIntroTitle("MyApp")

	view := model.View()

	if !strings.Contains(stripANSI(view), "MyApp") {
		t.Error("View() should show intro title in block border")
	}
}

func TestView_HidesIntro_WhenMessagesPresent(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 24)
	model.SetIntro("UNIQUE_INTRO_MARKER")
	model.AddMessage("user", "Hello")

	view := model.View()

	if strings.Contains(stripANSI(view), "UNIQUE_INTRO_MARKER") {
		t.Error("View() should not show intro when messages are present")
	}
}

func TestView_NoIntro_WhenIntroNotSet(t *testing.T) {
	model := NewModel(nil)
	model.SetDimensions(80, 24)

	// Without SetIntro, nothing about intro should appear
	view := model.View()

	// Should still render without panicking and return something
	if view == "" {
		t.Error("View() should return non-empty string even without intro set")
	}
}
