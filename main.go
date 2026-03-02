package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"mogger/api"
	"mogger/config"
	"mogger/tui"
)

// sendPrompt calls the LLM API with the user-provided prompt and returns the response
func sendPrompt(prompt string) (string, error) {
	// Load configuration
	loader := config.NewEnvLoader()
	cfg, err := loader.Load()
	if err != nil {
		return "", err
	}

	// Create API client
	client := api.NewClient(cfg.BaseURL, cfg.APIKey, api.WithModel(cfg.Model))

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
	fmt.Fprintln(os.Stderr, "Usage: mogger -p \"<prompt>\"")
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

	client := api.NewClient(cfg.BaseURL, cfg.APIKey, api.WithModel(cfg.Model))
	model := tui.NewModel(client)

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
