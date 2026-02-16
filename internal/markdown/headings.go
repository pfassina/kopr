package markdown

import (
	"bufio"
	"bytes"
	"strings"
)

// Heading represents a markdown heading.
type Heading struct {
	Level int
	Text  string
	Line  int // 1-based line number
}

// ExtractHeadings extracts all ATX headings from markdown content.
func ExtractHeadings(content []byte) []Heading {
	var headings []Heading
	scanner := bufio.NewScanner(bytes.NewReader(content))

	inFrontmatter := false
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip frontmatter
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

		// Match ATX headings: # Heading
		trimmed := strings.TrimLeft(line, " ")
		if !strings.HasPrefix(trimmed, "#") {
			continue
		}

		level := 0
		for _, ch := range trimmed {
			if ch == '#' {
				level++
			} else {
				break
			}
		}

		if level > 6 || level == 0 {
			continue
		}

		text := strings.TrimSpace(trimmed[level:])
		// Remove trailing # markers
		text = strings.TrimRight(text, "# ")
		text = strings.TrimSpace(text)

		if text != "" {
			headings = append(headings, Heading{
				Level: level,
				Text:  text,
				Line:  lineNum,
			})
		}
	}

	return headings
}
