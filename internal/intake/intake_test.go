package intake

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestPolicy(t *testing.T, root string) {
	t.Helper()
	policyPath := filepath.Join(root, "policies")
	if err := os.MkdirAll(policyPath, 0755); err != nil {
		t.Fatal(err)
	}
	content := `version: 1
intake_labels:
  tiny:
    runtime_tier: light
    signals:
      - comment_only
      - documentation
  normal:
    runtime_tier: standard
    signals:
      - routine_implementation
      - standard_refactor
  high_risk:
    runtime_tier: deep
    signals:
      - auth
      - token
      - session
      - admission
      - schema
      - permissions
      - ci
      - release
      - destructive_filesystem
high_risk_signals:
  auth:
    description: Authentication changes
    examples:
      - login
  token:
    description: Token handling
    examples:
      - refresh
  session:
    description: Session management
    examples:
      - creation
  admission:
    description: Admission policy
    examples:
      - verify gate
  schema:
    description: Schema changes
    examples:
      - completion card
  permissions:
    description: Permission changes
    examples:
      - chmod
  ci:
    description: CI/CD changes
    examples:
      - workflow
  release:
    description: Release logic
    examples:
      - script
  destructive_filesystem:
    description: Destructive ops
    examples:
      - rm -rf
runtime_tier_confirmation:
  tiers: [light, standard, deep]
  note: Tiers remain light, standard, deep.
`
	if err := os.WriteFile(filepath.Join(policyPath, "intake.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadIntakePolicyMissing(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := LoadIntakePolicy(tmpDir)
	if err == nil {
		t.Fatal("expected error for missing policy")
	}
}

func TestLoadIntakePolicyPresent(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestPolicy(t, tmpDir)
	policy, err := LoadIntakePolicy(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy == nil {
		t.Fatal("expected policy")
	}
	if policy.Version != 1 {
		t.Fatalf("expected version 1, got %d", policy.Version)
	}
}

func TestClassifyTaskCommentOnly(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestPolicy(t, tmpDir)
	policy, _ := LoadIntakePolicy(tmpDir)

	result := ClassifyTask("update docs", []string{}, "comment-only", policy)
	if result.IntakeLabel != IntakeLabelTiny {
		t.Fatalf("expected tiny, got %s", result.IntakeLabel)
	}
	if result.RuntimeTier != RuntimeTierLight {
		t.Fatalf("expected light, got %s", result.RuntimeTier)
	}
}

func TestClassifyTaskHighRiskKeyword(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestPolicy(t, tmpDir)
	policy, _ := LoadIntakePolicy(tmpDir)

	result := ClassifyTask("update auth logic", []string{}, "", policy)
	if result.IntakeLabel != IntakeLabelHighRisk {
		t.Fatalf("expected high_risk, got %s", result.IntakeLabel)
	}
	if result.RuntimeTier != RuntimeTierDeep {
		t.Fatalf("expected deep, got %s", result.RuntimeTier)
	}
	if !result.AutoEscalated {
		t.Fatal("expected auto_escalated")
	}
}

func TestClassifyTaskHighRiskFilePattern(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestPolicy(t, tmpDir)
	policy, _ := LoadIntakePolicy(tmpDir)

	result := ClassifyTask("routine fix", []string{"src/auth.go"}, "", policy)
	if result.IntakeLabel != IntakeLabelHighRisk {
		t.Fatalf("expected high_risk, got %s", result.IntakeLabel)
	}
}

func TestClassifyTaskCIWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestPolicy(t, tmpDir)
	policy, _ := LoadIntakePolicy(tmpDir)

	result := ClassifyTask("routine fix", []string{".github/workflows/ci.yml"}, "", policy)
	if result.IntakeLabel != IntakeLabelHighRisk {
		t.Fatalf("expected high_risk, got %s", result.IntakeLabel)
	}
}

func TestClassifyTaskDestructivePattern(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestPolicy(t, tmpDir)
	policy, _ := LoadIntakePolicy(tmpDir)

	result := ClassifyTask("routine fix", []string{"scripts/rm -rf old.sh"}, "", policy)
	if result.IntakeLabel != IntakeLabelHighRisk {
		t.Fatalf("expected high_risk, got %s", result.IntakeLabel)
	}
}

func TestClassifyTaskNormal(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestPolicy(t, tmpDir)
	policy, _ := LoadIntakePolicy(tmpDir)

	result := ClassifyTask("fix bug in formatter", []string{"src/formatter.go"}, "", policy)
	if result.IntakeLabel != IntakeLabelNormal {
		t.Fatalf("expected normal, got %s", result.IntakeLabel)
	}
	if result.RuntimeTier != RuntimeTierStandard {
		t.Fatalf("expected standard, got %s", result.RuntimeTier)
	}
}

func TestIsRuntimeTier(t *testing.T) {
	if !IsRuntimeTier("light") {
		t.Fatal("expected light to be valid")
	}
	if !IsRuntimeTier("standard") {
		t.Fatal("expected standard to be valid")
	}
	if !IsRuntimeTier("deep") {
		t.Fatal("expected deep to be valid")
	}
	if IsRuntimeTier("bogus") {
		t.Fatal("expected bogus to be invalid")
	}
}

func TestIsIntakeLabel(t *testing.T) {
	if !IsIntakeLabel("tiny") {
		t.Fatal("expected tiny to be valid")
	}
	if !IsIntakeLabel("normal") {
		t.Fatal("expected normal to be valid")
	}
	if !IsIntakeLabel("high_risk") {
		t.Fatal("expected high_risk to be valid")
	}
	if IsIntakeLabel("bogus") {
		t.Fatal("expected bogus to be invalid")
	}
}

func TestIsTierDowngrade(t *testing.T) {
	if !IsTierDowngrade(RuntimeTierLight, RuntimeTierStandard) {
		t.Fatal("expected light->standard to be downgrade")
	}
	if !IsTierDowngrade(RuntimeTierLight, RuntimeTierDeep) {
		t.Fatal("expected light->deep to be downgrade")
	}
	if !IsTierDowngrade(RuntimeTierStandard, RuntimeTierDeep) {
		t.Fatal("expected standard->deep to be downgrade")
	}
	if IsTierDowngrade(RuntimeTierStandard, RuntimeTierLight) {
		t.Fatal("expected standard->light to not be downgrade")
	}
}

func TestHasApprovedTierDowngradeIntervention(t *testing.T) {
	if HasApprovedTierDowngradeIntervention(nil) {
		t.Fatal("expected nil governance to be false")
	}
	if HasApprovedTierDowngradeIntervention(map[string]any{"approval_status": "pending"}) {
		t.Fatal("expected pending to be false")
	}
	if !HasApprovedTierDowngradeIntervention(map[string]any{
		"approval_status":       "approved",
		"approval_required_for": []any{"tier_downgrade"},
	}) {
		t.Fatal("expected approved tier_downgrade to be true")
	}
	if !HasApprovedTierDowngradeIntervention(map[string]any{
		"approval_status":       "approved",
		"approval_required_for": []any{"intake_tier_downgrade"},
	}) {
		t.Fatal("expected approved intake_tier_downgrade to be true")
	}
	if !HasApprovedTierDowngradeIntervention(map[string]any{
		"approval_status":       "approved",
		"approval_required_for": []any{"tier downgrade"},
	}) {
		t.Fatal("expected approved 'tier downgrade' to be true")
	}
}

func TestExplainCardIntakeInferred(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestPolicy(t, tmpDir)
	policy, _ := LoadIntakePolicy(tmpDir)

	card := map[string]any{
		"tier": "standard",
		"claim": map[string]any{
			"summary": "routine fix",
		},
	}

	explanation := ExplainCardIntake(card, policy)
	if !explanation.OK {
		t.Fatalf("expected ok, got errors: %v", explanation.Errors)
	}
	if explanation.Source != "inferred" {
		t.Fatalf("expected inferred source, got %s", explanation.Source)
	}
	if explanation.IntakeLabel != IntakeLabelNormal {
		t.Fatalf("expected normal, got %s", explanation.IntakeLabel)
	}
}

func TestExplainCardIntakeDeclared(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestPolicy(t, tmpDir)
	policy, _ := LoadIntakePolicy(tmpDir)

	card := map[string]any{
		"tier": "standard",
		"intake": map[string]any{
			"classification": "normal",
			"mapped_tier":    "standard",
			"rationale":      "routine fix",
		},
	}

	explanation := ExplainCardIntake(card, policy)
	if !explanation.OK {
		t.Fatalf("expected ok, got errors: %v", explanation.Errors)
	}
	if explanation.Source != "declared" {
		t.Fatalf("expected declared source, got %s", explanation.Source)
	}
}

func TestExplainCardIntakeDowngradeError(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestPolicy(t, tmpDir)
	policy, _ := LoadIntakePolicy(tmpDir)

	card := map[string]any{
		"tier": "light",
		"claim": map[string]any{
			"summary": "update auth logic",
		},
	}

	explanation := ExplainCardIntake(card, policy)
	if explanation.OK {
		t.Fatal("expected not ok for downgrade without intervention")
	}
	if !explanation.TierDowngrade {
		t.Fatal("expected tier_downgrade")
	}
	if !explanation.InterventionRequired {
		t.Fatal("expected intervention_required")
	}
}

func TestExplainCardIntakeDowngradeApproved(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestPolicy(t, tmpDir)
	policy, _ := LoadIntakePolicy(tmpDir)

	card := map[string]any{
		"tier": "light",
		"claim": map[string]any{
			"summary": "update auth logic",
		},
		"governance": map[string]any{
			"approval_status":       "approved",
			"approval_required_for": []any{"tier_downgrade"},
		},
	}

	explanation := ExplainCardIntake(card, policy)
	if !explanation.OK {
		t.Fatalf("expected ok with approved intervention, got errors: %v", explanation.Errors)
	}
	if !explanation.InterventionApproved {
		t.Fatal("expected intervention_approved")
	}
}

func TestExplainCardIntakeInvalidDeclaredLabel(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestPolicy(t, tmpDir)
	policy, _ := LoadIntakePolicy(tmpDir)

	card := map[string]any{
		"tier": "standard",
		"intake": map[string]any{
			"classification": "bogus",
			"mapped_tier":    "standard",
		},
	}

	explanation := ExplainCardIntake(card, policy)
	if explanation.OK {
		t.Fatal("expected not ok for invalid label")
	}
	found := false
	for _, e := range explanation.Errors {
		if e == "intake.classification must be tiny, normal, or high_risk" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected classification error, got: %v", explanation.Errors)
	}
}

func TestExplainCardIntakeMismappedTier(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestPolicy(t, tmpDir)
	policy, _ := LoadIntakePolicy(tmpDir)

	card := map[string]any{
		"tier": "standard",
		"intake": map[string]any{
			"classification": "normal",
			"mapped_tier":    "deep",
		},
	}

	explanation := ExplainCardIntake(card, policy)
	if explanation.OK {
		t.Fatal("expected not ok for mismapped tier")
	}
	found := false
	for _, e := range explanation.Errors {
		if strings.Contains(e, "does not match policy tier") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected policy tier mismatch error, got: %v", explanation.Errors)
	}
}
