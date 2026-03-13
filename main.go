package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/juanhuttemann/monkey-cli/api"
	"github.com/juanhuttemann/monkey-cli/config"
	"github.com/juanhuttemann/monkey-cli/tui"
)

// gitBranch returns the current git branch in dir, or empty string if not a git repo.
func gitBranch(dir string) string {
	out, err := exec.Command("git", "-C", dir, "symbolic-ref", "--short", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// buildDynamicContext returns a string with dynamic runtime context (date, cwd, git branch).
func buildDynamicContext(now time.Time, cwd string) string {
	var sb strings.Builder
	sb.WriteString("Today's date: " + now.Format("2006-01-02") + "\n")
	sb.WriteString("Working directory: " + cwd + "\n")
	if branch := gitBranch(cwd); branch != "" {
		sb.WriteString("Git branch: " + branch + "\n")
	}
	return sb.String()
}

// systemPromptCandidates returns the paths to check for the system prompt, in priority order.
// Local MONKEY.md takes precedence over the global ~/.config/monkey/MONKEY.md.
func systemPromptCandidates() []string {
	candidates := []string{"MONKEY.md"}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, home+"/.config/monkey/MONKEY.md")
	}
	return candidates
}

// claudeMDCandidates returns paths to CLAUDE.md files to append as project context.
func claudeMDCandidates() []string {
	candidates := []string{"CLAUDE.md"}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, home+"/.claude/CLAUDE.md")
	}
	return candidates
}

// buildClientOpts returns the client options for the given config,
// including the system prompt loaded from the first MONKEY.md found.
func buildClientOpts(cfg config.Config) ([]api.ClientOption, error) {
	opts := []api.ClientOption{
		api.WithModel(cfg.DefaultModel()),
		api.WithMaxRetries(3),
	}
	if cfg.MaxTokens > 0 {
		opts = append(opts, api.WithMaxTokens(cfg.MaxTokens))
	}
	var systemPrompt string
	for _, path := range systemPromptCandidates() {
		s, err := config.LoadSystemPromptFile(path)
		if err != nil {
			return nil, err
		}
		if s != "" {
			systemPrompt = s
			break
		}
	}
	// Append any CLAUDE.md files found as additional project context.
	for _, path := range claudeMDCandidates() {
		s, err := config.LoadSystemPromptFile(path)
		if err != nil {
			return nil, err
		}
		if s != "" {
			if systemPrompt != "" {
				systemPrompt += "\n\n" + s
			} else {
				systemPrompt = s
			}
		}
	}
	cwd, _ := os.Getwd()
	prefix := buildDynamicContext(time.Now(), cwd)
	if systemPrompt != "" {
		systemPrompt = prefix + systemPrompt
	} else {
		systemPrompt = prefix
	}
	opts = append(opts, api.WithSystemPrompt(systemPrompt))
	return opts, nil
}

// sendPrompt calls the LLM API with the user-provided prompt and returns the response
func sendPrompt(prompt string) (string, error) {
	// Load configuration
	loader := config.NewEnvLoader()
	cfg, err := loader.Load()
	if err != nil {
		return "", err
	}

	// Create API client
	opts, err := buildClientOpts(cfg)
	if err != nil {
		return "", err
	}
	client := api.NewClient(cfg.BaseURL, cfg.APIKey, opts...)

	// Send message and get response
	return client.SendMessage(context.Background(), prompt)
}

// runPrompt sends the prompt to the LLM API and returns the response
// On error, it prints to stderr and exits with code 1
func runPrompt(prompt string) string {
	response, err := sendPrompt(prompt)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	return response
}

// printVersion prints the application name and version to stdout.
func printVersion() {
	fmt.Printf("%s v%s\n", AppTitle, Version)
}

// printUsage displays the help message to stderr
func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: monkey -p \"<prompt>\"")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Flags:")
	fmt.Fprintln(os.Stderr, "  -p, --prompt string   Your prompt to send to the LLM (required)")
}

// shouldLaunchTUI returns true when no prompt was provided (empty or whitespace-only)
func shouldLaunchTUI(prompt string) bool {
	return strings.TrimSpace(prompt) == ""
}

// launchTUI starts the interactive TUI.
// If continueSession is true, the last saved session is restored.
func launchTUI(continueSession bool) {
	loader := config.NewEnvLoader()
	cfg, err := loader.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	opts, err := buildClientOpts(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	client := api.NewClient(cfg.BaseURL, cfg.APIKey, opts...)
	model := tui.NewModel(client)
	model.SetModels(cfg.AvailableModels())
	model.SetIntro(introContent())
	model.SetIntroTitle(AppTitle)
	model.SetIntroVersion("v" + Version)

	if continueSession {
		sess, err := tui.LoadSession(tui.SessionPath())
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not load session: %v\n", err)
		} else {
			model.RestoreSession(sess)
		}
	}

	p := tea.NewProgram(model)
	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	// Persist session for --continue on next invocation.
	if m, ok := finalModel.(tui.Model); ok {
		modelName := ""
		if client != nil {
			modelName = client.GetModel()
		}
		_ = tui.SaveSession(tui.SessionPath(), modelName, m.GetAPIMessages(), m.GetHistory())
	}
}

// run is the core application logic, separated for testability.
// tuiRunner is called when no prompt is provided; in production this is launchTUI.
func run(prompt string, tuiRunner func()) {
	if shouldLaunchTUI(prompt) {
		tuiRunner()
		return
	}
	fmt.Println(runPrompt(prompt))
}

func main() {
	// Handle subcommands before flag parsing
	if len(os.Args) > 1 && os.Args[1] == "update" {
		msg, err := runUpdate(githubAPIURL, "")
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		fmt.Println(msg)
		return
	}

	// Define flags
	promptFlag := flag.String("p", "", "Your prompt to send to the LLM")
	flag.StringVar(promptFlag, "prompt", "", "Your prompt to send to the LLM")
	continueFlag := flag.Bool("continue", false, "Resume the last saved session")
	versionFlag := flag.Bool("v", false, "Print version and exit")
	flag.BoolVar(versionFlag, "version", false, "Print version and exit")

	flag.Parse()

	if *versionFlag {
		printVersion()
		return
	}

	// Build prompt from flag value and any remaining positional arguments
	// This supports both quoted prompts (-p "hello world") and unquoted (-p hello world)
	prompt := *promptFlag
	if len(flag.Args()) > 0 {
		if prompt == "" {
			prompt = strings.Join(flag.Args(), " ")
		} else {
			prompt = prompt + " " + strings.Join(flag.Args(), " ")
		}
	}

	run(prompt, func() { launchTUI(*continueFlag) })
}
