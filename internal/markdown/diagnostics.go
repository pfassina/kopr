package markdown

import (
	"bufio"
	"bytes"
	"fmt"
	"sort"
	"strings"
)

// Severity classifies a diagnostic.
type Severity int

const (
	SeverityError Severity = iota
	SeverityWarning
)

func (s Severity) String() string {
	if s == SeverityWarning {
		return "warning"
	}
	return "error"
}

// Diagnostic is a single lint finding in a markdown document.
type Diagnostic struct {
	Line     int // 1-based line number
	Col      int // 0-based byte column
	EndCol   int // 0-based exclusive end column; 0 means whole line
	Severity Severity
	Message  string
}

// LintOptions configures the lint pass.
type LintOptions struct {
	// ResolveLink reports whether a wiki-link target resolves to an existing
	// note. Nil disables the broken-link check.
	ResolveLink func(target string) bool
}

// Lint runs all diagnostic checks over a markdown document and returns
// findings sorted by position.
func Lint(content []byte, opts LintOptions) []Diagnostic {
	var diags []Diagnostic
	diags = append(diags, lintFrontmatter(content)...)
	diags = append(diags, lintWikiLinks(content, opts.ResolveLink)...)
	diags = append(diags, lintHeadings(content)...)

	sort.SliceStable(diags, func(i, j int) bool {
		if diags[i].Line != diags[j].Line {
			return diags[i].Line < diags[j].Line
		}
		return diags[i].Col < diags[j].Col
	})
	return diags
}

// lintFrontmatter checks the --- delimited frontmatter block.
func lintFrontmatter(content []byte) []Diagnostic {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	if !scanner.Scan() {
		return nil
	}
	if strings.TrimSpace(scanner.Text()) != "---" {
		return nil
	}

	var diags []Diagnostic
	lineNum := 1
	closed := false
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			closed = true
			break
		}
		// Blank lines, comments, and YAML list items are fine.
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || trimmed == "-" || strings.HasPrefix(trimmed, "- ") {
			continue
		}
		if !strings.Contains(line, ":") {
			diags = append(diags, Diagnostic{
				Line:     lineNum,
				Severity: SeverityError,
				Message:  `malformed frontmatter line: expected "key: value"`,
			})
		}
	}

	if !closed {
		// Without a closing delimiter the whole document scans as frontmatter,
		// so per-line findings would be noise; report only the real problem.
		return []Diagnostic{{
			Line:     1,
			Severity: SeverityError,
			Message:  "unterminated frontmatter: missing closing ---",
		}}
	}
	return diags
}

// lintWikiLinks checks [[wiki links]] for unclosed brackets, missing targets,
// and (when resolve is provided) targets that don't resolve to a note.
func lintWikiLinks(content []byte, resolve func(string) bool) []Diagnostic {
	var diags []Diagnostic
	scanner := bufio.NewScanner(bytes.NewReader(content))

	inFrontmatter := false
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip frontmatter, matching ExtractWikiLinks.
		if lineNum == 1 && strings.TrimSpace(line) == "---" {
			inFrontmatter = true
			continue
		}
		if inFrontmatter {
			if strings.TrimSpace(line) == "---" {
				inFrontmatter = false
			}
			continue
		}

		col := 0
		for {
			idx := strings.Index(line[col:], "[[")
			if idx == -1 {
				break
			}
			open := col + idx
			start := open + 2

			end := strings.Index(line[start:], "]]")
			if end == -1 {
				diags = append(diags, Diagnostic{
					Line:     lineNum,
					Col:      open,
					EndCol:   open + 2,
					Severity: SeverityError,
					Message:  "unclosed wiki link: missing ]]",
				})
				break
			}

			inner := line[start : start+end]
			endCol := start + end + 2

			// Target is everything before #section or |alias.
			target := inner
			if sep := strings.IndexAny(inner, "#|"); sep != -1 {
				target = inner[:sep]
			}
			target = strings.TrimSpace(target)

			switch {
			case target == "":
				diags = append(diags, Diagnostic{
					Line:     lineNum,
					Col:      open,
					EndCol:   endCol,
					Severity: SeverityError,
					Message:  "wiki link missing target",
				})
			case resolve != nil && !resolve(target):
				diags = append(diags, Diagnostic{
					Line:     lineNum,
					Col:      open,
					EndCol:   endCol,
					Severity: SeverityWarning,
					Message:  fmt.Sprintf("wiki link target not found: %s", target),
				})
			}

			col = endCol
		}
	}
	return diags
}

// lintHeadings checks for duplicate headings and heading level jumps.
func lintHeadings(content []byte) []Diagnostic {
	headings := ExtractHeadings(content)
	var diags []Diagnostic

	seen := make(map[string]int, len(headings)) // normalized text → first line
	for i, h := range headings {
		key := strings.ToLower(h.Text)
		if first, ok := seen[key]; ok {
			diags = append(diags, Diagnostic{
				Line:     h.Line,
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("duplicate heading %q (first at line %d)", h.Text, first),
			})
		} else {
			seen[key] = h.Line
		}

		if i > 0 && h.Level > headings[i-1].Level+1 {
			diags = append(diags, Diagnostic{
				Line:     h.Line,
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("heading level jumps from h%d to h%d", headings[i-1].Level, h.Level),
			})
		}
	}
	return diags
}
