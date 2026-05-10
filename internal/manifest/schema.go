package manifest

import "fmt"

// Manifest represents the structure of a rice.toml file.
type Manifest struct {
	SchemaVersion int                   `toml:"schema_version"`
	Name          string                `toml:"name"`
	Description   string                `toml:"description"`
	SupportedOS   []string              `toml:"supported_os"`
	Target        string                `toml:"target"`
	ProfileKey    string                `toml:"profile_key"`
	Profiles      map[string]ProfileDef `toml:"profiles"`
}

// ProfileDef represents a single profile within a Manifest.
type ProfileDef struct {
	Sources []SourceSpec `toml:"sources"`
}

// SourceSpec describes a single source entry under a profile. It accepts two
// TOML forms:
//
//  1. Bare string  : sources = ["common", "macbook"]
//     -> SourceSpec{Path: "common", Mode: "file"}
//
//  2. Inline table : sources = [{path = "foo", mode = "folder", target = ".config/nvim"}]
//     -> SourceSpec{Path: "foo", Mode: "folder", Target: ".config/nvim"}
//
// Mode defaults to "file" when omitted. "folder" mode opts the entire source
// directory into being symlinked as a single directory link rather than walked
// file-by-file; in that case Target is required and is interpreted relative to
// the package's Target.
type SourceSpec struct {
	Path   string `toml:"path"`
	Mode   string `toml:"mode"`
	Target string `toml:"target"`
}

// UnmarshalTOML implements the toml.Unmarshaler interface from
// github.com/BurntSushi/toml. It accepts either a bare string or a table.
func (s *SourceSpec) UnmarshalTOML(data interface{}) error {
	switch v := data.(type) {
	case string:
		s.Path = v
		s.Mode = "file"
		s.Target = ""
		return nil
	case map[string]interface{}:
		if raw, ok := v["path"]; ok {
			str, ok := raw.(string)
			if !ok {
				return fmt.Errorf("source: \"path\" must be a string, got %T", raw)
			}
			s.Path = str
		}
		if raw, ok := v["mode"]; ok {
			str, ok := raw.(string)
			if !ok {
				return fmt.Errorf("source: \"mode\" must be a string, got %T", raw)
			}
			s.Mode = str
		}
		if raw, ok := v["target"]; ok {
			str, ok := raw.(string)
			if !ok {
				return fmt.Errorf("source: \"target\" must be a string, got %T", raw)
			}
			s.Target = str
		}
		if s.Mode == "" {
			s.Mode = "file"
		}
		return nil
	default:
		return fmt.Errorf("source: expected string or table, got %T", data)
	}
}
