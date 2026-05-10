package profile

import (
	"fmt"
	"sort"
	"strings"

	"github.com/guneet/rice/internal/manifest"
)

// Resolve returns the ordered list of source subdirectory paths for the given profile name.
// Returns error if the profile name is not defined in the manifest.
func Resolve(m *manifest.Manifest, profileName string) ([]string, error) {
	profile, exists := m.Profiles[profileName]
	if !exists {
		// Build sorted list of available profiles
		available := make([]string, 0, len(m.Profiles))
		for name := range m.Profiles {
			available = append(available, name)
		}
		sort.Strings(available)

		availableStr := strings.Join(available, ", ")
		return nil, fmt.Errorf("profile %q not defined in package %q; available: %s", profileName, m.Name, availableStr)
	}

	return profile.Sources, nil
}
