package profile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guneet/rice/internal/manifest"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		name        string
		manifest    *manifest.Manifest
		profileName string
		want        []string
		wantErr     bool
		errContains []string
	}{
		{
			name: "single source",
			manifest: &manifest.Manifest{
				Name: "nvim",
				Profiles: map[string]manifest.ProfileDef{
					"default": {
						Sources: []manifest.SourceSpec{{Path: "."}},
					},
				},
			},
			profileName: "default",
			want:        []string{"."},
			wantErr:     false,
		},
		{
			name: "multiple sources in order",
			manifest: &manifest.Manifest{
				Name: "ghostty",
				Profiles: map[string]manifest.ProfileDef{
					"macbook": {
						Sources: []manifest.SourceSpec{{Path: "common"}, {Path: "macbook"}},
					},
				},
			},
			profileName: "macbook",
			want:        []string{"common", "macbook"},
			wantErr:     false,
		},
		{
			name: "unknown profile with available profiles",
			manifest: &manifest.Manifest{
				Name: "nvim",
				Profiles: map[string]manifest.ProfileDef{
					"default": {
						Sources: []manifest.SourceSpec{{Path: "."}},
					},
					"minimal": {
						Sources: []manifest.SourceSpec{{Path: "minimal"}},
					},
				},
			},
			profileName: "unknown",
			want:        nil,
			wantErr:     true,
			errContains: []string{"unknown", "nvim", "default", "minimal"},
		},
		{
			name: "unknown profile with empty profiles map",
			manifest: &manifest.Manifest{
				Name:     "empty",
				Profiles: map[string]manifest.ProfileDef{},
			},
			profileName: "any",
			want:        nil,
			wantErr:     true,
			errContains: []string{"any", "empty"},
		},
		{
			name: "preserves source order",
			manifest: &manifest.Manifest{
				Name: "zsh",
				Profiles: map[string]manifest.ProfileDef{
					"work": {
						Sources: []manifest.SourceSpec{{Path: "base"}, {Path: "work"}, {Path: "secrets"}},
					},
				},
			},
			profileName: "work",
			want:        []string{"base", "work", "secrets"},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Resolve(tt.manifest, tt.profileName)

			if tt.wantErr {
				require.Error(t, err)
				errMsg := err.Error()
				for _, substr := range tt.errContains {
					assert.Contains(t, errMsg, substr, "error message should contain %q", substr)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
