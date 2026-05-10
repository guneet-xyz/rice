package installer

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/guneet/rice/internal/logger"
	"github.com/guneet/rice/internal/manifest"
	"github.com/guneet/rice/internal/plan"
	"github.com/guneet/rice/internal/profile"
	"github.com/guneet/rice/internal/state"
	"github.com/guneet/rice/internal/symlink"
)

// InstallRequest captures all inputs needed to compute and execute an install.
type InstallRequest struct {
	RepoRoot    string
	PackageName string
	Profile     string
	CurrentOS   string
	HomeDir     string
	StatePath   string
}

// InstallResult is returned from a successful (or partial) execution.
type InstallResult struct {
	LinksCreated []state.InstalledLink
}

// expandHome replaces leading $HOME or %USERPROFILE% in target with home.
// Returns the path unchanged if no placeholder is present.
func expandHome(target, home string) string {
	if strings.HasPrefix(target, "$HOME") {
		return filepath.Join(home, strings.TrimPrefix(target, "$HOME"))
	}
	if strings.HasPrefix(target, "%USERPROFILE%") {
		return filepath.Join(home, strings.TrimPrefix(target, "%USERPROFILE%"))
	}
	return target
}

// withinHome reports whether target is contained in home (defense in depth).
func withinHome(target, home string) bool {
	absHome, err := filepath.Abs(home)
	if err != nil {
		return false
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absHome, absTarget)
	if err != nil {
		return false
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return false
	}
	return true
}

// BuildInstallPlan computes the plan WITHOUT touching the filesystem (other than reads).
// On conflicts, returns the plan AND an error so callers can render details.
func BuildInstallPlan(req InstallRequest) (*plan.Plan, error) {
	logger.Debug("BuildInstallPlan: start",
		zap.String("repoRoot", req.RepoRoot),
		zap.String("package", req.PackageName),
		zap.String("profile", req.Profile),
		zap.String("os", req.CurrentOS),
	)

	// 1. Discover manifests
	manifests, err := manifest.Discover(req.RepoRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to discover manifests: %w", err)
	}

	// 2. Find the requested package
	m, ok := manifests[req.PackageName]
	if !ok {
		return nil, fmt.Errorf("package %q not found in %s", req.PackageName, req.RepoRoot)
	}

	// 3. OS gate
	if err := manifest.CheckOS(m, req.CurrentOS); err != nil {
		return nil, err
	}

	// 4. Resolve profile to source dir list
	sources, err := profile.Resolve(m, req.Profile)
	if err != nil {
		return nil, err
	}

	// Determine target root (default to HomeDir if Target empty)
	targetRoot := req.HomeDir
	if t := strings.TrimSpace(m.Target); t != "" {
		targetRoot = expandHome(t, req.HomeDir)
	}

	logger.Info("Building install plan",
		zap.String("package", req.PackageName),
		zap.String("profile", req.Profile),
		zap.Strings("sources", sources),
	)

	// 5. Walk each source dir and build planned links.
	// Later sources OVERRIDE earlier ones (last wins) for the same relative path.
	type pendingOp struct {
		Source string
		Target string
	}
	indexByTarget := make(map[string]int)
	var ops []pendingOp

	for _, sourceName := range sources {
		sourceDir := filepath.Join(req.RepoRoot, req.PackageName, sourceName)
		fi, err := os.Stat(sourceDir)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil, fmt.Errorf("source directory %q does not exist for package %q", sourceName, req.PackageName)
			}
			return nil, fmt.Errorf("failed to stat source dir %q: %w", sourceDir, err)
		}
		if !fi.IsDir() {
			return nil, fmt.Errorf("source %q is not a directory", sourceDir)
		}

		walkErr := filepath.WalkDir(sourceDir, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			// Skip rice.toml files anywhere in tree
			if d.Name() == "rice.toml" {
				logger.Warn("Skipping rice.toml in source tree", zap.String("path", path))
				return nil
			}
			// Skip symlinks (we only manage real files)
			if d.Type()&fs.ModeSymlink != 0 {
				logger.Warn("Skipping symlink in source tree", zap.String("path", path))
				return nil
			}

			rel, err := filepath.Rel(sourceDir, path)
			if err != nil {
				return fmt.Errorf("failed to compute relative path: %w", err)
			}

			target := filepath.Join(targetRoot, rel)

			// Defense in depth: ensure target is within HomeDir
			if !withinHome(target, req.HomeDir) {
				return fmt.Errorf("target %q escapes home directory %q", target, req.HomeDir)
			}

			absSource, err := filepath.Abs(path)
			if err != nil {
				return fmt.Errorf("failed to abs source: %w", err)
			}

			logger.Debug("planned op",
				zap.String("source", absSource),
				zap.String("target", target),
			)

			if idx, exists := indexByTarget[target]; exists {
				// Override: later source wins
				ops[idx] = pendingOp{Source: absSource, Target: target}
			} else {
				indexByTarget[target] = len(ops)
				ops = append(ops, pendingOp{Source: absSource, Target: target})
			}
			return nil
		})
		if walkErr != nil {
			return nil, fmt.Errorf("failed to walk source %q: %w", sourceName, walkErr)
		}
	}

	// 6. Build planned-links list for conflict detection
	planned := make([]PlannedLink, 0, len(ops))
	for _, op := range ops {
		planned = append(planned, PlannedLink{Source: op.Source, Target: op.Target})
	}
	conflicts := DetectConflicts(planned, nil)

	// 7. Build plan
	p := &plan.Plan{
		PackageName: req.PackageName,
		Profile:     req.Profile,
	}
	for _, op := range ops {
		p.Ops = append(p.Ops, plan.Op{
			Kind:   plan.OpCreate,
			Source: op.Source,
			Target: op.Target,
		})
	}
	for _, c := range conflicts {
		p.Conflicts = append(p.Conflicts, plan.Conflict{
			Target: c.Target,
			Source: c.Source,
			Reason: c.Reason,
		})
	}

	if len(conflicts) > 0 {
		return p, fmt.Errorf("conflicts detected: %d", len(conflicts))
	}

	logger.Debug("BuildInstallPlan: done", zap.Int("ops", len(p.Ops)))
	return p, nil
}

