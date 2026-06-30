package tools

import (
	"context"
	"testing"
)

func TestSandboxCwd(t *testing.T) {
	tests := []struct {
		name            string
		ctxWorkspace    string // empty = no workspace in context
		globalWorkspace string
		containerBase   string
		want            string
		wantErr         bool
	}{
		{
			name:            "no workspace in context — fallback to container base",
			ctxWorkspace:    "",
			globalWorkspace: "/app/workspace",
			containerBase:   "/workspace",
			want:            "/workspace",
		},
		{
			name:            "workspace equals global mount",
			ctxWorkspace:    "/app/workspace",
			globalWorkspace: "/app/workspace",
			containerBase:   "/workspace",
			want:            "/workspace",
		},
		{
			name:            "per-agent workspace",
			ctxWorkspace:    "/app/workspace/agent-a-workspace",
			globalWorkspace: "/app/workspace",
			containerBase:   "/workspace",
			want:            "/workspace/agent-a-workspace",
		},
		{
			name:            "per-user workspace",
			ctxWorkspace:    "/app/workspace/agent-a/user-123",
			globalWorkspace: "/app/workspace",
			containerBase:   "/workspace",
			want:            "/workspace/agent-a/user-123",
		},
		{
			name:            "team workspace",
			ctxWorkspace:    "/app/workspace/teams/team-uuid/chat-123",
			globalWorkspace: "/app/workspace",
			containerBase:   "/workspace",
			want:            "/workspace/teams/team-uuid/chat-123",
		},
		{
			name:            "workspace outside global mount — error",
			ctxWorkspace:    "/other/path/agent-a",
			globalWorkspace: "/app/workspace",
			containerBase:   "/workspace",
			wantErr:         true,
		},
		{
			name:            "custom container base",
			ctxWorkspace:    "/app/workspace/agent-a",
			globalWorkspace: "/app/workspace",
			containerBase:   "/home/sandbox",
			want:            "/home/sandbox/agent-a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.ctxWorkspace != "" {
				ctx = WithToolWorkspace(ctx, tt.ctxWorkspace)
			}
			got, err := SandboxCwd(ctx, tt.globalWorkspace, tt.containerBase)
			if tt.wantErr {
				if err == nil {
					t.Errorf("SandboxCwd() = %q, want error", got)
				}
				return
			}
			if err != nil {
				t.Errorf("SandboxCwd() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("SandboxCwd() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveSandboxPath(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		containerCwd string
		want         string
	}{
		{
			name:         "relative path joined with cwd",
			path:         "file.txt",
			containerCwd: "/workspace/agent-a",
			want:         "/workspace/agent-a/file.txt",
		},
		{
			name:         "relative subdirectory path",
			path:         "subdir/file.txt",
			containerCwd: "/workspace/agent-a",
			want:         "/workspace/agent-a/subdir/file.txt",
		},
		{
			name:         "absolute sibling workspace path resolves (no clamp)",
			path:         "/workspace/agent-a/file.txt",
			containerCwd: "/workspace/agent-b",
			want:         "/workspace/agent-a/file.txt",
		},
		{
			name:         "absolute path inside cwd stays absolute",
			path:         "/workspace/agent-a/file.txt",
			containerCwd: "/workspace/agent-a",
			want:         "/workspace/agent-a/file.txt",
		},
		{
			name:         "relative parent escape resolves (no clamp)",
			path:         "../agent-b/file.txt",
			containerCwd: "/workspace/agent-a",
			want:         "/workspace/agent-b/file.txt",
		},
		{
			name:         "dot path",
			path:         ".",
			containerCwd: "/workspace/agent-a",
			want:         "/workspace/agent-a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveSandboxPath(tt.path, tt.containerCwd)
			if got != tt.want {
				t.Errorf("ResolveSandboxPath(%q, %q) = %q, want %q", tt.path, tt.containerCwd, got, tt.want)
			}
		})
	}
}

func TestRejectCrossScopeWrite(t *testing.T) {
	tests := []struct {
		name         string
		resolvedPath string
		containerCwd string
		wantErr      bool
	}{
		{
			name:         "same scope — file inside cwd",
			resolvedPath: "/workspace/chat_1/file.txt",
			containerCwd: "/workspace/chat_1",
			wantErr:      false,
		},
		{
			name:         "same scope — exact cwd",
			resolvedPath: "/workspace/chat_1",
			containerCwd: "/workspace/chat_1",
			wantErr:      false,
		},
		{
			name:         "sibling scope — write to chat_2 from chat_1",
			resolvedPath: "/workspace/chat_2/evil.txt",
			containerCwd: "/workspace/chat_1",
			wantErr:      true,
		},
		{
			name:         "team root — write to root from chat",
			resolvedPath: "/workspace/file.txt",
			containerCwd: "/workspace/chat_1",
			wantErr:      true,
		},
		{
			name:         "outside mount — write outside /workspace",
			resolvedPath: "/etc/passwd",
			containerCwd: "/workspace/chat_1",
			wantErr:      true,
		},
		{
			name:         "relative escape — ../chat_2/evil.txt",
			resolvedPath: "/workspace/chat_2/evil.txt",
			containerCwd: "/workspace/chat_1",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rejectCrossScopeWrite(tt.resolvedPath, tt.containerCwd)
			if tt.wantErr && err == nil {
				t.Errorf("rejectCrossScopeWrite(%q, %q) = nil, want error", tt.resolvedPath, tt.containerCwd)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("rejectCrossScopeWrite(%q, %q) = %v, want nil", tt.resolvedPath, tt.containerCwd, err)
			}
		})
	}
}
