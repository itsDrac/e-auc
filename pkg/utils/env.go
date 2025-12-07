package utils

import "os"

// GetEnv retrieves an environment variable;
// return default variable when missing.
func GetEnv(key, defKey string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defKey
}
