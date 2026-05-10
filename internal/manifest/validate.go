package manifest

import (
	"fmt"
	"strings"
)

// Validate checks that a Manifest conforms to all schema rules.
// Returns an error if any rule is violated.
func Validate(m *Manifest) error {
	// Rule 1: SchemaVersion must be 1
	if m.SchemaVersion != 1 {
		return fmt.Errorf("unsupported schema_version: %d", m.SchemaVersion)
	}

	// Rule 2: Name must be non-empty
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("name is required and must not be empty")
	}

	// Rule 3: SupportedOS must be non-empty and each element must be in {linux, darwin, windows}
	if len(m.SupportedOS) == 0 {
		return fmt.Errorf("supported_os must not be empty")
	}

	validOS := map[string]bool{"linux": true, "darwin": true, "windows": true}
	for _, os := range m.SupportedOS {
		if !validOS[os] {
			return fmt.Errorf("unsupported OS: %q (must be one of: linux, darwin, windows)", os)
		}
	}

	// Rule 4: At least one profile must be defined; each profile's Sources must be non-empty
	if len(m.Profiles) == 0 {
		return fmt.Errorf("at least one profile must be defined")
	}

	for profileName, profileDef := range m.Profiles {
		if len(profileDef.Sources) == 0 {
			return fmt.Errorf("profile %q has no sources", profileName)
		}

		// Rule 5: Each Sources entry must be a relative path (no leading /, no .. segments)
		for i, source := range profileDef.Sources {
			if strings.HasPrefix(source, "/") {
				return fmt.Errorf("profile %q source[%d]: %q must be a relative path (no leading /)", profileName, i, source)
			}

			// Check for .. segments in the original source string
			if strings.Contains(source, "..") {
				return fmt.Errorf("profile %q source[%d]: %q must not contain .. segments", profileName, i, source)
			}
		}

		// Rule 6: Sources within a single profile must be unique
		seen := make(map[string]bool)
		for i, source := range profileDef.Sources {
			if seen[source] {
				return fmt.Errorf("profile %q has duplicate source at index %d: %q", profileName, i, source)
			}
			seen[source] = true
		}
	}

	// Rule 7: Target (if set) must start with $HOME, $XDG_CONFIG_HOME, %USERPROFILE%, or %APPDATA%
	if m.Target != "" {
		validPrefixes := []string{"$HOME", "$XDG_CONFIG_HOME", "%USERPROFILE%", "%APPDATA%"}
		isValid := false
		for _, prefix := range validPrefixes {
			if strings.HasPrefix(m.Target, prefix) {
				isValid = true
				break
			}
		}
		if !isValid {
			return fmt.Errorf("target %q must start with one of: $HOME, $XDG_CONFIG_HOME, %%USERPROFILE%%, %%APPDATA%%", m.Target)
		}
	}

	return nil
}
