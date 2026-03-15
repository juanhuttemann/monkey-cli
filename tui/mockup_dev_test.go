//go:build dev

package tui

import (
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// pump simulates the BubbleTea runtime: runs commands concurrently, feeds
// results into Update, and collects every View() snapshot along the way.
func pump(t *testing.T, m Model, initial tea.Cmd) (Model, []string) {
	t.Helper()

	msgCh := make(chan tea.Msg, 64)
	var wg sync.WaitGroup

	var launch func(cmd tea.Cmd)
	launch = func(cmd tea.Cmd) {
		if cmd == nil {
			return
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			msg := cmd()
			if msg == nil {
				return
			}
			if batch, ok := msg.(tea.BatchMsg); ok {
				for _, c := range batch {
					launch(c)
				}
				return
			}
			msgCh <- msg
		}()
	}

	launch(initial)
	go func() { wg.Wait(); close(msgCh) }()

	var snapshots []string
	snapshots = append(snapshots, stripANSI(m.View()))

	deadline := time.After(15 * time.Second)
	for m.state == StateLoading {
		select {
		case msg, ok := <-msgCh:
			if !ok {
				return m, snapshots
			}
			updated, nextCmd := m.Update(msg)
			m = updated.(Model)
			snapshots = append(snapshots, stripANSI(m.View()))
			launch(nextCmd)
		case <-deadline:
			t.Fatal("scenario did not complete within 15 seconds")
		}
	}
	return m, snapshots
}

func runScenario(t *testing.T, name string, ape bool) (Model, []string) {
	t.Helper()
	idx := scenarioIndex(t, name)

	m := NewModel(nil)
	m.SetDimensions(120, 40)
	m.autoApprove = ape

	m, cmd := setupMockupModel(m, idx)
	return pump(t, m, cmd)
}

func printViews(t *testing.T, snapshots []string) {
	t.Helper()
	// Print the first, a middle, and final snapshot so the output is readable.
	show := []int{0, len(snapshots) / 2, len(snapshots) - 1}
	seen := map[int]bool{}
	for _, i := range show {
		if i < 0 || i >= len(snapshots) || seen[i] {
			continue
		}
		seen[i] = true
		t.Logf("── snapshot %d/%d ──────────────────────────────────\n%s",
			i+1, len(snapshots), snapshots[i])
	}
}

// pumpDenying is like pump but automatically denies every tool approval request.
func pumpDenying(t *testing.T, m Model, initial tea.Cmd) (Model, []string) {
	t.Helper()

	msgCh := make(chan tea.Msg, 64)
	var wg sync.WaitGroup

	var launch func(cmd tea.Cmd)
	launch = func(cmd tea.Cmd) {
		if cmd == nil {
			return
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			msg := cmd()
			if msg == nil {
				return
			}
			if batch, ok := msg.(tea.BatchMsg); ok {
				for _, c := range batch {
					launch(c)
				}
				return
			}
			msgCh <- msg
		}()
	}

	launch(initial)
	go func() { wg.Wait(); close(msgCh) }()

	var snapshots []string
	snapshots = append(snapshots, stripANSI(m.View()))

	deadline := time.After(15 * time.Second)
	for m.state == StateLoading {
		select {
		case msg, ok := <-msgCh:
			if !ok {
				return m, snapshots
			}
			if req, ok := msg.(ToolApprovalRequestMsg); ok {
				req.ResponseCh <- false
			}
			updated, nextCmd := m.Update(msg)
			m = updated.(Model)
			snapshots = append(snapshots, stripANSI(m.View()))
			launch(nextCmd)
		case <-deadline:
			t.Fatal("scenario did not complete within 15 seconds")
		}
	}
	return m, snapshots
}

// ── scenario rendering tests ──────────────────────────────────────────────────

func TestMockup_Render_SimpleText(t *testing.T) {
	m, views := runScenario(t, "simple text", false)
	printViews(t, views)
	assertContains(t, strings.Join(views, " "), "dev mockup")
	assertContainsRole(t, m, roleAssistant)
}

func TestMockup_Render_BashTool(t *testing.T) {
	m, views := runScenario(t, "bash tool", true)
	printViews(t, views)
	all := strings.Join(views, " ")
	assertContains(t, all, "bash")
	assertContains(t, all, "ls -la")
	assertHasToolMessage(t, m, "bash")
}

func TestMockup_Render_ReadTool(t *testing.T) {
	m, views := runScenario(t, "read tool", true)
	printViews(t, views)
	assertHasToolMessage(t, m, "read")
}

func TestMockup_Render_WriteTool(t *testing.T) {
	m, views := runScenario(t, "write tool", true)
	printViews(t, views)
	assertHasToolMessage(t, m, "write")
}

func TestMockup_Render_EditTool(t *testing.T) {
	m, views := runScenario(t, "edit tool", true)
	printViews(t, views)
	assertHasToolMessage(t, m, "edit")
}

func TestMockup_Render_GlobTool(t *testing.T) {
	m, views := runScenario(t, "glob tool", true)
	printViews(t, views)
	assertHasToolMessage(t, m, "glob")
}

func TestMockup_Render_GrepTool(t *testing.T) {
	m, views := runScenario(t, "grep tool", true)
	printViews(t, views)
	assertHasToolMessage(t, m, "grep")
}

func TestMockup_Render_WebSearch(t *testing.T) {
	m, views := runScenario(t, "web_search tool", true)
	printViews(t, views)
	assertHasToolMessage(t, m, "web_search")
}

func TestMockup_Render_WebFetch(t *testing.T) {
	m, views := runScenario(t, "web_fetch tool", true)
	printViews(t, views)
	assertHasToolMessage(t, m, "web_fetch")
}

func TestMockup_Render_RateLimitError(t *testing.T) {
	_, views := runScenario(t, "rate limit error", false)
	printViews(t, views)
	assertContains(t, strings.Join(views, " "), "429")
}

func TestMockup_Render_ServerError(t *testing.T) {
	_, views := runScenario(t, "server error", false)
	printViews(t, views)
	assertContains(t, strings.Join(views, " "), "500")
}

func TestMockup_Render_NetworkError(t *testing.T) {
	_, views := runScenario(t, "network error", false)
	printViews(t, views)
	assertContains(t, strings.Join(views, " "), "network")
}

func TestMockup_Render_Retry(t *testing.T) {
	_, views := runScenario(t, "retry then success", false)
	printViews(t, views)
	// One of the intermediate snapshots must show the retry spinner label.
	found := false
	for _, v := range views {
		if strings.Contains(v, "Retrying") || strings.Contains(v, "rate limit") {
			found = true
			break
		}
	}
	if !found {
		t.Error("no snapshot showed retry state")
	}
	// The success response gets committed to scrollback, so search all snapshots.
	assertContains(t, strings.Join(views, " "), "retry")
}

func TestMockup_Render_MultiTool(t *testing.T) {
	m, views := runScenario(t, "multi-tool", true)
	printViews(t, views)
	count := 0
	for _, msg := range m.GetHistory() {
		if msg.Role == roleTool {
			count++
		}
	}
	if count < 2 {
		t.Errorf("expected ≥2 tool messages, got %d", count)
	}
}

func TestMockup_Render_DenialDemo(t *testing.T) {
	idx := scenarioIndex(t, "denial demo")

	m := NewModel(nil)
	m.SetDimensions(120, 40)
	m.autoApprove = false

	m, cmd := setupMockupModel(m, idx)
	final, views := pumpDenying(t, m, cmd)

	printViews(t, views)
	// The dialog must have appeared at some point.
	assertContains(t, strings.Join(views, " "), "bash")
	// After denial the scenario is cancelled; model should not still be loading.
	if final.state == StateLoading {
		t.Error("model still in StateLoading after denial")
	}
	if !final.wasCancelled {
		t.Error("expected wasCancelled=true after denial")
	}
}

// TestMockup_Render_AllScenarios prints a single consolidated rendering of
// every scenario so the full output can be reviewed at once.
func TestMockup_Render_AllScenarios(t *testing.T) {
	for i, sc := range mockupScenarios {
		t.Run(sc.Name, func(t *testing.T) {
			m := NewModel(nil)
			m.SetDimensions(120, 40)
			m.autoApprove = true

			m, cmd := setupMockupModel(m, i)
			final, views := pump(t, m, cmd)
			// Reset printedCount so View() shows all messages (not just uncommitted ones).
			final.printedCount = 0
			fullView := stripANSI(final.View())

			t.Logf("scenario %d/%d: %q  (%d snapshots, history=%d msgs)\n%s",
				i+1, len(mockupScenarios), sc.Name, len(views), len(final.GetHistory()), fullView)
		})
	}
}

// ── helpers ────────────────────────────────────────────────────────────────────

func scenarioIndex(t *testing.T, name string) int {
	t.Helper()
	for i, sc := range mockupScenarios {
		if sc.Name == name {
			return i
		}
	}
	t.Fatalf("scenario %q not found", name)
	return -1
}

func assertContains(t *testing.T, view, substr string, _ ...any) {
	t.Helper()
	if !strings.Contains(view, substr) {
		t.Errorf("expected view to contain %q\n── view ──\n%s", substr, view)
	}
}

func assertHasToolMessage(t *testing.T, m Model, toolName string) {
	t.Helper()
	for _, msg := range m.GetHistory() {
		if msg.Role == roleTool && strings.Contains(msg.ToolName, toolName) {
			return
		}
	}
	t.Errorf("no tool message with name %q in history", toolName)
}

func assertContainsRole(t *testing.T, m Model, role string) {
	t.Helper()
	for _, msg := range m.GetHistory() {
		if msg.Role == role {
			return
		}
	}
	t.Errorf("no message with role %q in history", role)
}

