package agui_adapter

import "fmt"

// ValidateMessages validates that messages have the required structure
// This is shared across all transport handlers
func ValidateMessages(messages []map[string]interface{}) error {
	for i, msg := range messages {
		if msg == nil {
			return fmt.Errorf("message at index %d is nil", i)
		}

		// Check for required fields
		id, hasID := msg["id"]
		if !hasID || id == nil || id == "" {
			return fmt.Errorf("message at index %d missing required field 'id'", i)
		}

		role, hasRole := msg["role"]
		if !hasRole || role == nil {
			return fmt.Errorf("message at index %d missing required field 'role'", i)
		}

		roleStr, ok := role.(string)
		if !ok {
			return fmt.Errorf("message at index %d has invalid 'role' type (expected string)", i)
		}

		// Validate role value
		validRoles := map[string]bool{
			"user":      true,
			"assistant":  true,
			"system":    true,
			"developer": true,
			"tool":      true,
		}
		if !validRoles[roleStr] {
			return fmt.Errorf("message at index %d has invalid 'role' value: %s", i, roleStr)
		}

		// Check for content field (required for user and assistant messages)
		if roleStr == "user" || roleStr == "assistant" {
			content, hasContent := msg["content"]
			if !hasContent || content == nil {
				return fmt.Errorf("message at index %d missing required field 'content' for role '%s'", i, roleStr)
			}

			// Content should be a string or array
			if _, ok := content.(string); !ok {
				if _, ok := content.([]interface{}); !ok {
					return fmt.Errorf("message at index %d has invalid 'content' type (expected string or array)", i)
				}
			}
		}
	}

	return nil
}

