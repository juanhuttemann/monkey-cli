package main

import (
	"context"
	"fmt"
	"os"

	"mogger/api"
	"mogger/config"
)

// getGreeting calls the LLM API and returns a greeting message
func getGreeting() (string, error) {
	// Load configuration
	loader := config.NewEnvLoader()
	cfg, err := loader.Load()
	if err != nil {
		return "", err
	}

	// Create API client
	client := api.NewClient(cfg.BaseURL, cfg.APIKey, api.WithModel(cfg.Model))

	// Send message and get response
	return client.SendMessage(context.Background(), "Return a greeting message")
}

// printHello calls the LLM API and returns a greeting message
// On error, it prints to stderr and exits with code 1
func printHello() string {
	greeting, err := getGreeting()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	return greeting
}

func main() {
	fmt.Println(printHello())
}
