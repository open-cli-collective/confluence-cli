// Package md provides markdown conversion utilities for Confluence.
package md

import (
	"bytes"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

// mdParser is a pre-configured goldmark instance with GFM table extension.
var mdParser = goldmark.New(
	goldmark.WithExtensions(extension.Table),
)

// macroPlaceholder is used to mark where macros should be inserted after goldmark processing.
// Using a format that won't be interpreted as markdown formatting (no underscores, asterisks, etc).
const macroPlaceholderPrefix = "CFMACRO"
const macroPlaceholderSuffix = "END"

// ToConfluenceStorage converts markdown content to Confluence storage format (XHTML).
func ToConfluenceStorage(markdown []byte) (string, error) {
	if len(markdown) == 0 {
		return "", nil
	}

	// Preprocess: replace macro placeholders with unique markers
	processed, macros := preprocessMacros(markdown)

	var buf bytes.Buffer
	if err := mdParser.Convert(processed, &buf); err != nil {
		return "", err
	}

	// Postprocess: replace markers with actual macro XML
	result := postprocessMacros(buf.String(), macros)

	return result, nil
}

// preprocessMacros replaces macro placeholders like [TOC] with unique markers.
// Returns the processed markdown and a map of marker IDs to macro XML.
func preprocessMacros(markdown []byte) ([]byte, map[int]string) {
	input := string(markdown)
	macros := make(map[int]string)
	counter := 0

	// Tokenize to find macro patterns and handle them
	tokens, err := TokenizeBrackets(input)
	if err != nil {
		// If tokenization fails, return original markdown unchanged
		return markdown, macros
	}

	var outputBuf strings.Builder
	pos := 0

	for _, token := range tokens {
		// Emit any text before this token
		if token.Position > pos {
			outputBuf.WriteString(input[pos:token.Position])
			pos = token.Position
		}

		switch token.Type {
		case BracketTokenText:
			// Text token - just output it
			outputBuf.WriteString(token.Text)
			pos += len(token.Text)

		case BracketTokenOpenTag:
			// Check if this is a known macro
			macroType, known := LookupMacro(token.MacroName)
			if !known {
				// Unknown macro - leave as-is in original text form
				// Find the end position of this token and output original text
				endPos := findTokenEndPos(input, token)
				outputBuf.WriteString(input[token.Position:endPos])
				pos = endPos
				continue
			}

			// Create macro node from token
			node := &MacroNode{
				Name:       strings.ToLower(token.MacroName),
				Parameters: token.Parameters,
			}

			// Find matching close tag if macro has body
			if macroType.HasBody {
				bodyText, endPos, found := findMacroBody(input, token, tokens)
				if found {
					node.Body = bodyText
					pos = endPos
				} else {
					// No matching close tag found - output original text
					endPos := findTokenEndPos(input, token)
					outputBuf.WriteString(input[token.Position:endPos])
					pos = endPos
					continue
				}
			} else {
				// No body macro - token ends immediately after ]
				endPos := findTokenEndPos(input, token)
				pos = endPos
			}

			// Convert body content from markdown to HTML
			if macroType.HasBody && node.Body != "" {
				var bodyBuf bytes.Buffer
				if err := mdParser.Convert([]byte(node.Body), &bodyBuf); err == nil {
					node.Body = bodyBuf.String()
				} else {
					node.Body = "<p>" + node.Body + "</p>"
				}
			}

			// Render macro to XML
			macroXML := RenderMacroToXML(node)
			macros[counter] = macroXML

			// Insert placeholder
			outputBuf.WriteString(FormatPlaceholder(counter))
			counter++

		case BracketTokenCloseTag, BracketTokenSelfClose:
			// These shouldn't appear at top level in a properly tokenized stream
			// but if they do, treat as text
			endPos := findTokenEndPos(input, token)
			outputBuf.WriteString(input[token.Position:endPos])
			pos = endPos
		}
	}

	// Emit any remaining text
	if pos < len(input) {
		outputBuf.WriteString(input[pos:])
	}

	return []byte(outputBuf.String()), macros
}

// findTokenEndPos finds the end position of a token in the input.
// For text tokens, it's position + length.
// For bracket tokens, we need to find the closing ] or ].
func findTokenEndPos(input string, token BracketToken) int {
	if token.Type == BracketTokenText {
		return token.Position + len(token.Text)
	}
	// For bracket tokens, find the closing ]
	// This is a simplified approach - we look for the next ] after the position
	pos := token.Position + 1
	for pos < len(input) && input[pos] != ']' {
		pos++
	}
	if pos < len(input) {
		pos++ // include the ]
	}
	return pos
}

// findMacroBody searches for the body and closing tag of a macro.
// Returns the body text, end position, and whether a matching close tag was found.
func findMacroBody(input string, openToken BracketToken, tokens []BracketToken) (string, int, bool) {
	// Find opening ] of the open tag
	openTagEnd := findTokenEndPos(input, openToken)

	// Search for close tag after the opening tag - try multiple case variations
	// since macros are case-insensitive
	searchText := input[openTagEnd:]

	// Try to find matching close tag with case-insensitive search
	// We need to look for [/MACRO] or [/macro] or [/Macro] etc.
	closePosLen := 0

	// Try uppercase
	closePatternUpper := "[/" + strings.ToUpper(openToken.MacroName) + "]"
	closePosRel := strings.Index(searchText, closePatternUpper)
	if closePosRel != -1 {
		closePosLen = len(closePatternUpper)
	} else {
		// Try lowercase
		closePatternLower := "[/" + strings.ToLower(openToken.MacroName) + "]"
		closePosRel = strings.Index(searchText, closePatternLower)
		if closePosRel != -1 {
			closePosLen = len(closePatternLower)
		} else {
			// Try case-insensitive search by looking for [/...] pattern and checking macro name
			for i := 0; i < len(searchText)-2; i++ {
				if searchText[i] == '[' && searchText[i+1] == '/' {
					// Found potential close tag, extract name
					j := i + 2
					for j < len(searchText) && searchText[j] != ']' && searchText[j] != ' ' {
						j++
					}
					if j < len(searchText) && searchText[j] == ']' {
						foundName := searchText[i+2 : j]
						if strings.EqualFold(foundName, openToken.MacroName) {
							closePosRel = i
							closePosLen = j - i + 1
							break
						}
					}
				}
			}
		}
	}

	if closePosRel != -1 {
		bodyText := searchText[:closePosRel]
		closePos := openTagEnd + closePosRel + closePosLen
		return bodyText, closePos, true
	}

	return "", openTagEnd, false
}

// postprocessMacros replaces placeholder markers with actual macro XML.
func postprocessMacros(html string, macros map[int]string) string {
	// First pass: resolve any placeholders that exist within other macro values.
	// This handles nested macros (e.g., [TOC] inside [INFO]...[/INFO]).
	// The inner macro placeholder ends up embedded in the outer macro's XML.
	for id, macroXML := range macros {
		for innerId, innerXML := range macros {
			if innerId == id {
				continue
			}
			placeholder := FormatPlaceholder(innerId)
			if strings.Contains(macroXML, placeholder) {
				macros[id] = strings.Replace(macroXML, placeholder, innerXML, 1)
				macroXML = macros[id]
			}
		}
	}

	// Second pass: replace placeholders in the main HTML
	for id, macroXML := range macros {
		placeholder := FormatPlaceholder(id)
		// The placeholder might be wrapped in <p> tags, so handle that
		wrappedPlaceholder := "<p>" + placeholder + "</p>"
		if strings.Contains(html, wrappedPlaceholder) {
			html = strings.Replace(html, wrappedPlaceholder, macroXML, 1)
		} else {
			html = strings.Replace(html, placeholder, macroXML, 1)
		}
	}
	return html
}

// parseKeyValueParams parses a string like "key1=value1 key2=value2" into ["key1=value1", "key2=value2"].
// Handles values with quotes: key="value with spaces"
func parseKeyValueParams(s string) []string {
	var params []string
	var current strings.Builder
	inQuotes := false
	quoteChar := rune(0)

	for i, r := range s {
		switch {
		case (r == '"' || r == '\'') && !inQuotes:
			inQuotes = true
			quoteChar = r
			// Don't include the opening quote in the value
		case r == quoteChar && inQuotes:
			inQuotes = false
			quoteChar = 0
			// Don't include the closing quote in the value
		case r == ' ' && !inQuotes:
			if current.Len() > 0 {
				params = append(params, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}

		// Handle end of string
		if i == len(s)-1 && current.Len() > 0 {
			params = append(params, current.String())
		}
	}

	return params
}
