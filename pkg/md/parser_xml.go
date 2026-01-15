// parser_xml.go parses Confluence XML into MacroNode trees.
package md

import (
	"strings"
)

// ParseConfluenceXML parses Confluence storage format XML and returns a ParseResult.
// Input: XHTML with <ac:structured-macro> elements
// Output: segments of text/HTML and MacroNode trees
func ParseConfluenceXML(input string) (*ParseResult, error) {
	tokens, err := TokenizeConfluenceXML(input)
	if err != nil {
		return nil, err
	}

	result := &ParseResult{}
	stack := []*xmlStackFrame{}

	for i := 0; i < len(tokens); i++ {
		token := tokens[i]

		switch token.Type {
		case XMLTokenText:
			if len(stack) > 0 {
				// Inside a macro body - accumulate content
				stack[len(stack)-1].bodyContent += token.Text
			} else {
				// Top level - add as text segment
				result.AddTextSegment(token.Text)
			}

		case XMLTokenOpenTag:
			// Check if this is a known macro
			_, known := LookupMacro(token.MacroName)
			if !known {
				result.AddWarning("unknown macro: %s", token.MacroName)
			}

			// Create a new stack frame
			frame := &xmlStackFrame{
				node: &MacroNode{
					Name:       strings.ToLower(token.MacroName),
					Parameters: make(map[string]string),
				},
			}
			stack = append(stack, frame)

		case XMLTokenParameter:
			if len(stack) > 0 && !stack[len(stack)-1].inBody {
				// Parameter belongs to current macro (before body)
				stack[len(stack)-1].node.Parameters[token.ParamName] = token.Value
			}
			// Parameters inside body are part of nested macros, handled separately

		case XMLTokenBody:
			if len(stack) > 0 {
				stack[len(stack)-1].inBody = true
				stack[len(stack)-1].bodyType = token.Value
			}

		case XMLTokenBodyEnd:
			if len(stack) > 0 {
				current := stack[len(stack)-1]
				current.node.Body = current.bodyContent
				current.inBody = false
			}

		case XMLTokenCloseTag:
			if len(stack) == 0 {
				result.AddWarning("orphan close tag at position %d", token.Position)
				continue
			}

			// Pop and finalize the current macro
			current := stack[len(stack)-1]
			stack = stack[:len(stack)-1]

			// If body has nested macros, parse them recursively
			if current.node.Body != "" {
				nested, err := ParseConfluenceXML(current.node.Body)
				if err == nil {
					for _, seg := range nested.Segments {
						if seg.Type == SegmentMacro {
							current.node.Children = append(current.node.Children, seg.Macro)
						}
					}
					result.Warnings = append(result.Warnings, nested.Warnings...)
				}
			}

			if len(stack) > 0 {
				// Add as child of parent macro's body
				stack[len(stack)-1].node.Children = append(
					stack[len(stack)-1].node.Children, current.node)
				// Don't add to bodyContent - children are separate
			} else {
				// Top level - add as segment
				result.AddMacroSegment(current.node)
			}
		}
	}

	// Handle any unclosed macros (malformed XML)
	for len(stack) > 0 {
		current := stack[len(stack)-1]
		result.AddWarning("unclosed macro: %s", current.node.Name)
		stack = stack[:len(stack)-1]
		// Treat unclosed macro as text
		if len(stack) > 0 {
			// Add to parent body as-is (can't reconstruct XML properly)
		} else {
			result.AddMacroSegment(current.node) // best effort
		}
	}

	return result, nil
}

// xmlStackFrame tracks parsing state for nested XML macros.
type xmlStackFrame struct {
	node        *MacroNode
	inBody      bool
	bodyType    string // "rich-text" or "plain-text"
	bodyContent string
}
