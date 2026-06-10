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

func TestWikiLinkAt(t *testing.T) {
	tests := []struct {
		name  string
		input string
		line  int
		col   int
		want  string // expected target, "" if no link expected
	}{
		{
			name:  "cursor on link target",
			input: "See [[my note]] for details",
			line:  1, col: 8,
			want: "my note",
		},
		{
			name:  "cursor on opening brackets",
			input: "See [[my note]] for details",
			line:  1, col: 4,
			want: "my note",
		},
		{
			name:  "cursor on closing brackets",
			input: "See [[my note]] for details",
			line:  1, col: 14,
			want: "my note",
		},
		{
			name:  "cursor before link",
			input: "See [[my note]] for details",
			line:  1, col: 3,
			want: "",
		},
		{
			name:  "cursor after link",
			input: "See [[my note]] for details",
			line:  1, col: 15,
			want: "",
		},
		{
			name:  "second link on same line",
			input: "Link [[a]] and [[b]]",
			line:  1, col: 17,
			want: "b",
		},
		{
			name:  "link on second line",
			input: "first line\nSee [[note]]",
			line:  2, col: 6,
			want: "note",
		},
		{
			name:  "wrong line",
			input: "See [[note]]",
			line:  2, col: 5,
			want: "",
		},
		{
			name:  "no links",
			input: "No links here",
			line:  1, col: 5,
			want: "",
		},
		{
			name:  "link with alias",
			input: "Click [[note|display text]]",
			line:  1, col: 10,
			want: "note",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			links := ExtractWikiLinks([]byte(tt.input))
			got := WikiLinkAt(links, tt.line, tt.col)
			if tt.want == "" {
				if got != nil {
					t.Errorf("expected no link, got target=%q", got.Target)
				}
			} else {
				if got == nil {
					t.Fatalf("expected link with target=%q, got nil", tt.want)
				}
				if got.Target != tt.want {
					t.Errorf("target: got %q, want %q", got.Target, tt.want)
				}
			}
		})
	}
}

// TestWikiLinkAtEveryPosition verifies issue #12 acceptance criteria:
// gf must work with the cursor anywhere inside [[...]], including on the
// brackets and on the .md suffix, for targets containing spaces.
func TestWikiLinkAtEveryPosition(t *testing.T) {
	const prefix = "See "
	const raw = "[[file name with spaces.md]]"
	const want = "file name with spaces.md"
	input := prefix + raw + " end"

	links := ExtractWikiLinks([]byte(input))
	start := len(prefix)
	end := start + len(raw) // exclusive

	for col := 0; col < len(input); col++ {
		got := WikiLinkAt(links, 1, col)
		inside := col >= start && col < end
		if inside && (got == nil || got.Target != want) {
			t.Errorf("col=%d (%q): expected target %q, got %+v", col, input[col], want, got)
		}
		if !inside && got != nil {
			t.Errorf("col=%d (%q): expected no link, got target=%q", col, input[col], got.Target)
		}
	}
}

// TestWikiLinkAtMultibyte verifies cursor columns stay byte-based (matching
// Neovim's nvim_win_get_cursor) when multibyte text precedes the link.
func TestWikiLinkAtMultibyte(t *testing.T) {
	input := "héllo wörld [[my note]]"
	links := ExtractWikiLinks([]byte(input))

	start := len("héllo wörld ") // byte offset of [[
	for col := start; col < start+len("[[my note]]"); col++ {
		got := WikiLinkAt(links, 1, col)
		if got == nil || got.Target != "my note" {
			t.Errorf("col=%d: expected target %q, got %+v", col, "my note", got)
		}
	}
	if got := WikiLinkAt(links, 1, start-1); got != nil {
		t.Errorf("col before link: expected no link, got target=%q", got.Target)
	}
}

func TestExtractWikiLinksSpacesWithSectionAndAlias(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		target  string
		section string
		alias   string
	}{
		{
			name:   "spaces with md suffix",
			input:  "[[file name with spaces.md]]",
			target: "file name with spaces.md",
		},
		{
			name:    "spaces with section",
			input:   "[[file name#some section]]",
			target:  "file name",
			section: "some section",
		},
		{
			name:   "spaces with alias",
			input:  "[[file name|display name]]",
			target: "file name",
			alias:  "display name",
		},
		{
			name:    "md suffix with section and alias",
			input:   "[[file name.md#sec|alias text]]",
			target:  "file name.md",
			section: "sec",
			alias:   "alias text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			links := ExtractWikiLinks([]byte(tt.input))
			if len(links) != 1 {
				t.Fatalf("got %d links, want 1", len(links))
			}
			l := links[0]
			if l.Target != tt.target {
				t.Errorf("target: got %q, want %q", l.Target, tt.target)
			}
			if l.Section != tt.section {
				t.Errorf("section: got %q, want %q", l.Section, tt.section)
			}
			if l.Alias != tt.alias {
				t.Errorf("alias: got %q, want %q", l.Alias, tt.alias)
			}
		})
	}
}
