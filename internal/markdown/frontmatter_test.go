package markdown

import "testing"

func TestExtractFrontmatter(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  *Frontmatter
	}{
		{
			name:  "no frontmatter",
			input: "# Hello\n\nWorld",
			want:  nil,
		},
		{
			name:  "basic frontmatter",
			input: "---\ntitle: My Note\ntags: [go, test]\nstatus: draft\n---\n\n# Content",
			want: &Frontmatter{
				Title:   "My Note",
				Tags:    []string{"go", "test"},
				Status:  "draft",
				EndLine: 5,
				Raw:     map[string]string{"title": "My Note", "tags": "[go, test]", "status": "draft"},
			},
		},
		{
			name:  "unclosed frontmatter",
			input: "---\ntitle: Unclosed\n",
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractFrontmatter([]byte(tt.input))
			if tt.want == nil {
				if got != nil {
					t.Errorf("expected nil, got %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil frontmatter")
			}
			if got.Title != tt.want.Title {
				t.Errorf("title: got %q, want %q", got.Title, tt.want.Title)
			}
			if got.Status != tt.want.Status {
				t.Errorf("status: got %q, want %q", got.Status, tt.want.Status)
			}
			if len(got.Tags) != len(tt.want.Tags) {
				t.Errorf("tags: got %v, want %v", got.Tags, tt.want.Tags)
			}
		})
	}
}
