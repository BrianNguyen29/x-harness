package intake

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseMarkdownTitleDefaultsIDAndGoal(t *testing.T) {
	md := "# Foo Bar\n"
	spec, err := ParseMarkdown(md)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spec.Title != "Foo Bar" {
		t.Fatalf("expected title=foo bar, got %q", spec.Title)
	}
	if spec.ID != "foo-bar" {
		t.Fatalf("expected id=foo-bar, got %q", spec.ID)
	}
	if spec.ProductGoal != "Foo Bar" {
		t.Fatalf("expected product_goal=Foo Bar, got %q", spec.ProductGoal)
	}
}

func TestParseMarkdownGoalSectionOverridesTitleGoal(t *testing.T) {
	md := "# Foo Bar\n\n## Goal\nFirst goal paragraph\n"
	spec, err := ParseMarkdown(md)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spec.ID != "foo-bar" {
		t.Fatalf("expected id=foo-bar, got %q", spec.ID)
	}
	if spec.ProductGoal != "First goal paragraph" {
		t.Fatalf("expected product_goal=First goal paragraph, got %q", spec.ProductGoal)
	}
}

func TestParseMarkdownAcceptanceListItems(t *testing.T) {
	md := "# Title\n\n## Acceptance\n- [ ] first\n- [x] second\n* third\n1. fourth\n"
	spec, err := ParseMarkdown(md)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"first", "second", "third", "fourth"}
	if !reflect.DeepEqual(spec.Acceptance, want) {
		t.Fatalf("expected acceptance=%v, got %v", want, spec.Acceptance)
	}
}

func TestParseMarkdownAcceptanceCriteriaAndCriteriaAliases(t *testing.T) {
	for _, heading := range []string{"Acceptance", "Acceptance Criteria", "Criteria"} {
		md := "# T\n\n## " + heading + "\n- a\n- b\n"
		spec, err := ParseMarkdown(md)
		if err != nil {
			t.Fatalf("heading %q: unexpected error: %v", heading, err)
		}
		if !reflect.DeepEqual(spec.Acceptance, []string{"a", "b"}) {
			t.Fatalf("heading %q: expected [a b], got %v", heading, spec.Acceptance)
		}
	}
}

func TestParseMarkdownNonGoalsAndOutOfScope(t *testing.T) {
	for _, heading := range []string{"Non-Goals", "Non Goals", "Out of Scope"} {
		md := "# T\n\n## " + heading + "\n- a\n- b\n"
		spec, err := ParseMarkdown(md)
		if err != nil {
			t.Fatalf("heading %q: unexpected error: %v", heading, err)
		}
		if !reflect.DeepEqual(spec.NonGoals, []string{"a", "b"}) {
			t.Fatalf("heading %q: expected [a b], got %v", heading, spec.NonGoals)
		}
	}
}

func TestParseMarkdownProtectedBehaviorAliases(t *testing.T) {
	for _, heading := range []string{"Protected Behavior", "Protected Behaviors"} {
		md := "# T\n\n## " + heading + "\n- a\n"
		spec, err := ParseMarkdown(md)
		if err != nil {
			t.Fatalf("heading %q: unexpected error: %v", heading, err)
		}
		if !reflect.DeepEqual(spec.ProtectedBehaviors, []string{"a"}) {
			t.Fatalf("heading %q: expected [a], got %v", heading, spec.ProtectedBehaviors)
		}
	}
}

func TestParseMarkdownAmbiguitySetsStatus(t *testing.T) {
	for _, heading := range []string{"Ambiguity", "Open Questions", "Unresolved Questions"} {
		md := "# T\n\n## " + heading + "\n- q1\n- q2\n"
		spec, err := ParseMarkdown(md)
		if err != nil {
			t.Fatalf("heading %q: unexpected error: %v", heading, err)
		}
		if !spec.AmbiguitySet {
			t.Fatalf("heading %q: expected AmbiguitySet true", heading)
		}
		if !reflect.DeepEqual(spec.AmbiguityQuestions, []string{"q1", "q2"}) {
			t.Fatalf("heading %q: expected [q1 q2], got %v", heading, spec.AmbiguityQuestions)
		}
	}
}

func TestParseMarkdownUserVisibleChange(t *testing.T) {
	cases := map[string]bool{
		"true":  true,
		"yes":   true,
		"1":     true,
		"false": false,
		"no":    false,
		"0":     false,
	}
	for raw, want := range cases {
		md := "# T\n\n## User-Visible Change\n" + raw + "\n"
		spec, err := ParseMarkdown(md)
		if err != nil {
			t.Fatalf("raw %q: unexpected error: %v", raw, err)
		}
		if spec.UserVisibleChange == nil || *spec.UserVisibleChange != want {
			t.Fatalf("raw %q: expected %v, got %v", raw, want, spec.UserVisibleChange)
		}
	}
}

func TestParseMarkdownUserVisibleChangeInvalid(t *testing.T) {
	md := "# T\n\n## User-Visible Change\nmaybe\n"
	_, err := ParseMarkdown(md)
	if err == nil {
		t.Fatalf("expected error for invalid bool, got nil")
	}
	if !strings.Contains(err.Error(), "user-visible change") {
		t.Fatalf("expected error to mention user-visible change, got %q", err.Error())
	}
}

func TestParseMarkdownNotesContextProblemJoined(t *testing.T) {
	for _, heading := range []string{"Notes", "Context", "Problem"} {
		md := "# T\n\n## " + heading + "\nfirst line\nsecond line\n"
		spec, err := ParseMarkdown(md)
		if err != nil {
			t.Fatalf("heading %q: unexpected error: %v", heading, err)
		}
		want := "first line\nsecond line"
		if spec.Notes != want {
			t.Fatalf("heading %q: expected %q, got %q", heading, want, spec.Notes)
		}
	}
}

func TestParseMarkdownUnknownSectionSkipped(t *testing.T) {
	md := "# T\n\n## Mystery\nbody\n\n## Acceptance\n- a\n"
	spec, err := ParseMarkdown(md)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(spec.Acceptance, []string{"a"}) {
		t.Fatalf("expected acceptance=[a], got %v", spec.Acceptance)
	}
	if spec.Notes != "" {
		t.Fatalf("expected notes empty, got %q", spec.Notes)
	}
}

func TestParseMarkdownCaseInsensitiveHeadings(t *testing.T) {
	md := "# Title\n\n## GOAL\nmy goal\n\n## acceptance\n- a\n"
	spec, err := ParseMarkdown(md)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spec.ProductGoal != "my goal" {
		t.Fatalf("expected product_goal=my goal, got %q", spec.ProductGoal)
	}
	if !reflect.DeepEqual(spec.Acceptance, []string{"a"}) {
		t.Fatalf("expected acceptance=[a], got %v", spec.Acceptance)
	}
}

func TestParseMarkdownEmpty(t *testing.T) {
	spec, err := ParseMarkdown("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spec.Title != "" || spec.ID != "" || spec.ProductGoal != "" {
		t.Fatalf("expected empty spec, got %+v", spec)
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Foo Bar":      "foo-bar",
		"  Trim Me!  ": "trim-me",
		"Multi  Space": "multi-space",
		"v1.0 release": "v1-0-release",
		"":             "",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Fatalf("slugify(%q): want %q, got %q", in, want, got)
		}
	}
}
