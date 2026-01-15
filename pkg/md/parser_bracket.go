// parser_bracket.go parses bracket syntax [MACRO]...[/MACRO] into MacroNode trees.
package md

import (
	"strings"
)

// ParseBracketMacros parses bracket macro syntax and returns a ParseResult.
// Input: markdown with [MACRO]...[/MACRO] syntax
// Output: segments of text and MacroNode trees
func ParseBracketMacros(input string) (*ParseResult, error) {
	tokens, err := TokenizeBrackets(input)
	if err != nil {
		return nil, err
	}

	result := &ParseResult{}
	stack := []*stackFrame{}

	for _, token := range tokens {
		switch token.Type {
		case BracketTokenText:
			if len(stack) > 0 {
				// Inside a macro - accumulate body text
				stack[len(stack)-1].bodyContent += token.Text
			} else {
				// Top level - add as text segment
				result.AddTextSegment(token.Text)
			}

		case BracketTokenOpenTag:
			// Check if this is a known macro
			macroType, known := LookupMacro(token.MacroName)
			if !known {
				// Unknown macro - treat as text
				result.AddWarning("unknown macro: %s", token.MacroName)
				text := reconstructBracketTag(token)
				if len(stack) > 0 {
					stack[len(stack)-1].bodyContent += text
				} else {
					result.AddTextSegment(text)
				}
				continue
			}

			// Create a new stack frame for this macro
			frame := &stackFrame{
				node: &MacroNode{
					Name:       strings.ToLower(token.MacroName),
					Parameters: token.Parameters,
				},
				macroType: macroType,
			}
			stack = append(stack, frame)

			// If macro has no body, close it immediately
			if !macroType.HasBody {
				closeMacro(result, &stack)
			}

		case BracketTokenCloseTag:
			if len(stack) == 0 {
				// Orphan close tag - treat as text
				result.AddWarning("orphan close tag: [/%s]", token.MacroName)
				result.AddTextSegment("[/" + token.MacroName + "]")
				continue
			}

			// Check if close tag matches current open
			current := stack[len(stack)-1]
			if !strings.EqualFold(current.node.Name, token.MacroName) {
				// Mismatched close tag
				result.AddWarning("mismatched close tag: expected [/%s], got [/%s]",
					current.node.Name, token.MacroName)
				// Try to recover by treating as text
				if len(stack) > 1 {
					stack[len(stack)-1].bodyContent += "[/" + token.MacroName + "]"
				} else {
					result.AddTextSegment("[/" + token.MacroName + "]")
				}
				continue
			}

			// Set body content and close macro
			current.node.Body = current.bodyContent
			closeMacro(result, &stack)

		case BracketTokenSelfClose:
			// Self-closing macro (no body)
			macroType, known := LookupMacro(token.MacroName)
			if !known {
				result.AddWarning("unknown macro: %s", token.MacroName)
				text := reconstructBracketTag(token)
				if len(stack) > 0 {
					stack[len(stack)-1].bodyContent += text
				} else {
					result.AddTextSegment(text)
				}
				continue
			}

			node := &MacroNode{
				Name:       strings.ToLower(token.MacroName),
				Parameters: token.Parameters,
			}

			if len(stack) > 0 {
				// Nested - add as child
				stack[len(stack)-1].node.Children = append(
					stack[len(stack)-1].node.Children, node)
			} else {
				// Top level
				result.AddMacroSegment(node)
			}
			_ = macroType // validated but not needed for self-close
		}
	}

	// Handle any unclosed macros
	for len(stack) > 0 {
		current := stack[len(stack)-1]
		result.AddWarning("unclosed macro: [%s]", current.node.Name)
		// Emit as text instead of macro
		text := reconstructOpenTag(current.node) + current.bodyContent
		stack = stack[:len(stack)-1]
		if len(stack) > 0 {
			stack[len(stack)-1].bodyContent += text
		} else {
			result.AddTextSegment(text)
		}
	}

	return result, nil
}

// stackFrame tracks parsing state for nested macros.
type stackFrame struct {
	node        *MacroNode
	macroType   MacroType
	bodyContent string
}

// closeMacro pops the current macro from the stack and adds it appropriately.
func closeMacro(result *ParseResult, stack *[]*stackFrame) {
	if len(*stack) == 0 {
		return
	}

	current := (*stack)[len(*stack)-1]
	*stack = (*stack)[:len(*stack)-1]

	// Parse any nested macros in the body
	if current.node.Body != "" && current.macroType.HasBody {
		nested, err := ParseBracketMacros(current.node.Body)
		if err == nil {
			// Extract child macros
			for _, seg := range nested.Segments {
				if seg.Type == SegmentMacro {
					current.node.Children = append(current.node.Children, seg.Macro)
				}
			}
			// Combine warnings
			result.Warnings = append(result.Warnings, nested.Warnings...)
		}
	}

	if len(*stack) > 0 {
		// Add as child of parent macro
		(*stack)[len(*stack)-1].node.Children = append(
			(*stack)[len(*stack)-1].node.Children, current.node)
	} else {
		// Top level - add as segment
		result.AddMacroSegment(current.node)
	}
}

// reconstructBracketTag rebuilds the original bracket syntax for a token.
func reconstructBracketTag(token BracketToken) string {
	var sb strings.Builder
	sb.WriteString("[")
	if token.Type == BracketTokenCloseTag {
		sb.WriteString("/")
	}
	sb.WriteString(token.MacroName)
	for k, v := range token.Parameters {
		sb.WriteString(" ")
		sb.WriteString(k)
		sb.WriteString("=")
		if strings.Contains(v, " ") {
			sb.WriteString("\"")
			sb.WriteString(v)
			sb.WriteString("\"")
		} else {
			sb.WriteString(v)
		}
	}
	if token.Type == BracketTokenSelfClose {
		sb.WriteString("/")
	}
	sb.WriteString("]")
	return sb.String()
}

// reconstructOpenTag rebuilds the opening tag from a MacroNode.
func reconstructOpenTag(node *MacroNode) string {
	var sb strings.Builder
	sb.WriteString("[")
	sb.WriteString(strings.ToUpper(node.Name))
	for k, v := range node.Parameters {
		sb.WriteString(" ")
		sb.WriteString(k)
		sb.WriteString("=")
		if strings.Contains(v, " ") {
			sb.WriteString("\"")
			sb.WriteString(v)
			sb.WriteString("\"")
		} else {
			sb.WriteString(v)
		}
	}
	sb.WriteString("]")
	return sb.String()
}
