package manifest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		manifest *Manifest
		wantErr bool
		errMsg  string
	}{
		// Rule 1: SchemaVersion must be 1
		{
			name: "valid schema version 1",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []string{"file.txt"}},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid schema version 0",
			manifest: &Manifest{
				SchemaVersion: 0,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []string{"file.txt"}},
				},
			},
			wantErr: true,
			errMsg:  "unsupported schema_version: 0",
		},
		{
			name: "invalid schema version 2",
			manifest: &Manifest{
				SchemaVersion: 2,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []string{"file.txt"}},
				},
			},
			wantErr: true,
			errMsg:  "unsupported schema_version: 2",
		},

		// Rule 2: Name must be non-empty
		{
			name: "valid non-empty name",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "mypackage",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []string{"file.txt"}},
				},
			},
			wantErr: false,
		},
		{
			name: "empty name",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []string{"file.txt"}},
				},
			},
			wantErr: true,
			errMsg:  "name is required and must not be empty",
		},
		{
			name: "whitespace-only name",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "   ",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []string{"file.txt"}},
				},
			},
			wantErr: true,
			errMsg:  "name is required and must not be empty",
		},

		// Rule 3: SupportedOS must be non-empty and each element must be in {linux, darwin, windows}
		{
			name: "valid single OS linux",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []string{"file.txt"}},
				},
			},
			wantErr: false,
		},
		{
			name: "valid single OS darwin",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"darwin"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []string{"file.txt"}},
				},
			},
			wantErr: false,
		},
		{
			name: "valid single OS windows",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"windows"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []string{"file.txt"}},
				},
			},
			wantErr: false,
		},
		{
			name: "valid multiple OS",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux", "darwin", "windows"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []string{"file.txt"}},
				},
			},
			wantErr: false,
		},
		{
			name: "empty supported_os",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []string{"file.txt"}},
				},
			},
			wantErr: true,
			errMsg:  "supported_os must not be empty",
		},
		{
			name: "invalid OS freebsd",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"freebsd"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []string{"file.txt"}},
				},
			},
			wantErr: true,
			errMsg:  "unsupported OS: \"freebsd\"",
		},
		{
			name: "mixed valid and invalid OS",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux", "macos"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []string{"file.txt"}},
				},
			},
			wantErr: true,
			errMsg:  "unsupported OS: \"macos\"",
		},

		// Rule 4: At least one profile must be defined; each profile's Sources must be non-empty
		{
			name: "valid single profile",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []string{"file.txt"}},
				},
			},
			wantErr: false,
		},
		{
			name: "valid multiple profiles",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"personal": {Sources: []string{"file1.txt"}},
					"work":     {Sources: []string{"file2.txt"}},
				},
			},
			wantErr: false,
		},
		{
			name: "no profiles",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles:      map[string]ProfileDef{},
			},
			wantErr: true,
			errMsg:  "at least one profile must be defined",
		},
		{
			name: "profile with empty sources",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []string{}},
				},
			},
			wantErr: true,
			errMsg:  "profile \"default\" has no sources",
		},
		{
			name: "one profile valid, one with empty sources",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"personal": {Sources: []string{"file.txt"}},
					"work":     {Sources: []string{}},
				},
			},
			wantErr: true,
			errMsg:  "profile \"work\" has no sources",
		},

		// Rule 5: Each Sources entry must be a relative path (no leading /, no .. segments)
		{
			name: "valid relative path",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []string{"config/file.txt"}},
				},
			},
			wantErr: false,
		},
		{
			name: "valid nested relative path",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []string{"a/b/c/file.txt"}},
				},
			},
			wantErr: false,
		},
		{
			name: "absolute path with leading slash",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []string{"/etc/config"}},
				},
			},
			wantErr: true,
			errMsg:  "must be a relative path (no leading /)",
		},
		{
			name: "path with .. segment",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []string{"../config/file.txt"}},
				},
			},
			wantErr: true,
			errMsg:  "must not contain .. segments",
		},
		{
			name: "path with .. in middle",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []string{"a/../b/file.txt"}},
				},
			},
			wantErr: true,
			errMsg:  "must not contain .. segments",
		},

		// Rule 6: Sources within a single profile must be unique
		{
			name: "valid unique sources",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []string{"file1.txt", "file2.txt", "file3.txt"}},
				},
			},
			wantErr: false,
		},
		{
			name: "duplicate sources in same profile",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []string{"file.txt", "file.txt"}},
				},
			},
			wantErr: true,
			errMsg:  "has duplicate source",
		},
		{
			name: "duplicate sources with different profiles allowed",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"personal": {Sources: []string{"file.txt"}},
					"work":     {Sources: []string{"file.txt"}},
				},
			},
			wantErr: false,
		},
		{
			name: "multiple duplicates in same profile",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Profiles: map[string]ProfileDef{
					"default": {Sources: []string{"a.txt", "b.txt", "a.txt", "c.txt"}},
				},
			},
			wantErr: true,
			errMsg:  "has duplicate source",
		},

		// Rule 7: Target (if set) must start with $HOME, $XDG_CONFIG_HOME, %USERPROFILE%, or %APPDATA%
		{
			name: "valid target with $HOME",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Target:        "$HOME/.config/myapp",
				Profiles: map[string]ProfileDef{
					"default": {Sources: []string{"file.txt"}},
				},
			},
			wantErr: false,
		},
		{
			name: "valid target with $XDG_CONFIG_HOME",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Target:        "$XDG_CONFIG_HOME/myapp",
				Profiles: map[string]ProfileDef{
					"default": {Sources: []string{"file.txt"}},
				},
			},
			wantErr: false,
		},
		{
			name: "valid target with %USERPROFILE%",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"windows"},
				Target:        "%USERPROFILE%/AppData/Local/myapp",
				Profiles: map[string]ProfileDef{
					"default": {Sources: []string{"file.txt"}},
				},
			},
			wantErr: false,
		},
		{
			name: "valid target with %APPDATA%",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"windows"},
				Target:        "%APPDATA%/myapp",
				Profiles: map[string]ProfileDef{
					"default": {Sources: []string{"file.txt"}},
				},
			},
			wantErr: false,
		},
		{
			name: "empty target is valid",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Target:        "",
				Profiles: map[string]ProfileDef{
					"default": {Sources: []string{"file.txt"}},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid target with /etc",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Target:        "/etc/myapp",
				Profiles: map[string]ProfileDef{
					"default": {Sources: []string{"file.txt"}},
				},
			},
			wantErr: true,
			errMsg:  "must start with one of",
		},
		{
			name: "invalid target with relative path",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "test",
				SupportedOS:   []string{"linux"},
				Target:        "config/myapp",
				Profiles: map[string]ProfileDef{
					"default": {Sources: []string{"file.txt"}},
				},
			},
			wantErr: true,
			errMsg:  "must start with one of",
		},

		// Integration tests: multiple rules
		{
			name: "comprehensive valid manifest",
			manifest: &Manifest{
				SchemaVersion: 1,
				Name:          "nvim",
				Description:   "Neovim configuration",
				SupportedOS:   []string{"linux", "darwin"},
				Target:        "$HOME/.config/nvim",
				ProfileKey:    "nvim_profile",
				Profiles: map[string]ProfileDef{
					"personal": {Sources: []string{"init.lua", "lua/config.lua"}},
					"work":     {Sources: []string{"init.lua", "lua/work.lua"}},
				},
			},
			wantErr: false,
		},
		{
			name: "comprehensive invalid manifest (multiple errors, first caught)",
			manifest: &Manifest{
				SchemaVersion: 2,
				Name:          "",
				SupportedOS:   []string{},
				Profiles:      map[string]ProfileDef{},
			},
			wantErr: true,
			errMsg:  "unsupported schema_version: 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.manifest)
			if tt.wantErr {
				assert.Error(t, err, "expected error but got none")
				assert.Contains(t, err.Error(), tt.errMsg, "error message mismatch")
			} else {
				assert.NoError(t, err, "expected no error but got: %v", err)
			}
		})
	}
}
