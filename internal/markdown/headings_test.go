package markdown

import "testing"

func TestExtractHeadings(t *testing.T) {
	input := `---
title: Test
---

# Heading 1

Some text.

## Heading 2

### Heading 3
`
	headings := ExtractHeadings([]byte(input))

	if len(headings) != 3 {
		t.Fatalf("got %d headings, want 3", len(headings))
	}

	tests := []struct {
		level int
		text  string
	}{
		{1, "Heading 1"},
		{2, "Heading 2"},
		{3, "Heading 3"},
	}

	for i, tt := range tests {
		if headings[i].Level != tt.level {
			t.Errorf("[%d] level: got %d, want %d", i, headings[i].Level, tt.level)
		}
		if headings[i].Text != tt.text {
			t.Errorf("[%d] text: got %q, want %q", i, headings[i].Text, tt.text)
		}
	}
}
