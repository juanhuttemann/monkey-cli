package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"mogger/api"
	"mogger/config"
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

	// Check if prompt is provided
	if prompt == "" {
		printUsage()
		fmt.Fprintln(os.Stderr, "\nError: -p flag is required")
		os.Exit(1)
	}

	fmt.Println(runPrompt(prompt))
}
