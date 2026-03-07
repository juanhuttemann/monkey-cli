package tui

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/juanhuttemann/monkey-cli/api"
)

// Environment variables for overriding model pricing (USD per million tokens).
const (
	EnvPriceInput  = "MONKEY_PRICE_INPUT"
	EnvPriceOutput = "MONKEY_PRICE_OUTPUT"
)

// modelPricing holds per-million-token prices for a model family.
type modelPricing struct {
	inputPerMTok  float64
	outputPerMTok float64
}

// pricingDefaults maps lowercase model name substrings to default pricing.
// Checked in order; more specific patterns first.
var pricingDefaults = []struct {
	substr  string
	pricing modelPricing
}{
	{"opus", modelPricing{15.0, 75.0}},
	{"haiku", modelPricing{0.80, 4.0}},
	{"sonnet", modelPricing{3.0, 15.0}},
}

// modelDefaultPricing returns the default pricing for a model by name substring.
// Falls back to sonnet pricing for unrecognized models.
func modelDefaultPricing(modelName string) modelPricing {
	lower := strings.ToLower(modelName)
	for _, entry := range pricingDefaults {
		if strings.Contains(lower, entry.substr) {
			return entry.pricing
		}
	}
	return modelPricing{3.0, 15.0}
}

// envFloat reads an env var as a float64. Returns (value, true) on success,
// (0, false) when the variable is unset or empty.
func envFloat(key string) (float64, bool) {
	s := os.Getenv(key)
	if s == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || v < 0 {
		return 0, false
	}
	return v, true
}

// estimateCost returns the estimated USD cost for the given usage and model.
// Pricing is read from MONKEY_PRICE_INPUT / MONKEY_PRICE_OUTPUT (USD per million
// tokens) when set; otherwise falls back to model-name-based defaults.
func estimateCost(modelName string, usage api.Usage) float64 {
	if usage.InputTokens == 0 && usage.OutputTokens == 0 {
		return 0
	}
	defaults := modelDefaultPricing(modelName)

	inputPrice := defaults.inputPerMTok
	if v, ok := envFloat(EnvPriceInput); ok {
		inputPrice = v
	}
	outputPrice := defaults.outputPerMTok
	if v, ok := envFloat(EnvPriceOutput); ok {
		outputPrice = v
	}

	return float64(usage.InputTokens)/1_000_000*inputPrice +
		float64(usage.OutputTokens)/1_000_000*outputPrice
}

// formatCost formats a USD cost as a human-readable string (e.g. "$0.023").
// Returns "" when cost is zero.
func formatCost(cost float64) string {
	if cost == 0 {
		return ""
	}
	if cost < 0.01 {
		return "<$0.01"
	}
	if cost < 1 {
		return fmt.Sprintf("$%.3f", cost)
	}
	return fmt.Sprintf("$%.2f", cost)
}
