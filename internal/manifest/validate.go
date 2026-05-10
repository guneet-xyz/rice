package manifest

import (
	"fmt"
	"strings"
)

// Validate checks that a Manifest conforms to all schema rules.
// Returns an error if any rule is violated.
func Validate(m *Manifest) error {
	if m.SchemaVersion != 1 {
		return fmt.Errorf("unsupported schema_version: %d", m.SchemaVersion)
	}

	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("name is required and must not be empty")
	}

	if len(m.SupportedOS) == 0 {
		return fmt.Errorf("supported_os must not be empty")
	}

	validOS := map[string]bool{"linux": true, "darwin": true, "windows": true}
	for _, os := range m.SupportedOS {
		if !validOS[os] {
			return fmt.Errorf("unsupported OS: %q (must be one of: linux, darwin, windows)", os)
		}
	}

	if len(m.Profiles) == 0 {
		return fmt.Errorf("at least one profile must be defined")
	}

	for profileName, profileDef := range m.Profiles {
		if len(profileDef.Sources) == 0 {
			return fmt.Errorf("profile %q has no sources", profileName)
		}

		for i, source := range profileDef.Sources {
			if strings.HasPrefix(source.Path, "/") {
				return fmt.Errorf("profile %q source[%d]: %q must be a relative path (no leading /)", profileName, i, source.Path)
			}

			if strings.Contains(source.Path, "..") {
				return fmt.Errorf("profile %q source[%d]: %q must not contain .. segments", profileName, i, source.Path)
			}

			switch source.Mode {
			case "", "file":
				if source.Target != "" {
					return fmt.Errorf("source %q: target field is only valid for folder-mode", source.Path)
				}
			case "folder":
				if source.Target == "" {
					return fmt.Errorf("source %q: folder-mode requires a non-empty target field", source.Path)
				}
			default:
				return fmt.Errorf("source %q: unknown mode %q", source.Path, source.Mode)
			}
		}

		seen := make(map[string]bool)
		for i, source := range profileDef.Sources {
			if seen[source.Path] {
				return fmt.Errorf("profile %q has duplicate source at index %d: %q", profileName, i, source.Path)
			}
			seen[source.Path] = true
		}
	}

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
