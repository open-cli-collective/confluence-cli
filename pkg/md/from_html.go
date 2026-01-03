package md

import (
	"regexp"
	"strings"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
)

// ConvertOptions configures the HTML to markdown conversion.
type ConvertOptions struct {
	// ShowMacros shows placeholder text for Confluence macros instead of stripping them.
	ShowMacros bool
}

// FromConfluenceStorage converts Confluence storage format (XHTML) to markdown.
func FromConfluenceStorage(html string) (string, error) {
	return FromConfluenceStorageWithOptions(html, ConvertOptions{})
}

// FromConfluenceStorageWithOptions converts Confluence storage format (XHTML) to markdown
// with configurable options.
func FromConfluenceStorageWithOptions(html string, opts ConvertOptions) (string, error) {
	if html == "" {
		return "", nil
	}

	// Process Confluence macros before conversion
	html = processConfluenceMacros(html, opts.ShowMacros)

	markdown, err := htmltomarkdown.ConvertString(html)
	if err != nil {
		return "", err
	}

	// Clean up the output - trim whitespace
	return strings.TrimSpace(markdown), nil
}

// processConfluenceMacros handles Confluence-specific macro elements.
// If showMacros is false, macros are stripped entirely.
// If showMacros is true, macros are replaced with placeholder text.
func processConfluenceMacros(html string, showMacros bool) string {
	// Pattern to match ac:structured-macro elements
	// These look like: <ac:structured-macro ac:name="toc" ...>...</ac:structured-macro>
	macroPattern := regexp.MustCompile(`<ac:structured-macro[^>]*ac:name="([^"]*)"[^>]*>.*?</ac:structured-macro>`)

	if !showMacros {
		// Strip macros entirely
		html = macroPattern.ReplaceAllString(html, "")
	} else {
		// Replace with placeholder text
		html = macroPattern.ReplaceAllStringFunc(html, func(match string) string {
			nameMatch := regexp.MustCompile(`ac:name="([^"]*)"`).FindStringSubmatch(match)
			if len(nameMatch) > 1 {
				macroName := strings.ToUpper(nameMatch[1])
				return "[" + macroName + "]"
			}
			return "[MACRO]"
		})
	}

	// Also strip or convert other Confluence-specific elements:

	// ac:link elements (internal Confluence links)
	linkPattern := regexp.MustCompile(`<ac:link[^>]*>.*?</ac:link>`)
	html = linkPattern.ReplaceAllString(html, "")

	// ri:page references
	pageRefPattern := regexp.MustCompile(`<ri:page[^>]*/?>`)
	html = pageRefPattern.ReplaceAllString(html, "")

	// ac:plain-text-link-body (link text)
	linkBodyPattern := regexp.MustCompile(`<ac:plain-text-link-body><!\[CDATA\[(.*?)\]\]></ac:plain-text-link-body>`)
	html = linkBodyPattern.ReplaceAllString(html, "$1")

	// ac:parameter elements (macro parameters like minLevel, maxLevel)
	paramPattern := regexp.MustCompile(`<ac:parameter[^>]*>.*?</ac:parameter>`)
	html = paramPattern.ReplaceAllString(html, "")

	// Self-closing ac:parameter
	paramSelfClosingPattern := regexp.MustCompile(`<ac:parameter[^>]*/>`)
	html = paramSelfClosingPattern.ReplaceAllString(html, "")

	return html
}
