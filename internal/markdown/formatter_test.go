package markdown

import "testing"

func TestFormat(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "trailing whitespace",
			input: "Hello   \nWorld  \n",
			want:  "Hello\nWorld\n",
		},
		{
			name:  "heading spacing",
			input: "##  Too Many Spaces  ##\n",
			want:  "## Too Many Spaces\n",
		},
		{
			name:  "blank line before heading",
			input: "Some text\n# Heading\n",
			want:  "Some text\n\n# Heading\n",
		},
		{
			name:  "excessive blank lines",
			input: "A\n\n\n\n\nB\n",
			want:  "A\n\n\nB\n",
		},
		{
			name:  "preserve frontmatter",
			input: "---\ntitle: Test  \ntags: [a, b]\n---\n\n# Content\n",
			want:  "---\ntitle: Test  \ntags: [a, b]\n---\n\n# Content\n",
		},
		{
			name:  "ensure trailing newline",
			input: "Hello",
			want:  "Hello\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(Format([]byte(tt.input)))
			if got != tt.want {
				t.Errorf("\ngot:  %q\nwant: %q", got, tt.want)
			}
		})
	}
}
