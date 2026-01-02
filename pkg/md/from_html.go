package md

import (
	"strings"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
)

// FromConfluenceStorage converts Confluence storage format (XHTML) to markdown.
func FromConfluenceStorage(html string) (string, error) {
	if html == "" {
		return "", nil
	}

	markdown, err := htmltomarkdown.ConvertString(html)
	if err != nil {
		return "", err
	}

	// Clean up the output - trim whitespace
	return strings.TrimSpace(markdown), nil
}
