package vault

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReplaceWikiLinkTargets(t *testing.T) {
	tests := []struct {
		name    string
		content string
		oldName string
		newName string
		want    string
	}{
		{
			name:    "simple link",
			content: "See [[my-note]] for details.",
			oldName: "my-note",
			newName: "renamed-note",
			want:    "See [[renamed-note]] for details.",
		},
		{
			name:    "link with .md extension",
			content: "See [[my-note.md]] for details.",
			oldName: "my-note",
			newName: "renamed-note",
			want:    "See [[renamed-note.md]] for details.",
		},
		{
			name:    "link with section",
			content: "See [[my-note#intro]] for details.",
			oldName: "my-note",
			newName: "renamed-note",
			want:    "See [[renamed-note#intro]] for details.",
		},
		{
			name:    "link with alias",
			content: "See [[my-note|My Note]] for details.",
			oldName: "my-note",
			newName: "renamed-note",
			want:    "See [[renamed-note|My Note]] for details.",
		},
		{
			name:    "link with section and alias",
			content: "See [[my-note#intro|Introduction]] for details.",
			oldName: "my-note",
			newName: "renamed-note",
			want:    "See [[renamed-note#intro|Introduction]] for details.",
		},
		{
			name:    "link with .md and section",
			content: "See [[my-note.md#intro]] for details.",
			oldName: "my-note",
			newName: "renamed-note",
			want:    "See [[renamed-note.md#intro]] for details.",
		},
		{
			name:    "link with .md and alias",
			content: "See [[my-note.md|My Note]] for details.",
			oldName: "my-note",
			newName: "renamed-note",
			want:    "See [[renamed-note.md|My Note]] for details.",
		},
		{
			name:    "link with .md section and alias",
			content: "See [[my-note.md#intro|Introduction]] for details.",
			oldName: "my-note",
			newName: "renamed-note",
			want:    "See [[renamed-note.md#intro|Introduction]] for details.",
		},
		{
			name:    "multiple links",
			content: "See [[my-note]] and [[my-note#section]].",
			oldName: "my-note",
			newName: "renamed-note",
			want:    "See [[renamed-note]] and [[renamed-note#section]].",
		},
		{
			name:    "no match",
			content: "See [[other-note]] for details.",
			oldName: "my-note",
			newName: "renamed-note",
			want:    "See [[other-note]] for details.",
		},
		{
			name:    "partial name no match",
			content: "See [[my-note-extra]] for details.",
			oldName: "my-note",
			newName: "renamed-note",
			want:    "See [[my-note-extra]] for details.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := replaceWikiLinkTargets(tt.content, tt.oldName, tt.newName)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRewriteLinksInNote(t *testing.T) {
	dir := t.TempDir()
	notePath := filepath.Join(dir, "source.md")

	content := "# Source\n\nLinks to [[old-name]] and [[old-name#section]].\n"
	if err := os.WriteFile(notePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	changed, err := RewriteLinksInNote(notePath, "old-name", "new-name")
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("expected file to be changed")
	}

	data, err := os.ReadFile(notePath)
	if err != nil {
		t.Fatal(err)
	}

	want := "# Source\n\nLinks to [[new-name]] and [[new-name#section]].\n"
	if string(data) != want {
		t.Errorf("got %q, want %q", string(data), want)
	}
}

func TestRewriteLinksInNote_NoChange(t *testing.T) {
	dir := t.TempDir()
	notePath := filepath.Join(dir, "source.md")

	content := "# Source\n\nLinks to [[other-note]].\n"
	if err := os.WriteFile(notePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	changed, err := RewriteLinksInNote(notePath, "old-name", "new-name")
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Error("expected file to not be changed")
	}
}
