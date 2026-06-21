package googlechat

import (
	"testing"
)

func TestFormatGoogleChat(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string // substrings that MUST be in output
		absent   []string // substrings that must NOT be in output
	}{
		{
			name:     "bold conversion double to single asterisk",
			input:    "This is **bold** text",
			contains: []string{"*bold*"},
			absent:   []string{"**bold**"},
		},
		{
			name:     "heading converted to bold",
			input:    "### Important\n\nContent here",
			contains: []string{"*Important*"},
			absent:   []string{"### Important"},
		},
		{
			name:     "markdown link to Google Chat format",
			input:    "See [Google](https://google.com) here",
			contains: []string{"<https://google.com|Google>"},
			absent:   []string{"[Google](https://google.com)"},
		},
		{
			name:     "italic preserved",
			input:    "This is _italic_ text",
			contains: []string{"_italic_"},
		},
		{
			name:     "inline code preserved",
			input:    "Use `fmt.Println()` for output",
			contains: []string{"`fmt.Println()`"},
		},
		{
			name:     "strikethrough preserved",
			input:    "This is ~wrong~ correct",
			contains: []string{"~wrong~"},
		},
		{
			name:     "zero-width chars stripped",
			input:    "hello\u200b\u200cworld\u200dtest",
			contains: []string{"helloworldtest"},
			absent:   []string{"\u200b", "\u200c", "\u200d"},
		},
		{
			name:  "bullet list preserved",
			input: "- Item 1\n- Item 2\n- Item 3",
			contains: []string{"- Item 1", "- Item 2", "- Item 3"},
		},
		{
			name:     "plain text unchanged",
			input:    "Hello, world! This is plain text.",
			contains: []string{"Hello, world! This is plain text."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatGoogleChat(tt.input)
			for _, c := range tt.contains {
				if !containsStr(result, c) {
					t.Errorf("expected output to contain %q\ngot: %s", c, result)
				}
			}
			for _, a := range tt.absent {
				if containsStr(result, a) {
					t.Errorf("expected output NOT to contain %q\ngot: %s", a, result)
				}
			}
		})
	}
}

func TestChunkText(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		maxChars int
		want    int // expected number of chunks
	}{
		{"short text", "Hello world", 4000, 1},
		{"exact limit", string(make([]byte, 100)), 100, 1},
		{"two chunks at newline", "first line\nsecond line that is long", 15, 3},
		{"three chunks at spaces", "one two three four five six seven", 10, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := chunkText(tt.input, tt.maxChars)
			if len(chunks) != tt.want {
				t.Errorf("chunkText(%q, %d) = %d chunks, want %d", tt.input, tt.maxChars, len(chunks), tt.want)
			}
		})
	}
}

func TestTableToASCII(t *testing.T) {
	tests := []struct {
		name  string
		rows  []string
		check string // substring that must appear in output
	}{
		{
			name:  "simple table",
			rows:  []string{"| Name | Age |", "|------|-----|", "| Alice | 30 |", "| Bob | 25 |"},
			check: "Alice",
		},
		{
			name:  "empty input",
			rows:  nil,
			check: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tableToASCII(tt.rows)
			if tt.check != "" && !containsAnyLine(result, tt.check) {
				t.Errorf("tableToASCII missing %q in: %v", tt.check, result)
			}
			if tt.rows == nil && result != nil {
				t.Errorf("expected nil for empty input, got %v", result)
			}
		})
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func containsAnyLine(lines []string, substr string) bool {
	for _, l := range lines {
		if containsStr(l, substr) {
			return true
		}
	}
	return false
}
