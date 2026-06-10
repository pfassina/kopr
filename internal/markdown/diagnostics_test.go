package markdown

import (
	"strings"
	"testing"
)

func TestLintCleanDocument(t *testing.T) {
	input := `---
title: Clean
tags: [a, b]
---

# Title

Some text with a [[good-link]].

## Section
`
	resolve := func(target string) bool { return true }
	diags := Lint([]byte(input), LintOptions{ResolveLink: resolve})
	if len(diags) != 0 {
		t.Fatalf("got %d diagnostics, want 0: %+v", len(diags), diags)
	}
}

func TestLintUnterminatedFrontmatter(t *testing.T) {
	input := "---\ntitle: Test\n\n# Heading\n"
	diags := Lint([]byte(input), LintOptions{})
	if len(diags) != 1 {
		t.Fatalf("got %d diagnostics, want 1: %+v", len(diags), diags)
	}
	d := diags[0]
	if d.Line != 1 || d.Severity != SeverityError {
		t.Errorf("got line %d severity %v, want line 1 error", d.Line, d.Severity)
	}
	if !strings.Contains(d.Message, "unterminated frontmatter") {
		t.Errorf("unexpected message: %q", d.Message)
	}
}

func TestLintMalformedFrontmatterLine(t *testing.T) {
	input := `---
title: Test
not a key value pair
---
`
	diags := Lint([]byte(input), LintOptions{})
	if len(diags) != 1 {
		t.Fatalf("got %d diagnostics, want 1: %+v", len(diags), diags)
	}
	d := diags[0]
	if d.Line != 3 || d.Severity != SeverityError {
		t.Errorf("got line %d severity %v, want line 3 error", d.Line, d.Severity)
	}
	if !strings.Contains(d.Message, "malformed frontmatter") {
		t.Errorf("unexpected message: %q", d.Message)
	}
}

func TestLintFrontmatterAllowsListsCommentsBlanks(t *testing.T) {
	input := `---
title: Test
tags:
  - one
  - two

# a comment
---
`
	diags := Lint([]byte(input), LintOptions{})
	if len(diags) != 0 {
		t.Fatalf("got %d diagnostics, want 0: %+v", len(diags), diags)
	}
}

func TestLintUnclosedWikiLink(t *testing.T) {
	input := "# T\n\nbroken [[link here\n"
	diags := Lint([]byte(input), LintOptions{})
	if len(diags) != 1 {
		t.Fatalf("got %d diagnostics, want 1: %+v", len(diags), diags)
	}
	d := diags[0]
	if d.Line != 3 || d.Col != 7 || d.Severity != SeverityError {
		t.Errorf("got line %d col %d severity %v, want line 3 col 7 error", d.Line, d.Col, d.Severity)
	}
	if !strings.Contains(d.Message, "unclosed wiki link") {
		t.Errorf("unexpected message: %q", d.Message)
	}
}

func TestLintEmptyWikiLinkTarget(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"empty", "see [[]]\n"},
		{"section only", "see [[#section]]\n"},
		{"alias only", "see [[|alias]]\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			diags := Lint([]byte(tc.input), LintOptions{})
			if len(diags) != 1 {
				t.Fatalf("got %d diagnostics, want 1: %+v", len(diags), diags)
			}
			d := diags[0]
			if d.Severity != SeverityError || !strings.Contains(d.Message, "missing target") {
				t.Errorf("got severity %v message %q, want error about missing target", d.Severity, d.Message)
			}
			if d.Col != 4 {
				t.Errorf("got col %d, want 4", d.Col)
			}
		})
	}
}

func TestLintBrokenWikiLink(t *testing.T) {
	input := "see [[exists]] and [[missing]]\n"
	resolve := func(target string) bool { return target == "exists" }
	diags := Lint([]byte(input), LintOptions{ResolveLink: resolve})
	if len(diags) != 1 {
		t.Fatalf("got %d diagnostics, want 1: %+v", len(diags), diags)
	}
	d := diags[0]
	if d.Severity != SeverityWarning {
		t.Errorf("got severity %v, want warning", d.Severity)
	}
	if d.Col != 19 || d.EndCol != 30 {
		t.Errorf("got col %d endCol %d, want 19/30", d.Col, d.EndCol)
	}
	if !strings.Contains(d.Message, "missing") {
		t.Errorf("unexpected message: %q", d.Message)
	}
}

func TestLintBrokenLinkCheckDisabledWithoutResolver(t *testing.T) {
	input := "see [[missing]]\n"
	diags := Lint([]byte(input), LintOptions{})
	if len(diags) != 0 {
		t.Fatalf("got %d diagnostics, want 0: %+v", len(diags), diags)
	}
}

func TestLintWikiLinkResolvesTargetBeforeSectionAndAlias(t *testing.T) {
	input := "see [[note#sec|alias]]\n"
	var got string
	resolve := func(target string) bool {
		got = target
		return true
	}
	Lint([]byte(input), LintOptions{ResolveLink: resolve})
	if got != "note" {
		t.Errorf("resolver got %q, want %q", got, "note")
	}
}

func TestLintWikiLinksSkipFrontmatter(t *testing.T) {
	input := "---\ntitle: has [[brackets\n---\n\nbody\n"
	diags := Lint([]byte(input), LintOptions{})
	if len(diags) != 0 {
		t.Fatalf("got %d diagnostics, want 0: %+v", len(diags), diags)
	}
}

func TestLintDuplicateHeading(t *testing.T) {
	input := "# Notes\n\n## Ideas\n\n## ideas\n"
	diags := Lint([]byte(input), LintOptions{})
	if len(diags) != 1 {
		t.Fatalf("got %d diagnostics, want 1: %+v", len(diags), diags)
	}
	d := diags[0]
	if d.Line != 5 || d.Severity != SeverityWarning {
		t.Errorf("got line %d severity %v, want line 5 warning", d.Line, d.Severity)
	}
	if !strings.Contains(d.Message, "duplicate heading") || !strings.Contains(d.Message, "line 3") {
		t.Errorf("unexpected message: %q", d.Message)
	}
}

func TestLintHeadingLevelSkip(t *testing.T) {
	input := "# Top\n\n### Deep\n"
	diags := Lint([]byte(input), LintOptions{})
	if len(diags) != 1 {
		t.Fatalf("got %d diagnostics, want 1: %+v", len(diags), diags)
	}
	d := diags[0]
	if d.Line != 3 || d.Severity != SeverityWarning {
		t.Errorf("got line %d severity %v, want line 3 warning", d.Line, d.Severity)
	}
	if !strings.Contains(d.Message, "h1 to h3") {
		t.Errorf("unexpected message: %q", d.Message)
	}
}

func TestLintHeadingDecreaseAndStepUpAllowed(t *testing.T) {
	input := "# Top\n\n## Mid\n\n### Deep\n\n# Other\n"
	diags := Lint([]byte(input), LintOptions{})
	if len(diags) != 0 {
		t.Fatalf("got %d diagnostics, want 0: %+v", len(diags), diags)
	}
}

func TestLintSortedByPosition(t *testing.T) {
	input := "# A\n\n### Skip\n\nbad [[\n\n# A\n"
	diags := Lint([]byte(input), LintOptions{})
	if len(diags) != 3 {
		t.Fatalf("got %d diagnostics, want 3: %+v", len(diags), diags)
	}
	for i := 1; i < len(diags); i++ {
		if diags[i].Line < diags[i-1].Line {
			t.Errorf("diagnostics not sorted: %+v", diags)
		}
	}
}
