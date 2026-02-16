package markdown

import (
	"bufio"
	"bytes"
	"strings"
)

// Frontmatter represents YAML frontmatter.
type Frontmatter struct {
	Title   string
	Tags    []string
	Status  string
	Raw     map[string]string
	EndLine int // line number where frontmatter ends (0-based)
}

// ExtractFrontmatter parses YAML frontmatter from markdown content.
// Supports the common --- delimited format.
func ExtractFrontmatter(content []byte) *Frontmatter {
	scanner := bufio.NewScanner(bytes.NewReader(content))

	// First line must be ---
	if !scanner.Scan() {
		return nil
	}
	if strings.TrimSpace(scanner.Text()) != "---" {
		return nil
	}

	fm := &Frontmatter{
		Raw: make(map[string]string),
	}

	lineNum := 1
	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		if strings.TrimSpace(line) == "---" {
			fm.EndLine = lineNum
			break
		}

		// Simple key: value parsing
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		fm.Raw[key] = val

		switch key {
		case "title":
			fm.Title = val
		case "status":
			fm.Status = val
		case "tags":
			// Parse [tag1, tag2] or tag1, tag2
			val = strings.Trim(val, "[]")
			for _, tag := range strings.Split(val, ",") {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					fm.Tags = append(fm.Tags, tag)
				}
			}
		}
	}

	if fm.EndLine == 0 {
		return nil // unclosed frontmatter
	}

	return fm
}
