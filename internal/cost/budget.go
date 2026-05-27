package cost

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BrianNguyen29/x-harness/internal/loader"
	"gopkg.in/yaml.v3"
)

type CostBudgetPolicy struct {
	MaxInputTokens     int64   `yaml:"max_input_tokens"`
	MaxOutputTokens    int64   `yaml:"max_output_tokens"`
	MaxActualUSD       float64 `yaml:"max_actual_usd"`
	AffectsAdmission   bool    `yaml:"affects_admission"`
	Enabled            bool
	OverBudgetRecovery string
}

type CostBudgetReport struct {
	OK                 bool    `json:"ok"`
	Status             string  `json:"status"`
	OverBudget         bool    `json:"over_budget"`
	EnforcementEnabled bool    `json:"enforcement_enabled"`
	ActualUSD          float64 `json:"actual_usd"`
	MaxUSD             float64 `json:"max_usd"`
	InputTokens        int64   `json:"input_tokens"`
	MaxInputTokens     int64   `json:"max_input_tokens"`
	OutputTokens       int64   `json:"output_tokens"`
	MaxOutputTokens    int64   `json:"max_output_tokens"`
	AffectsAdmission   bool    `json:"affects_admission"`
}

type rawPolicy struct {
	Version    int `yaml:"version"`
	CostBudget struct {
		Enabled            bool    `yaml:"enabled"`
		MaxUSD             float64 `yaml:"max_usd"`
		MaxInputTokens     int64   `yaml:"max_input_tokens"`
		MaxOutputTokens    int64   `yaml:"max_output_tokens"`
		OverBudgetRecovery string  `yaml:"over_budget_recovery"`
		AffectsAdmission   bool    `yaml:"affects_admission"`
	} `yaml:"cost_budget"`
}

func LoadPolicy(root string) (*CostBudgetPolicy, error) {
	policyPath := filepath.Join(root, "policies", "cost-budget.yaml")
	var raw rawPolicy
	if err := loader.LoadYAML(policyPath, &raw); err != nil {
		return nil, fmt.Errorf("failed to load cost policy: %w", err)
	}
	return &CostBudgetPolicy{
		MaxInputTokens:     raw.CostBudget.MaxInputTokens,
		MaxOutputTokens:    raw.CostBudget.MaxOutputTokens,
		MaxActualUSD:       raw.CostBudget.MaxUSD,
		AffectsAdmission:   raw.CostBudget.AffectsAdmission,
		Enabled:            raw.CostBudget.Enabled,
		OverBudgetRecovery: raw.CostBudget.OverBudgetRecovery,
	}, nil
}

func EvaluateCostBudget(policy *CostBudgetPolicy, actualUSD float64, inputTokens, outputTokens int64, enforce bool) *CostBudgetReport {
	overBudget := actualUSD > policy.MaxActualUSD ||
		inputTokens > policy.MaxInputTokens ||
		outputTokens > policy.MaxOutputTokens

	status := "within_budget"
	if overBudget {
		status = "over_budget"
	}

	return &CostBudgetReport{
		OK:                 !overBudget,
		Status:             status,
		OverBudget:         overBudget,
		EnforcementEnabled: enforce && policy.Enabled,
		ActualUSD:          actualUSD,
		MaxUSD:             policy.MaxActualUSD,
		InputTokens:        inputTokens,
		MaxInputTokens:     policy.MaxInputTokens,
		OutputTokens:       outputTokens,
		MaxOutputTokens:    policy.MaxOutputTokens,
		AffectsAdmission:   policy.AffectsAdmission,
	}
}

func ReadCostBudgetReport(reportPath string) (*CostBudgetReport, error) {
	data, err := os.ReadFile(reportPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read report: %w", err)
	}

	var report CostBudgetReport
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&report); err == nil && report.Status != "" {
		return &report, nil
	}

	// Fall back to schema/TS format
	var tsReport struct {
		SchemaVersion string  `json:"schema_version"`
		MaxUSD        float64 `json:"max_usd"`
		ActualUSD     float64 `json:"actual_usd"`
		TokenUsage    struct {
			Input  int64 `json:"input"`
			Output int64 `json:"output"`
		} `json:"token_usage"`
		OverBudget         bool   `json:"over_budget"`
		Status             string `json:"status"`
		Recovery           string `json:"recovery"`
		PolicyEnabled      bool   `json:"policy_enabled"`
		EnforcementEnabled bool   `json:"enforcement_enabled"`
		AdmissionAuthority bool   `json:"admission_authority"`
	}
	if err := json.Unmarshal(data, &tsReport); err != nil {
		if err := yaml.Unmarshal(data, &tsReport); err != nil {
			return nil, fmt.Errorf("failed to parse report: %w", err)
		}
	}

	return &CostBudgetReport{
		OK:                 !tsReport.OverBudget,
		Status:             tsReport.Status,
		OverBudget:         tsReport.OverBudget,
		EnforcementEnabled: tsReport.EnforcementEnabled,
		ActualUSD:          tsReport.ActualUSD,
		MaxUSD:             tsReport.MaxUSD,
		InputTokens:        tsReport.TokenUsage.Input,
		OutputTokens:       tsReport.TokenUsage.Output,
		AffectsAdmission:   tsReport.AdmissionAuthority,
	}, nil
}
