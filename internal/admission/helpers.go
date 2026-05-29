package admission

import "strings"

func isCompletionCardShape(doc map[string]any) bool {
	if _, ok := doc["schema_version"]; ok {
		return true
	}
	if _, ok := doc["task_id"]; ok {
		return true
	}
	if _, ok := doc["admission"]; ok {
		return true
	}
	if _, ok := doc["acceptance_status"]; ok {
		return true
	}
	if claim, ok := doc["claim"].(map[string]any); ok {
		if _, ok := claim["fix_status"]; ok {
			if _, ok := claim["summary"]; ok {
				return true
			}
		}
	}
	return false
}

func isRuntimeTier(tier string) bool {
	return tier == "light" || tier == "standard" || tier == "deep"
}

func isTierDowngrade(declared, mapped string) bool {
	tierRank := map[string]int{"light": 1, "standard": 2, "deep": 3}
	return tierRank[declared] < tierRank[mapped]
}

func hasApprovedTierDowngrade(governance map[string]any) bool {
	if governance == nil {
		return false
	}
	approvalStatus := stringInMap(governance, "approval_status")
	return approvalStatus == "approved"
}

func isNonSuccessStatus(status string) bool {
	switch status {
	case "failed", "blocked", "skipped", "timeout", "error":
		return true
	}
	return false
}

func isValidOutcome(outcome string) bool {
	switch outcome {
	case "success", "failed", "blocked", "skipped", "timeout", "error":
		return true
	}
	return false
}

func stringValue(doc map[string]any, key string) string {
	if v, ok := doc[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func mapValue(doc map[string]any, key string) map[string]any {
	if v, ok := doc[key]; ok {
		if m, ok := v.(map[string]any); ok {
			return m
		}
	}
	return nil
}

func boolValue(doc map[string]any, key string) bool {
	if v, ok := doc[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func boolInMap(m map[string]any, key string) bool {
	if m == nil {
		return false
	}
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func stringInMap(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func sliceInMap(m map[string]any, key string) []any {
	if m == nil {
		return nil
	}
	if v, ok := m[key]; ok {
		if s, ok := v.([]any); ok {
			return s
		}
	}
	return nil
}

func verificationArtifactsHaveScope(artifacts []any) bool {
	for _, item := range artifacts {
		artifact, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if len(sliceInMap(artifact, "verifies")) > 0 || len(sliceInMap(artifact, "does_not_verify")) > 0 {
			return true
		}
	}
	return false
}

func intLikeValue(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int8:
		return int(n), true
	case int16:
		return int(n), true
	case int32:
		return int(n), true
	case int64:
		return int(n), true
	case uint:
		return int(n), true
	case uint8:
		return int(n), true
	case uint16:
		return int(n), true
	case uint32:
		return int(n), true
	case uint64:
		return int(n), true
	case float32:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}

func shellMetacharacter(command string) string {
	tokens := []string{"&&", "||", ";", "|", "`", "$(", ">", "<"}
	for _, token := range tokens {
		if strings.Contains(command, token) {
			return token
		}
	}
	return ""
}
