package validation

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	// MaxStringLength is the maximum allowed length for string inputs
	MaxStringLength = 10000
	// MaxStderrLines is the maximum number of stderr lines to collect
	MaxStderrLines = 1000
	// MaxStderrLineLength is the maximum length of a single stderr line
	MaxStderrLineLength = 1000
	// MaxJSONSize is the maximum size of JSON input in bytes
	MaxJSONSize = 10 * 1024 * 1024 // 10MB
)

// AllowedModels contains the list of valid model names
var AllowedModels = map[string]bool{
	// Claude 3 models
	"claude-3-opus-20240229":     true,
	"claude-3-sonnet-20240229":   true,
	"claude-3-haiku-20240307":    true,
	
	// Claude 3.5 models
	"claude-3-5-sonnet-20241022": true,
	"claude-3-5-haiku-20241022":  true,
	
	// Allow any string starting with "claude-" for future compatibility
	// The validation will be handled by the CLI itself
}

// shellMetacharacters contains characters that have special meaning in shells
// Including . and / to prevent path traversal attempts
var shellMetacharacters = regexp.MustCompile(`[;&|<>$` + "`" + `\\'"()\[\]{}*?!~\s./]`)

// SanitizeString validates and sanitizes a string input
func SanitizeString(input string, maxLength int) (string, error) {
	if maxLength <= 0 {
		maxLength = MaxStringLength
	}
	
	if len(input) > maxLength {
		return "", fmt.Errorf("input exceeds maximum length of %d characters", maxLength)
	}
	
	// Remove null bytes and other control characters
	input = strings.ReplaceAll(input, "\x00", "")
	
	// Trim whitespace
	input = strings.TrimSpace(input)
	
	return input, nil
}

// SanitizeCommandArg sanitizes a string to be safe for use as a command argument
func SanitizeCommandArg(input string) (string, error) {
	// First apply general string sanitization
	sanitized, err := SanitizeString(input, MaxStringLength)
	if err != nil {
		return "", err
	}
	
	// Check for shell metacharacters
	if shellMetacharacters.MatchString(sanitized) {
		// For safety, reject inputs with shell metacharacters
		// In a production system, you might want to escape these instead
		return "", fmt.Errorf("input contains shell metacharacters")
	}
	
	return sanitized, nil
}

// ValidateModel checks if the model name is valid
func ValidateModel(model string) error {
	if model == "" {
		return nil // Empty is allowed (will use default)
	}
	
	// Check if it's in the known list
	if AllowedModels[model] {
		return nil
	}
	
	// Allow any model starting with "claude-" for future compatibility
	if strings.HasPrefix(model, "claude-") {
		return nil
	}
	
	return fmt.Errorf("invalid model: %s (must start with 'claude-')", model)
}

// ValidatePath validates and cleans a file path
func ValidatePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}
	
	// Clean the path to remove any .. or . components
	cleaned := filepath.Clean(path)
	
	// Check for path traversal attempts
	if strings.Contains(cleaned, "..") {
		return "", fmt.Errorf("path traversal detected")
	}
	
	// Ensure the path is absolute
	if !filepath.IsAbs(cleaned) {
		var err error
		cleaned, err = filepath.Abs(cleaned)
		if err != nil {
			return "", fmt.Errorf("failed to resolve absolute path: %w", err)
		}
	}
	
	return cleaned, nil
}

// ValidateWorkingDirectory validates a working directory path
func ValidateWorkingDirectory(dir string) (string, error) {
	if dir == "" {
		return "", nil // Empty is allowed
	}
	
	return ValidatePath(dir)
}

// TruncateError sanitizes error messages to prevent information disclosure
func TruncateError(err error, maxLength int) string {
	if err == nil {
		return ""
	}
	
	msg := err.Error()
	
	// Remove any file paths that might expose system information
	// This is a simple implementation - in production you might want more sophisticated filtering
	pathPattern := regexp.MustCompile(`(/[^\s]+|[A-Za-z]:\\[^\s]+)`)
	msg = pathPattern.ReplaceAllString(msg, "[path]")
	
	if len(msg) > maxLength {
		msg = msg[:maxLength] + "..."
	}
	
	return msg
}

// FilterEnvironment filters environment variables to only include safe ones
func FilterEnvironment(env []string) []string {
	// Define a list of safe environment variable prefixes
	safeEnvPrefixes := []string{
		"CLAUDE_",
		"LANG",
		"LC_",
		"TZ",
		"TERM",
		"USER",
		"HOME",
		"PATH",
		"TMPDIR",
		"TEMP",
		"TMP",
	}
	
	// Define a list of explicitly blocked environment variables
	blockedEnv := map[string]bool{
		"AWS_SECRET_ACCESS_KEY": true,
		"AWS_SESSION_TOKEN":     true,
		"GITHUB_TOKEN":          true,
		"NPM_TOKEN":             true,
		"ANTHROPIC_API_KEY":     true,
		// Add more sensitive variables as needed
	}
	
	filtered := make([]string, 0, len(env))
	
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) != 2 {
			continue
		}
		
		key := parts[0]
		
		// Skip blocked variables
		if blockedEnv[key] {
			continue
		}
		
		// Check if it matches any safe prefix
		safe := false
		for _, prefix := range safeEnvPrefixes {
			if strings.HasPrefix(key, prefix) {
				safe = true
				break
			}
		}
		
		if safe {
			filtered = append(filtered, e)
		}
	}
	
	return filtered
}