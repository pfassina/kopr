package markdown

import "testing"

func TestResolveWikiLinkTarget(t *testing.T) {
	tests := []struct {
		name   string
		target string
		want   string
	}{
		{name: "bare name", target: "note", want: "note.md"},
		{name: "name with spaces", target: "file name with spaces", want: "file name with spaces.md"},
		{name: "already has extension", target: "note.md", want: "note.md"},
		{name: "extension with spaces", target: "file name with spaces.md", want: "file name with spaces.md"},
		{name: "uppercase extension", target: "note.MD", want: "note.MD"},
		{name: "mixed case extension", target: "note.Md", want: "note.Md"},
		{name: "folder path", target: "folder/note", want: "folder/note.md"},
		{name: "dot in name is not an extension", target: "release v1.2", want: "release v1.2.md"},
		{name: "surrounding whitespace trimmed", target: "  note  ", want: "note.md"},
		{name: "empty", target: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolveWikiLinkTarget(tt.target); got != tt.want {
				t.Errorf("ResolveWikiLinkTarget(%q) = %q, want %q", tt.target, got, tt.want)
			}
		})
	}
}

func TestNoteNameFromPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{path: "folder/my-note.md", want: "my-note"},
		{path: "my note.md", want: "my note"},
		{path: "note", want: "note"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := NoteNameFromPath(tt.path); got != tt.want {
				t.Errorf("NoteNameFromPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}
