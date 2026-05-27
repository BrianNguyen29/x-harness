package permissions

import "fmt"

// TestFixture represents a built-in test scenario
type TestFixture struct {
	Name       string
	Role       string
	Tier       string
	Command    string
	Capability string
	Expected   string
}

// FixtureResult represents the result of running a fixture
type FixtureResult struct {
	Name           string             `json:"name"`
	OK             bool               `json:"ok"`
	ExpectedStatus string             `json:"expected_status"`
	ActualStatus   string             `json:"actual_status"`
	Decision       PermissionDecision `json:"decision"`
}

// FixturesResult represents the overall fixture run result
type FixturesResult struct {
	OK       bool            `json:"ok"`
	Fixtures []FixtureResult `json:"fixtures"`
}

// RunFixtures runs all 5 built-in test fixtures
func RunFixtures(policy *PermissionsPolicy, root string) (*FixturesResult, error) {
	fixtures := []TestFixture{
		{
			Name:     "worker_safe_test_allowed",
			Role:     "worker",
			Tier:     "standard",
			Command:  "npm test",
			Expected: "allowed",
		},
		{
			Name:     "dangerous_command_denied",
			Role:     "worker",
			Tier:     "deep",
			Command:  "rm -rf dist",
			Expected: "denied",
		},
		{
			Name:     "chained_command_denied",
			Role:     "worker",
			Tier:     "standard",
			Command:  "npm test && node scripts/mutate.js",
			Expected: "denied",
		},
		{
			Name:       "verifier_write_source_denied",
			Role:       "verifier",
			Tier:       "deep",
			Capability: "write_source",
			Expected:   "denied",
		},
		{
			Name:       "deep_dependency_install_requires_intervention",
			Role:       "worker",
			Tier:       "deep",
			Capability: "dependency_install",
			Expected:   "requires_intervention",
		},
	}

	results := make([]FixtureResult, 0, len(fixtures))
	for _, fixture := range fixtures {
		decision, err := CheckPermission(policy, root, fixture.Role, fixture.Tier, fixture.Command, fixture.Capability, "")
		if err != nil {
			return nil, fmt.Errorf("fixture %s failed: %w", fixture.Name, err)
		}
		results = append(results, FixtureResult{
			Name:           fixture.Name,
			OK:             decision.Status == fixture.Expected,
			ExpectedStatus: fixture.Expected,
			ActualStatus:   decision.Status,
			Decision:       decision,
		})
	}

	ok := true
	for _, r := range results {
		if !r.OK {
			ok = false
			break
		}
	}

	return &FixturesResult{
		OK:       ok,
		Fixtures: results,
	}, nil
}
