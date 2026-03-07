package tui

import (
	"math"
	"os"
	"testing"

	"github.com/juanhuttemann/monkey-cli/api"
)

// --- estimateCost ---

func TestEstimateCost_ZeroUsage(t *testing.T) {
	cost := estimateCost("claude-sonnet-4-6", api.Usage{})
	if cost != 0 {
		t.Errorf("estimateCost with zero usage = %v, want 0", cost)
	}
}

func TestEstimateCost_SonnetDefault(t *testing.T) {
	// Sonnet: $3/MTok input, $15/MTok output (default when no env vars set)
	unsetPricingEnv(t)
	usage := api.Usage{InputTokens: 1_000_000, OutputTokens: 1_000_000}
	cost := estimateCost("claude-sonnet-4-6", usage)
	want := 3.0 + 15.0
	if math.Abs(cost-want) > 0.001 {
		t.Errorf("estimateCost(sonnet, 1M/1M) = %.4f, want %.4f", cost, want)
	}
}

func TestEstimateCost_OpusDefault(t *testing.T) {
	// Opus: $15/MTok input, $75/MTok output (default)
	unsetPricingEnv(t)
	usage := api.Usage{InputTokens: 1_000_000, OutputTokens: 1_000_000}
	cost := estimateCost("claude-opus-4-6", usage)
	want := 15.0 + 75.0
	if math.Abs(cost-want) > 0.001 {
		t.Errorf("estimateCost(opus, 1M/1M) = %.4f, want %.4f", cost, want)
	}
}

func TestEstimateCost_HaikuDefault(t *testing.T) {
	// Haiku: $0.80/MTok input, $4/MTok output (default)
	unsetPricingEnv(t)
	usage := api.Usage{InputTokens: 1_000_000, OutputTokens: 1_000_000}
	cost := estimateCost("claude-haiku-4-5", usage)
	want := 0.80 + 4.0
	if math.Abs(cost-want) > 0.001 {
		t.Errorf("estimateCost(haiku, 1M/1M) = %.4f, want %.4f", cost, want)
	}
}

func TestEstimateCost_EnvVarOverridesModel(t *testing.T) {
	t.Setenv(EnvPriceInput, "10.0")
	t.Setenv(EnvPriceOutput, "30.0")
	usage := api.Usage{InputTokens: 1_000_000, OutputTokens: 1_000_000}
	// Env var should override model-based pricing
	cost := estimateCost("claude-opus-4-6", usage)
	want := 10.0 + 30.0
	if math.Abs(cost-want) > 0.001 {
		t.Errorf("estimateCost with env override = %.4f, want %.4f", cost, want)
	}
}

func TestEstimateCost_PartialEnvVar_InputOnly(t *testing.T) {
	t.Setenv(EnvPriceInput, "5.0")
	t.Setenv(EnvPriceOutput, "") // not set
	usage := api.Usage{InputTokens: 1_000_000, OutputTokens: 1_000_000}
	cost := estimateCost("claude-sonnet-4-6", usage)
	// input overridden to $5, output falls back to sonnet default $15
	want := 5.0 + 15.0
	if math.Abs(cost-want) > 0.001 {
		t.Errorf("estimateCost with input-only env = %.4f, want %.4f", cost, want)
	}
}

func TestEstimateCost_UnknownModel_FallsBackToSonnet(t *testing.T) {
	unsetPricingEnv(t)
	usage := api.Usage{InputTokens: 1_000_000, OutputTokens: 0}
	cost := estimateCost("some-unknown-model-xyz", usage)
	// Should use sonnet default ($3/MTok input)
	want := 3.0
	if math.Abs(cost-want) > 0.001 {
		t.Errorf("estimateCost(unknown, 1M/0) = %.4f, want %.4f (sonnet fallback)", cost, want)
	}
}

func TestEstimateCost_SmallUsage(t *testing.T) {
	unsetPricingEnv(t)
	usage := api.Usage{InputTokens: 10_000, OutputTokens: 2_000}
	cost := estimateCost("claude-sonnet-4-6", usage)
	// input: 10_000/1_000_000 * 3 = 0.03; output: 2_000/1_000_000 * 15 = 0.03
	want := 0.03 + 0.03
	if math.Abs(cost-want) > 0.0001 {
		t.Errorf("estimateCost small usage = %.6f, want %.6f", cost, want)
	}
}

// --- formatCost ---

func TestFormatCost_Zero(t *testing.T) {
	if got := formatCost(0); got != "" {
		t.Errorf("formatCost(0) = %q, want ''", got)
	}
}

func TestFormatCost_SubCent(t *testing.T) {
	got := formatCost(0.003)
	if got == "" {
		t.Error("formatCost(0.003) = '', want non-empty")
	}
	if got != "<$0.01" {
		t.Errorf("formatCost(0.003) = %q, want '<$0.01'", got)
	}
}

func TestFormatCost_Cents(t *testing.T) {
	got := formatCost(0.023)
	if got == "" {
		t.Error("formatCost(0.023) = '', want non-empty")
	}
	wantContainsDollar := false
	for _, c := range got {
		if c == '$' {
			wantContainsDollar = true
		}
	}
	if !wantContainsDollar {
		t.Errorf("formatCost(0.023) = %q, want string containing '$'", got)
	}
}

func TestFormatCost_Dollar(t *testing.T) {
	got := formatCost(1.50)
	if got == "" {
		t.Error("formatCost(1.50) = '', want non-empty")
	}
}

// unsetPricingEnv clears the pricing env vars for the duration of the test.
func unsetPricingEnv(t *testing.T) {
	t.Helper()
	old1 := os.Getenv(EnvPriceInput)
	old2 := os.Getenv(EnvPriceOutput)
	os.Unsetenv(EnvPriceInput)
	os.Unsetenv(EnvPriceOutput)
	t.Cleanup(func() {
		os.Setenv(EnvPriceInput, old1)
		os.Setenv(EnvPriceOutput, old2)
	})
}
