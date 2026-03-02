package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"monkey/api"
	"monkey/config"
	"monkey/tui"
)

// systemPromptCandidates returns the paths to check for a system prompt, in priority order.
// Local system.md takes precedence over the global ~/.config/monkey/system.md.
func systemPromptCandidates() []string {
	candidates := []string{"system.md"}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, home+"/.config/monkey/system.md")
	}
	return candidates
}

// buildClientOpts returns the client options for the given config,
// including the system prompt loaded from the first system.md found.
func buildClientOpts(cfg config.Config) ([]api.ClientOption, error) {
	opts := []api.ClientOption{
		api.WithModel(cfg.Model),
		api.WithMaxRetries(3),
		api.WithRetryDelay(time.Second),
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
	if systemPrompt != "" {
		opts = append(opts, api.WithSystemPrompt(systemPrompt))
	}
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

// launchTUI starts the interactive TUI
func launchTUI() {
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
	model.SetIntro(introContent())
	model.SetIntroTitle(AppTitle)
	model.SetIntroVersion("v" + Version)

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
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
	// Define flags
	promptFlag := flag.String("p", "", "Your prompt to send to the LLM")
	flag.StringVar(promptFlag, "prompt", "", "Your prompt to send to the LLM")

	flag.Parse()

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

	run(prompt, launchTUI)
}
