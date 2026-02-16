package markdown

import (
	"bufio"
	"bytes"
	"strings"
)

// WikiLink represents a parsed [[wiki link]].
type WikiLink struct {
	Target  string // note name/path
	Section string // #section (if present)
	Alias   string // |alias (if present)
	Line    int    // 1-based line number
	Col     int    // 0-based column
}

// ExtractWikiLinks finds all [[wiki links]] in markdown content.
// Supports [[note]], [[note#section]], [[note|alias]], [[note#section|alias]].
func ExtractWikiLinks(content []byte) []WikiLink {
	var links []WikiLink
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

		// Find all [[ ]] in the line
		col := 0
		for col < len(line)-3 {
			idx := strings.Index(line[col:], "[[")
			if idx == -1 {
				break
			}
			start := col + idx + 2

			end := strings.Index(line[start:], "]]")
			if end == -1 {
				break
			}

			inner := line[start : start+end]
			if inner == "" {
				col = start + end + 2
				continue
			}

			link := WikiLink{
				Line: lineNum,
				Col:  col + idx,
			}

			// Parse section: note#section
			if hashIdx := strings.Index(inner, "#"); hashIdx != -1 {
				link.Target = inner[:hashIdx]
				rest := inner[hashIdx+1:]
				// Parse alias: section|alias
				if pipeIdx := strings.Index(rest, "|"); pipeIdx != -1 {
					link.Section = rest[:pipeIdx]
					link.Alias = rest[pipeIdx+1:]
				} else {
					link.Section = rest
				}
			} else if pipeIdx := strings.Index(inner, "|"); pipeIdx != -1 {
				// Parse alias: note|alias
				link.Target = inner[:pipeIdx]
				link.Alias = inner[pipeIdx+1:]
			} else {
				link.Target = inner
			}

			link.Target = strings.TrimSpace(link.Target)
			link.Section = strings.TrimSpace(link.Section)
			link.Alias = strings.TrimSpace(link.Alias)

			links = append(links, link)
			col = start + end + 2
		}
	}

	return links
}
