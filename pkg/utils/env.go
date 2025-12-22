package utils

import (
	"os"
	"strconv"
)

// GetEnv retrieves an environment variable;
// return default variable when missing.
func GetEnv(key, defKey string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defKey
}

func GetIntEnv(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed
		}
	}
	return def
}