// ExecuteInstallPlan applies the plan to the filesystem.
// On partial failure, the partial state is saved and the error returned.
func ExecuteInstallPlan(p *plan.Plan, statePath string) (*InstallResult, error) {
	logger.Info("Installing package",
		zap.String("package", p.PackageName),
		zap.String("profile", p.Profile),
		zap.Int("ops", len(p.Ops)),
	)

	// Load existing state
	st, err := state.Load(statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	created := make([]state.InstalledLink, 0, len(p.Ops))

	saveAndReturn := func(execErr error) (*InstallResult, error) {
		st[p.PackageName] = state.PackageState{
			Profile:        p.Profile,
			InstalledLinks: created,
			InstalledAt:    time.Now(),
		}
		if saveErr := state.Save(statePath, st); saveErr != nil {
			logger.Error("Failed to save partial state",
				zap.String("path", statePath),
				zap.Error(saveErr),
			)
			return &InstallResult{LinksCreated: created}, fmt.Errorf("%w; additionally failed to save state: %v", execErr, saveErr)
		}
		return &InstallResult{LinksCreated: created}, execErr
	}

	for _, op := range p.Ops {
		if op.Kind != plan.OpCreate {
			continue
		}
		err := symlink.CreateSymlink(op.Source, op.Target)
		if err != nil {
			// Idempotency: if target already a symlink to our source, treat as success.
			isOurs, checkErr := symlink.IsSymlinkTo(op.Target, op.Source)
			if checkErr == nil && isOurs {
				created = append(created, state.InstalledLink{Source: op.Source, Target: op.Target})
				continue
			}
			logger.Error("Failed to create symlink",
				zap.String("source", op.Source),
				zap.String("target", op.Target),
				zap.Error(err),
			)
			return saveAndReturn(fmt.Errorf("failed to create symlink %s -> %s: %w", op.Target, op.Source, err))
		}
		created = append(created, state.InstalledLink{Source: op.Source, Target: op.Target})
	}

	// Success: save full state
	st[p.PackageName] = state.PackageState{
		Profile:        p.Profile,
		InstalledLinks: created,
		InstalledAt:    time.Now(),
	}
	if err := state.Save(statePath, st); err != nil {
		return &InstallResult{LinksCreated: created}, fmt.Errorf("failed to save state: %w", err)
	}

	logger.Info("Installed package",
		zap.String("package", p.PackageName),
		zap.Int("symlinks", len(created)),
	)

	return &InstallResult{LinksCreated: created}, nil
}

// Install is a convenience wrapper combining Build and Execute (used by tests).
// CLI layer should call BuildInstallPlan and ExecuteInstallPlan separately
// to insert a confirmation prompt between them.
func Install(req InstallRequest) (*InstallResult, error) {
	p, err := BuildInstallPlan(req)
	if err != nil {
		return nil, err
	}
	return ExecuteInstallPlan(p, req.StatePath)
}
