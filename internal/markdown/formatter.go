package markdown

import (
	"bufio"
	"bytes"
	"strings"
)

// Format applies deterministic CommonMark-compatible formatting to markdown.
// Rules:
//   - Normalize heading spacing (blank line before, one space after #)
//   - Normalize list item spacing
//   - Trim trailing whitespace
//   - Ensure single trailing newline
//   - Normalize blank lines (max 2 consecutive)
//   - Preserve frontmatter as-is
func Format(content []byte) []byte {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	var lines []string

	inFrontmatter := false
	frontmatterDone := false
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Handle frontmatter
		if lineNum == 1 && strings.TrimSpace(line) == "---" {
			inFrontmatter = true
			lines = append(lines, line)
			continue
		}
		if inFrontmatter {
			lines = append(lines, line)
			if strings.TrimSpace(line) == "---" {
				inFrontmatter = false
				frontmatterDone = true
			}
			continue
		}

		// Trim trailing whitespace
		line = strings.TrimRight(line, " \t")

		// Normalize headings: ensure single space after #
		if isHeading(line) {
			line = normalizeHeading(line)
		}

		lines = append(lines, line)
	}

	// Normalize blank lines
	lines = normalizeBlankLines(lines, frontmatterDone)

	// Ensure single trailing newline
	result := strings.Join(lines, "\n")
	result = strings.TrimRight(result, "\n") + "\n"

	return []byte(result)
}

func isHeading(line string) bool {
	trimmed := strings.TrimLeft(line, " ")
	return strings.HasPrefix(trimmed, "#")
}

func normalizeHeading(line string) string {
	trimmed := strings.TrimLeft(line, " ")

	level := 0
	for _, ch := range trimmed {
		if ch == '#' {
			level++
		} else {
			break
		}
	}
	if level > 6 || level == 0 {
		return line
	}

	text := strings.TrimSpace(trimmed[level:])
	// Remove trailing # markers
	text = strings.TrimRight(text, "# ")
	text = strings.TrimSpace(text)

	if text == "" {
		return strings.Repeat("#", level)
	}
	return strings.Repeat("#", level) + " " + text
}

func normalizeBlankLines(lines []string, hasFrontmatter bool) []string {
	var result []string
	consecutiveBlanks := 0
	inFrontmatter := false

	for i, line := range lines {
		// Track frontmatter
		if i == 0 && strings.TrimSpace(line) == "---" {
			inFrontmatter = true
			result = append(result, line)
			consecutiveBlanks = 0
			continue
		}
		if inFrontmatter {
			result = append(result, line)
			if strings.TrimSpace(line) == "---" {
				inFrontmatter = false
			}
			consecutiveBlanks = 0
			continue
		}

		if strings.TrimSpace(line) == "" {
			consecutiveBlanks++
			if consecutiveBlanks <= 2 {
				result = append(result, line)
			}
		} else {
			// Ensure blank line before headings (unless at start or after frontmatter)
			if isHeading(line) && len(result) > 0 {
				lastLine := result[len(result)-1]
				if strings.TrimSpace(lastLine) != "" && !strings.HasPrefix(strings.TrimSpace(lastLine), "---") {
					result = append(result, "")
				}
			}
			consecutiveBlanks = 0
			result = append(result, line)
		}
	}

	return result
}
