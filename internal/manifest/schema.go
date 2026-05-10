package manifest

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
	Sources []string `toml:"sources"`
}
