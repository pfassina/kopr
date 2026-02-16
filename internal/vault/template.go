package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Template represents a note template.
type Template struct {
	Name    string
	Path    string
	Content string
}

// LoadTemplates loads all templates from the vault's templates directory.
func (v *Vault) LoadTemplates() ([]Template, error) {
	templateDir := filepath.Join(v.Root, "templates")

	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		return nil, nil
	}

	var templates []Template
	entries, err := os.ReadDir(templateDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		path := filepath.Join(templateDir, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".md")
		templates = append(templates, Template{
			Name:    name,
			Path:    path,
			Content: string(content),
		})
	}

	return templates, nil
}

// ExpandTemplate expands template variables in content.
// Variables:
//
//	{{title}}     - Note title
//	{{date}}      - Current date (YYYY-MM-DD)
//	{{datetime}}  - Current datetime (YYYY-MM-DD HH:MM:SS)
//	{{time}}      - Current time (HH:MM:SS)
//	{{slug}}      - Slugified title
func ExpandTemplate(content, title string) string {
	now := time.Now()

	replacements := map[string]string{
		"{{title}}":    title,
		"{{date}}":     now.Format("2006-01-02"),
		"{{datetime}}": now.Format("2006-01-02 15:04:05"),
		"{{time}}":     now.Format("15:04:05"),
		"{{slug}}":     Slugify(title),
	}

	result := content
	for placeholder, value := range replacements {
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

// CreateFromTemplate creates a new note from a template.
func (v *Vault) CreateFromTemplate(template Template, title string) (string, error) {
	slug := Slugify(title)
	relPath := slug + ".md"

	content := ExpandTemplate(template.Content, title)

	absPath, err := v.CreateNote(relPath, content)
	if err != nil {
		return "", fmt.Errorf("create from template: %w", err)
	}

	return absPath, nil
}
