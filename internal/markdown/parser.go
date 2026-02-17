package markdown

import (
	"bytes"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"
)

// Parser wraps goldmark for markdown processing.
type Parser struct {
	md goldmark.Markdown
}

func NewParser() *Parser {
	return &Parser{
		md: goldmark.New(),
	}
}

// Parse parses markdown content and returns a parsed document.
func (p *Parser) Parse(content []byte) *ParsedNote {
	reader := text.NewReader(content)
	doc := p.md.Parser().Parse(reader)

	note := &ParsedNote{
		Content: content,
	}

	note.Frontmatter = ExtractFrontmatter(content)
	note.Headings = ExtractHeadings(content)
	note.WikiLinks = ExtractWikiLinks(content)

	_ = doc // goldmark AST available for future use
	return note
}

// ParsedNote contains extracted metadata from a markdown file.
type ParsedNote struct {
	Content     []byte
	Frontmatter *Frontmatter
	Headings    []Heading
	WikiLinks   []WikiLink
}

// PlainContent returns the note content without frontmatter.
func (pn *ParsedNote) PlainContent() string {
	if pn.Frontmatter != nil && pn.Frontmatter.EndLine > 0 {
		lines := bytes.Split(pn.Content, []byte("\n"))
		if pn.Frontmatter.EndLine < len(lines) {
			return string(bytes.Join(lines[pn.Frontmatter.EndLine:], []byte("\n")))
		}
	}
	return string(pn.Content)
}
