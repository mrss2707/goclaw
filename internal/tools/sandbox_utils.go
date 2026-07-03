package tools

import (
	"context"
	"fmt"
	"log/slog"
	"path"
	"path/filepath"
	"strings"

	"github.com/nextlevelbuilder/goclaw/internal/store"
)

// SandboxCwd maps the current effective workspace (from context) to its
// corresponding path inside the sandbox container. The sandbox mounts the
// global workspace root at containerBase (usually "/workspace"). This function
// computes the relative path from globalWorkspace to the context workspace
// and joins it with containerBase.
//
// Example: globalWorkspace="/app/workspace", ctx workspace="/app/workspace/agent-a/user-123"
// → returns "/workspace/agent-a/user-123"
func SandboxCwd(ctx context.Context, globalWorkspace, containerBase string) (string, error) {
	ws := ToolWorkspaceFromCtx(ctx)
	if ws == "" {
		// No per-request workspace — fall back to container root.
		return containerBase, nil
	}

	rel, err := filepath.Rel(globalWorkspace, ws)
	if err != nil || strings.HasPrefix(filepath.Clean(rel), "..") {
		return "", fmt.Errorf("workspace %q is outside global mount %q", ws, globalWorkspace)
	}

	if rel == "." {
		return containerBase, nil
	}
	return path.Join(filepath.ToSlash(containerBase), filepath.ToSlash(rel)), nil
}

func effectiveSandboxWorkspace(ctx context.Context, globalWorkspace string) (string, error) {
	teamRoot := ToolTeamRootFromCtx(ctx)
	ws := ToolWorkspaceFromCtx(ctx)

	// Prefer team root as mount only when the agent's active workspace lives
	// under it (dispatched team task). Otherwise — e.g. a team member working in
	// personal workspace — the personal workspace is outside the team root tree
	// and sandboxCwdForHostPath would fail. In that case fall back to the
	// personal workspace; cross-scope team reads are still available through
	// host-side allowed-prefix resolution.
	if teamRoot != "" && ws != "" {
		rel, err := filepath.Rel(teamRoot, ws)
		if err == nil && !strings.HasPrefix(filepath.Clean(rel), "..") {
			return canonicalSandboxWorkspace(teamRoot), nil
		}
		return canonicalSandboxWorkspace(ws), nil
	}
	if teamRoot != "" {
		return canonicalSandboxWorkspace(teamRoot), nil
	}
	if ws != "" {
		return canonicalSandboxWorkspace(ws), nil
	}
	if globalWorkspace != "" && store.IsMasterScope(ctx) {
		slog.Warn("security.sandbox_global_workspace_fallback",
			"workspace", globalWorkspace,
			"tenant_id", store.TenantIDFromContext(ctx),
			"agent_id", store.AgentIDFromContext(ctx))
		return canonicalSandboxWorkspace(globalWorkspace), nil
	}
	return "", fmt.Errorf("sandbox workspace unavailable for tenant-scoped execution")
}

func canonicalSandboxWorkspace(workspace string) string {
	clean := filepath.Clean(workspace)
	if real, err := filepath.EvalSymlinks(clean); err == nil {
		return real
	}
	return clean
}

func sandboxCwdForHostPath(hostCwd, mountWorkspace, containerBase string) (string, error) {
	if hostCwd == "" {
		hostCwd = mountWorkspace
	}
	if containerBase == "" {
		containerBase = "/workspace"
	}
	cleanMount := filepath.Clean(mountWorkspace)
	cleanCwd := filepath.Clean(hostCwd)
	rel, err := filepath.Rel(cleanMount, cleanCwd)
	if err != nil || strings.HasPrefix(filepath.Clean(rel), "..") {
		return "", fmt.Errorf("working directory %q is outside sandbox mount %q", hostCwd, mountWorkspace)
	}
	if rel == "." {
		return filepath.ToSlash(containerBase), nil
	}
	return path.Join(filepath.ToSlash(containerBase), filepath.ToSlash(rel)), nil
}

// ResolveSandboxPath resolves a tool-provided path (relative or absolute)
// against the sandbox container CWD. Boundary enforcement is delegated to
// callers (e.g. rejectCrossScopeWrite for write operations).
func ResolveSandboxPath(filePath, containerCwd string) string {
	cwd := path.Clean(containerCwd)
	if cwd == "." || cwd == "/" {
		cwd = "/workspace"
	}
	if strings.HasPrefix(filePath, "/") {
		return path.Clean(filePath)
	}
	return path.Clean(path.Join(cwd, filePath))
}

// rejectCrossScopeWrite returns an error if resolvedPath falls outside the
// agent's sandbox CWD. This prevents an agent in chat_1 from writing into
// chat_2 through a cross-scope absolute path after ResolveSandboxPath resolves it.
func rejectCrossScopeWrite(resolvedPath, containerCwd string) error {
	cwd := path.Clean(containerCwd)
	if cwd == "." || cwd == "/" {
		cwd = "/workspace"
	}
	rp := path.Clean(resolvedPath)
	if rp == cwd || strings.HasPrefix(rp, cwd+"/") {
		return nil
	}
	return fmt.Errorf("access denied: write path %q is outside the agent workspace scope %q", resolvedPath, containerCwd)
}
