package chat

import (
	"bytes"
	"encoding/json"
	"strings"
)

const redactedValue = "[REDACTED]"

func redactSensitiveJSON(raw json.RawMessage) json.RawMessage {
	if len(bytes.TrimSpace(raw)) == 0 {
		return raw
	}

	var payload any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return json.RawMessage(`"[REDACTED_INVALID_JSON]"`)
	}

	redacted := redactSensitiveValue(payload)
	out, err := json.Marshal(redacted)
	if err != nil {
		return json.RawMessage(`"[REDACTED_UNMARSHALABLE_JSON]"`)
	}

	return out
}

func redactSensitiveValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		redacted := make(map[string]any, len(typed))
		for key, nested := range typed {
			if isSensitiveKey(key) {
				redacted[key] = redactedValue
				continue
			}
			redacted[key] = redactSensitiveValue(nested)
		}
		return redacted
	case []any:
		redacted := make([]any, 0, len(typed))
		for _, nested := range typed {
			redacted = append(redacted, redactSensitiveValue(nested))
		}
		return redacted
	case string:
		if isSensitiveString(typed) {
			return redactedValue
		}
		return typed
	default:
		return typed
	}
}

func isSensitiveKey(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(key, "-", "_"), " ", "_"))
	return normalized == "authorization" ||
		normalized == "api_key" ||
		normalized == "apikey" ||
		normalized == "access_token" ||
		normalized == "refresh_token" ||
		normalized == "password" ||
		normalized == "passwd" ||
		normalized == "secret" ||
		normalized == "dsn" ||
		normalized == "data_source_name"
}

func isSensitiveString(value string) bool {
	normalized := strings.ToLower(value)
	return strings.Contains(normalized, "postgres://") ||
		strings.Contains(normalized, "postgresql://") ||
		strings.Contains(normalized, "mysql://")
}
