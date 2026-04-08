package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	shellwords "github.com/mattn/go-shellwords"
)

// dynamicPathExemptions builds runtime exemptions for the active personal
// .uploads subtree, legacy uploads subtree, and team workspace root.
// Only paths nested under a denied root are included — other paths don't
// need exemptions because they aren't blocked in the first place.
func (t *ExecTool) dynamicPathExemptions(ctx context.Context) []string {
	var exemptions []string
	seen := make(map[string]struct{}, 4)
	workspace := ToolWorkspaceFromCtx(ctx)
	teamWorkspace := ToolTeamWorkspaceFromCtx(ctx)

	var dirs []string
	if teamWorkspace != "" {
		dirs = append(dirs, teamWorkspace)
	}
	if workspace != "" && filepath.Clean(workspace) != filepath.Clean(teamWorkspace) {
		dirs = append(dirs, filepath.Join(workspace, ".uploads"))
		dirs = append(dirs, filepath.Join(workspace, "uploads"))
	}

	for _, dir := range dirs {
		if dir == "" || strings.Contains(dir, "..") {
			continue
		}
		for _, variant := range pathAliasVariants(filepath.Clean(dir)) {
			if !t.isNestedUnderDeniedRoot(variant) {
				continue
			}
			for _, ex := range []string{variant, variant + string(filepath.Separator)} {
				if _, ok := seen[ex]; ok {
					continue
				}
				seen[ex] = struct{}{}
				exemptions = append(exemptions, ex)
			}
		}
	}
	return exemptions
}

// pathAliasVariants returns the original path plus any known runtime aliases.
// The Docker production layout maps /app/workspace → /app/.goclaw (symlink)
// so uploads accessed via either prefix must resolve to the same exemption.
func pathAliasVariants(path string) []string {
	variants := []string{path}
	for _, mapping := range [][2]string{
		{"/app/workspace", "/app/.goclaw"},
		{"/app/.goclaw", "/app/workspace"},
	} {
		from, to := mapping[0], mapping[1]
		if path == from {
			variants = append(variants, to)
			continue
		}
		if strings.HasPrefix(path, from+string(filepath.Separator)) {
			variants = append(variants, to+strings.TrimPrefix(path, from))
		}
	}
	return variants
}

// isNestedUnderDeniedRoot reports whether path falls under any denied root.
func (t *ExecTool) isNestedUnderDeniedRoot(path string) bool {
	for _, root := range t.pathDenyRoots {
		cleanRoot := filepath.Clean(root)
		if cleanRoot == "." || cleanRoot == string(filepath.Separator) {
			continue
		}
		if !filepath.IsAbs(cleanRoot) {
			marker := string(filepath.Separator) + cleanRoot + string(filepath.Separator)
			if strings.Contains(path, marker) {
				return true
			}
			continue
		}
		if path == cleanRoot {
			continue
		}
		if strings.HasPrefix(path, cleanRoot+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// matchesPathExemption checks if path is covered by any exemption prefix.
func matchesPathExemption(path string, exemptions []string) bool {
	sep := string(filepath.Separator)
	for _, ex := range exemptions {
		if ex == "" {
			continue
		}
		if path == ex {
			return true
		}
		if strings.HasSuffix(ex, sep) {
			if strings.HasPrefix(path, ex) {
				return true
			}
			continue
		}
		if strings.HasPrefix(path, ex+sep) {
			return true
		}
	}
	return false
}

// parseExecCommandWords splits a shell command into words using quote-aware
// parsing. Segments are first split on shell operators (;, |, &, etc.), then
// each segment is parsed via go-shellwords for proper quote handling.
func parseExecCommandWords(command string) []string {
	var words []string
	for _, segment := range splitExecCommandSegments(command) {
		parser := shellwords.NewParser()
		parser.ParseBacktick = false
		parser.ParseEnv = false

		segmentWords, err := parser.Parse(segment)
		if err != nil || len(segmentWords) == 0 {
			words = append(words, strings.Fields(segment)...)
			continue
		}
		words = append(words, segmentWords...)
	}
	if len(words) == 0 {
		return strings.Fields(command)
	}
	return words
}

// splitExecCommandSegments splits a command on shell operators while respecting
// single/double quotes and backslash escapes.
func splitExecCommandSegments(command string) []string {
	var segments []string
	start := 0
	inSingle := false
	inDouble := false

	for i := 0; i < len(command); i++ {
		ch := command[i]
		switch {
		case inSingle:
			if ch == '\'' {
				inSingle = false
			}
		case inDouble:
			if ch == '\\' && i+1 < len(command) {
				i++
			} else if ch == '"' {
				inDouble = false
			}
		default:
			switch ch {
			case '\\':
				if i+1 < len(command) {
					i++
				}
			case '\'':
				inSingle = true
			case '"':
				inDouble = true
			case ';', '|', '&', '<', '>', '\n', '\r':
				if segment := strings.TrimSpace(command[start:i]); segment != "" {
					segments = append(segments, segment)
				}
				start = i + 1
			}
		}
	}

	if tail := strings.TrimSpace(command[start:]); tail != "" {
		segments = append(segments, tail)
	}
	return segments
}

// extractPathCandidates extracts potential file paths from a shell word by
// splitting on = and @ separators (e.g. file=@/path, --input=/path).
func extractPathCandidates(word string) []string {
	if word == "" {
		return nil
	}

	queue := []string{word}
	seen := make(map[string]struct{}, 4)
	var out []string

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if current == "" {
			continue
		}
		if _, ok := seen[current]; ok {
			continue
		}
		seen[current] = struct{}{}
		if looksLikePathCandidate(current) {
			out = append(out, current)
		}
		for _, sep := range []string{"=", "@"} {
			if idx := strings.Index(current, sep); idx >= 0 && idx+1 < len(current) {
				queue = append(queue, current[idx+1:])
			}
		}
	}
	return out
}

// looksLikePathCandidate returns true if s looks like a filesystem path.
func looksLikePathCandidate(s string) bool {
	if s == "" {
		return false
	}
	if filepath.IsAbs(s) {
		return true
	}
	return strings.HasPrefix(s, "./") ||
		strings.HasPrefix(s, "../") ||
		strings.HasPrefix(s, ".uploads/") ||
		strings.HasPrefix(s, ".goclaw/") ||
		strings.HasPrefix(s, "teams/") ||
		strings.HasPrefix(s, "tenants/") ||
		strings.HasPrefix(s, "~/") ||
		strings.Contains(s, string(filepath.Separator))
}

// canonicalizeExecPath resolves a potentially relative path to an absolute,
// symlink-resolved canonical form for safe comparison.
func canonicalizeExecPath(path, baseDir string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(homeDir, strings.TrimPrefix(path, "~/"))
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}
	absPath, _ := filepath.Abs(path)
	if real, err := filepath.EvalSymlinks(absPath); err == nil {
		return real, nil
	}
	return resolveThroughExistingAncestors(absPath)
}

// matchesAnyPathExemption checks if any path candidate extracted from word
// matches any exemption after canonicalization.
func matchesAnyPathExemption(word string, exemptions []string, baseDir string) bool {
	for _, candidate := range extractPathCandidates(word) {
		if strings.Contains(candidate, "..") {
			continue
		}
		realCandidate, err := canonicalizeExecPath(candidate, baseDir)
		if err != nil {
			continue
		}
		for _, exemption := range exemptions {
			realExemption, err := canonicalizeExecPath(exemption, baseDir)
			if err != nil {
				continue
			}
			if matchesPathExemption(realCandidate, []string{realExemption}) {
				return true
			}
		}
	}
	return false
}
