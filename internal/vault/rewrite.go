package vault

import (
	"os"
	"regexp"
	"strings"
)

// replaceWikiLinkTargets replaces wiki link targets matching oldName with newName.
// Handles: [[old]], [[old.md]], [[old#section]], [[old|alias]], [[old#section|alias]],
// [[old.md#section]], [[old.md|alias]], [[old.md#section|alias]].
func replaceWikiLinkTargets(content, oldName, newName string) string {
	// Match [[oldName]] with optional .md, #section, and |alias
	// The pattern captures: [[ + oldName + optional .md + optional #section + optional |alias + ]]
	escaped := regexp.QuoteMeta(oldName)
	pattern := `\[\[` + escaped + `(\.md)?([#|][^\]]*?)?\]\]`
	re := regexp.MustCompile(pattern)

	return re.ReplaceAllStringFunc(content, func(match string) string {
		// Strip [[ and ]]
		inner := match[2 : len(match)-2]

		// Replace the target name, preserving suffix (.md, #section, |alias)
		var suffix string
		name := inner

		// Check for .md extension
		hasMd := false
		if strings.HasPrefix(name[len(oldName):], ".md") {
			hasMd = true
			suffix = name[len(oldName)+3:]
			name = oldName
		} else {
			suffix = name[len(oldName):]
			name = oldName
		}

		_ = name // name was oldName, replace with newName
		result := newName
		if hasMd {
			result += ".md"
		}
		result += suffix

		return "[[" + result + "]]"
	})
}

// RewriteLinksInNote reads a note file, replaces wiki link targets from oldName
// to newName, and writes it back if any changes were made.
// Returns true if the file was modified.
func RewriteLinksInNote(absPath, oldName, newName string) (bool, error) {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return false, err
	}

	original := string(data)
	updated := replaceWikiLinkTargets(original, oldName, newName)

	if updated == original {
		return false, nil
	}

	if err := os.WriteFile(absPath, []byte(updated), 0644); err != nil {
		return false, err
	}

	return true, nil
}
