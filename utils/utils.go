package utils

// --- Helper functions ---
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		}
	}
	return 0
}
func getFloatPtr(m map[string]interface{}, key string) *float64 {
	if val, ok := m[key]; ok {
		if f, ok := val.(float64); ok {
			return &f
		}
		if i, ok := val.(int); ok {
			f := float64(i)
			return &f
		}
	}
	return nil
}

func getInt64Ptr(m map[string]interface{}, key string) *int64 {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case int:
			i64 := int64(v)
			return &i64
		case int64:
			return &v
		case float64:
			i64 := int64(v)
			return &i64
		}
	}
	return nil
}