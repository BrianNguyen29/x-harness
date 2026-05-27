package evidence

import (
	"strings"
	"testing"
)

func TestRedactText_PrivateKey(t *testing.T) {
	input := `-----BEGIN RSA PRIVATE KEY-----
abc123
-----END RSA PRIVATE KEY-----`
	redacted, patterns, replacements := RedactText(input)
	if !strings.Contains(redacted, "[REDACTED:private_key]") {
		t.Fatalf("expected private key redaction, got: %s", redacted)
	}
	if len(patterns) != 1 || patterns[0] != "private_key" {
		t.Fatalf("expected [private_key], got %v", patterns)
	}
	if replacements != 1 {
		t.Fatalf("expected 1 replacement, got %d", replacements)
	}
}

func TestRedactText_GitHubToken(t *testing.T) {
	input := "token: ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	redacted, patterns, replacements := RedactText(input)
	if !strings.Contains(redacted, "[REDACTED:github_token]") {
		t.Fatalf("expected github_token redaction, got: %s", redacted)
	}
	if replacements != 1 {
		t.Fatalf("expected 1 replacement, got %d", replacements)
	}
	if len(patterns) != 1 || patterns[0] != "github_token" {
		t.Fatalf("expected [github_token], got %v", patterns)
	}
}

func TestRedactText_NpmToken(t *testing.T) {
	input := "npm_xxxxxxxxxxxxxxxxxxxxxxxxxx"
	redacted, patterns, replacements := RedactText(input)
	if !strings.Contains(redacted, "[REDACTED:npm_token]") {
		t.Fatalf("expected npm_token redaction, got: %s", redacted)
	}
	if replacements != 1 {
		t.Fatalf("expected 1 replacement, got %d", replacements)
	}
	if len(patterns) != 1 || patterns[0] != "npm_token" {
		t.Fatalf("expected [npm_token], got %v", patterns)
	}
}

func TestRedactText_BearerToken(t *testing.T) {
	input := "Authorization: Bearer abcdefghijklmnop"
	redacted, patterns, replacements := RedactText(input)
	if !strings.Contains(redacted, "Bearer [REDACTED:bearer_token]") {
		t.Fatalf("expected bearer_token redaction, got: %s", redacted)
	}
	if replacements != 1 {
		t.Fatalf("expected 1 replacement, got %d", replacements)
	}
	if len(patterns) != 1 || patterns[0] != "bearer_token" {
		t.Fatalf("expected [bearer_token], got %v", patterns)
	}
}

func TestRedactText_JWT(t *testing.T) {
	input := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"
	redacted, patterns, replacements := RedactText(input)
	if !strings.Contains(redacted, "[REDACTED:jwt]") {
		t.Fatalf("expected jwt redaction, got: %s", redacted)
	}
	if replacements != 1 {
		t.Fatalf("expected 1 replacement, got %d", replacements)
	}
	if len(patterns) != 1 || patterns[0] != "jwt" {
		t.Fatalf("expected [jwt], got %v", patterns)
	}
}

func TestRedactText_ConnectionString(t *testing.T) {
	input := "DATABASE_URL=postgres://user:pass@localhost/db"
	redacted, patterns, replacements := RedactText(input)
	if !strings.Contains(redacted, "postgres://[REDACTED:connection_string]") {
		t.Fatalf("expected connection_string redaction, got: %s", redacted)
	}
	if replacements != 1 {
		t.Fatalf("expected 1 replacement, got %d", replacements)
	}
	if len(patterns) != 1 || patterns[0] != "connection_string" {
		t.Fatalf("expected [connection_string], got %v", patterns)
	}
}

func TestRedactText_ApiKey(t *testing.T) {
	input := "api_key=abcdef1234567890abcdef123456"
	redacted, patterns, replacements := RedactText(input)
	if !strings.Contains(redacted, "api_key=[REDACTED:api_key]") {
		t.Fatalf("expected api_key redaction, got: %s", redacted)
	}
	if replacements != 1 {
		t.Fatalf("expected 1 replacement, got %d", replacements)
	}
	if len(patterns) != 1 || patterns[0] != "api_key" {
		t.Fatalf("expected [api_key], got %v", patterns)
	}
}

func TestRedactText_PasswordAssignment(t *testing.T) {
	input := "password=secret123"
	redacted, patterns, replacements := RedactText(input)
	if !strings.Contains(redacted, "password=[REDACTED:password_assignment]") {
		t.Fatalf("expected password_assignment redaction, got: %s", redacted)
	}
	if replacements != 1 {
		t.Fatalf("expected 1 replacement, got %d", replacements)
	}
	if len(patterns) != 1 || patterns[0] != "password_assignment" {
		t.Fatalf("expected [password_assignment], got %v", patterns)
	}
}

func TestRedactText_MultiplePatterns(t *testing.T) {
	input := "password=secret123\napi_key=abcdef1234567890abcdef123456"
	redacted, patterns, replacements := RedactText(input)
	if replacements != 2 {
		t.Fatalf("expected 2 replacements, got %d", replacements)
	}
	if len(patterns) != 2 {
		t.Fatalf("expected 2 patterns, got %v", patterns)
	}
	if !strings.Contains(redacted, "[REDACTED:password_assignment]") || !strings.Contains(redacted, "[REDACTED:api_key]") {
		t.Fatalf("expected both redactions, got: %s", redacted)
	}
}

func TestRedactText_NoSecrets(t *testing.T) {
	input := "hello world"
	redacted, patterns, replacements := RedactText(input)
	if redacted != input {
		t.Fatalf("expected unchanged text, got: %s", redacted)
	}
	if len(patterns) != 0 {
		t.Fatalf("expected no patterns, got %v", patterns)
	}
	if replacements != 0 {
		t.Fatalf("expected 0 replacements, got %d", replacements)
	}
}

func TestIsTextFile_TextExtension(t *testing.T) {
	if !IsTextFile("test.txt", []byte("hello")) {
		t.Fatal("expected .txt to be text")
	}
}

func TestIsTextFile_BinaryContent(t *testing.T) {
	if IsTextFile("unknown", []byte{0x00, 0x01, 0x02}) {
		t.Fatal("expected binary content to not be text")
	}
}

func TestIsTextFile_NoNullBytes(t *testing.T) {
	if !IsTextFile("unknown", []byte("hello world")) {
		t.Fatal("expected content without null bytes to be text")
	}
}
