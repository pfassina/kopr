package markdown

import "testing"

func TestExtractWikiLinks(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []WikiLink
	}{
		{
			name:  "simple link",
			input: "See [[my note]] for details",
			want:  []WikiLink{{Target: "my note", Line: 1, Col: 4}},
		},
		{
			name:  "link with section",
			input: "Refer to [[note#section]]",
			want:  []WikiLink{{Target: "note", Section: "section", Line: 1, Col: 9}},
		},
		{
			name:  "link with alias",
			input: "Click [[note|display text]]",
			want:  []WikiLink{{Target: "note", Alias: "display text", Line: 1, Col: 6}},
		},
		{
			name:  "link with section and alias",
			input: "See [[note#sec|alias]]",
			want:  []WikiLink{{Target: "note", Section: "sec", Alias: "alias", Line: 1, Col: 4}},
		},
		{
			name:  "multiple links",
			input: "Link [[a]] and [[b]]",
			want: []WikiLink{
				{Target: "a", Line: 1, Col: 5},
				{Target: "b", Line: 1, Col: 15},
			},
		},
		{
			name:  "no links",
			input: "No links here",
			want:  nil,
		},
		{
			name:  "skip frontmatter",
			input: "---\ntitle: test\n---\n[[real link]]",
			want:  []WikiLink{{Target: "real link", Line: 4, Col: 0}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractWikiLinks([]byte(tt.input))
			if len(got) != len(tt.want) {
				t.Fatalf("got %d links, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i].Target != tt.want[i].Target {
					t.Errorf("[%d] target: got %q, want %q", i, got[i].Target, tt.want[i].Target)
				}
				if got[i].Section != tt.want[i].Section {
					t.Errorf("[%d] section: got %q, want %q", i, got[i].Section, tt.want[i].Section)
				}
				if got[i].Alias != tt.want[i].Alias {
					t.Errorf("[%d] alias: got %q, want %q", i, got[i].Alias, tt.want[i].Alias)
				}
			}
		})
	}
}
