# Macro Parser Refactor - Phase 4

> **For Claude:** REQUIRED SUB-SKILL: Use ed3d-plan-and-execute:subagent-driven-development to implement this plan task-by-task.

**Goal:** Build MacroNode trees from token streams for both directions

**Architecture:** Stack-based parser consuming tokens, producing Segment sequences with MacroNode trees

**Tech Stack:** Go, testify for assertions

**Scope:** Phase 4 of 6

**Codebase verified:** 2026-01-15

---

## Phase 4: Parser Implementation

### Task 1: Create parser.go with ParseResult and Segment types

**Files:**
- Create: `pkg/md/parser.go`

**Step 1: Create the parser types file**

```go
// parser.go defines shared types for macro parsing in both directions.
package md

import (
	"log"
)

// SegmentType indicates whether a segment is text or a macro.
type SegmentType int

const (
	SegmentText  SegmentType = iota // plain text/HTML content
	SegmentMacro                    // parsed macro node
)

// Segment represents either text content or a parsed macro.
type Segment struct {
	Type  SegmentType
	Text  string     // set when Type == SegmentText
	Macro *MacroNode // set when Type == SegmentMacro
}

// ParseResult contains the parsed output: a sequence of segments
// that alternate between text content and macro nodes.
type ParseResult struct {
	Segments []Segment
	Warnings []string // any warnings generated during parsing
}

// AddTextSegment appends a text segment, merging with previous text if possible.
func (pr *ParseResult) AddTextSegment(text string) {
	if text == "" {
		return
	}
	// Merge adjacent text segments
	if len(pr.Segments) > 0 && pr.Segments[len(pr.Segments)-1].Type == SegmentText {
		pr.Segments[len(pr.Segments)-1].Text += text
		return
	}
	pr.Segments = append(pr.Segments, Segment{
		Type: SegmentText,
		Text: text,
	})
}

// AddMacroSegment appends a macro segment.
func (pr *ParseResult) AddMacroSegment(macro *MacroNode) {
	pr.Segments = append(pr.Segments, Segment{
		Type:  SegmentMacro,
		Macro: macro,
	})
}

// AddWarning logs a warning and stores it in the result.
func (pr *ParseResult) AddWarning(format string, args ...interface{}) {
	msg := format
	if len(args) > 0 {
		msg = log.Prefix() + format // simple formatting
	}
	pr.Warnings = append(pr.Warnings, msg)
	log.Printf("WARN: "+format, args...)
}

// GetMacros returns all MacroNodes from the parse result.
func (pr *ParseResult) GetMacros() []*MacroNode {
	var macros []*MacroNode
	for _, seg := range pr.Segments {
		if seg.Type == SegmentMacro && seg.Macro != nil {
			macros = append(macros, seg.Macro)
		}
	}
	return macros
}
```

**Step 2: Verify it compiles**

Run: `go build ./pkg/md/`
Expected: No errors

**Step 3: Commit**

```bash
git add pkg/md/parser.go
git commit -m "feat(md): add ParseResult and Segment types

Foundation for parser output - alternating text/macro segments.
Supports warning collection for malformed input handling."
```

---

### Task 2: Create parser_bracket.go for MD→XHTML direction

**Files:**
- Create: `pkg/md/parser_bracket.go`

**Step 1: Create bracket parser**

```go
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
```

**Step 2: Verify it compiles**

Run: `go build ./pkg/md/`
Expected: No errors

**Step 3: Commit**

```bash
git add pkg/md/parser_bracket.go
git commit -m "feat(md): add bracket syntax parser

ParseBracketMacros() builds MacroNode trees from bracket tokens.
Handles nesting, unknown macros, unclosed tags with warnings."
```

---

### Task 3: Create parser_xml.go for XHTML→MD direction

**Files:**
- Create: `pkg/md/parser_xml.go`

**Step 1: Create XML parser**

```go
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
```

**Step 2: Verify it compiles**

Run: `go build ./pkg/md/`
Expected: No errors

**Step 3: Commit**

```bash
git add pkg/md/parser_xml.go
git commit -m "feat(md): add Confluence XML parser

ParseConfluenceXML() builds MacroNode trees from XML tokens.
Handles nesting, parameters, rich-text and plain-text bodies."
```

---

### Task 4: Create comprehensive parser tests

**Files:**
- Create: `pkg/md/parser_test.go`

**Step 1: Create parser test file**

```go
package md

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== Bracket Parser Tests ====================

func TestParseBracketMacros_EmptyInput(t *testing.T) {
	result, err := ParseBracketMacros("")
	require.NoError(t, err)
	assert.Empty(t, result.Segments)
}

func TestParseBracketMacros_PlainText(t *testing.T) {
	result, err := ParseBracketMacros("Hello world")
	require.NoError(t, err)
	require.Len(t, result.Segments, 1)
	assert.Equal(t, SegmentText, result.Segments[0].Type)
	assert.Equal(t, "Hello world", result.Segments[0].Text)
}

func TestParseBracketMacros_SimpleTOC(t *testing.T) {
	result, err := ParseBracketMacros("[TOC]")
	require.NoError(t, err)
	require.Len(t, result.Segments, 1)
	assert.Equal(t, SegmentMacro, result.Segments[0].Type)
	assert.Equal(t, "toc", result.Segments[0].Macro.Name)
}

func TestParseBracketMacros_TOCWithParams(t *testing.T) {
	result, err := ParseBracketMacros("[TOC maxLevel=3 minLevel=1]")
	require.NoError(t, err)
	require.Len(t, result.Segments, 1)
	macro := result.Segments[0].Macro
	assert.Equal(t, "toc", macro.Name)
	assert.Equal(t, "3", macro.Parameters["maxLevel"])
	assert.Equal(t, "1", macro.Parameters["minLevel"])
}

func TestParseBracketMacros_PanelWithBody(t *testing.T) {
	result, err := ParseBracketMacros("[INFO]Content here[/INFO]")
	require.NoError(t, err)
	require.Len(t, result.Segments, 1)
	macro := result.Segments[0].Macro
	assert.Equal(t, "info", macro.Name)
	assert.Equal(t, "Content here", macro.Body)
}

func TestParseBracketMacros_PanelWithTitleAndBody(t *testing.T) {
	result, err := ParseBracketMacros(`[WARNING title="Watch Out"]Be careful![/WARNING]`)
	require.NoError(t, err)
	require.Len(t, result.Segments, 1)
	macro := result.Segments[0].Macro
	assert.Equal(t, "warning", macro.Name)
	assert.Equal(t, "Watch Out", macro.Parameters["title"])
	assert.Equal(t, "Be careful!", macro.Body)
}

func TestParseBracketMacros_NestedMacros(t *testing.T) {
	result, err := ParseBracketMacros("[INFO]Before [TOC] after[/INFO]")
	require.NoError(t, err)
	require.Len(t, result.Segments, 1)
	macro := result.Segments[0].Macro
	assert.Equal(t, "info", macro.Name)
	// TOC should be a child
	require.Len(t, macro.Children, 1)
	assert.Equal(t, "toc", macro.Children[0].Name)
}

func TestParseBracketMacros_MultipleMacros(t *testing.T) {
	result, err := ParseBracketMacros("Before [TOC] middle [INFO]content[/INFO] after")
	require.NoError(t, err)

	// Should have: text, macro, text, macro, text
	textCount := 0
	macroCount := 0
	for _, seg := range result.Segments {
		if seg.Type == SegmentText {
			textCount++
		} else {
			macroCount++
		}
	}
	assert.Equal(t, 2, macroCount, "should have 2 macros")
	assert.GreaterOrEqual(t, textCount, 2, "should have at least 2 text segments")
}

func TestParseBracketMacros_UnknownMacro(t *testing.T) {
	result, err := ParseBracketMacros("[UNKNOWN]content[/UNKNOWN]")
	require.NoError(t, err)
	// Unknown macro should be treated as text
	assert.GreaterOrEqual(t, len(result.Warnings), 1, "should have warning")
	// Content should be in text segments
	hasText := false
	for _, seg := range result.Segments {
		if seg.Type == SegmentText {
			hasText = true
		}
	}
	assert.True(t, hasText, "unknown macro should be preserved as text")
}

func TestParseBracketMacros_MismatchedClose(t *testing.T) {
	result, err := ParseBracketMacros("[INFO]content[/WARNING]more[/INFO]")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(result.Warnings), 1, "should have warning for mismatch")
}

func TestParseBracketMacros_UnclosedMacro(t *testing.T) {
	result, err := ParseBracketMacros("[INFO]content without close")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(result.Warnings), 1, "should have warning for unclosed")
	// Should be treated as text
	hasText := false
	for _, seg := range result.Segments {
		if seg.Type == SegmentText {
			hasText = true
		}
	}
	assert.True(t, hasText, "unclosed macro should be preserved as text")
}

// ==================== XML Parser Tests ====================

func TestParseConfluenceXML_EmptyInput(t *testing.T) {
	result, err := ParseConfluenceXML("")
	require.NoError(t, err)
	assert.Empty(t, result.Segments)
}

func TestParseConfluenceXML_PlainHTML(t *testing.T) {
	result, err := ParseConfluenceXML("<p>Hello world</p>")
	require.NoError(t, err)
	require.Len(t, result.Segments, 1)
	assert.Equal(t, SegmentText, result.Segments[0].Type)
	assert.Equal(t, "<p>Hello world</p>", result.Segments[0].Text)
}

func TestParseConfluenceXML_SimpleTOC(t *testing.T) {
	input := `<ac:structured-macro ac:name="toc" ac:schema-version="1"></ac:structured-macro>`
	result, err := ParseConfluenceXML(input)
	require.NoError(t, err)
	require.Len(t, result.Segments, 1)
	assert.Equal(t, SegmentMacro, result.Segments[0].Type)
	assert.Equal(t, "toc", result.Segments[0].Macro.Name)
}

func TestParseConfluenceXML_TOCWithParams(t *testing.T) {
	input := `<ac:structured-macro ac:name="toc" ac:schema-version="1"><ac:parameter ac:name="maxLevel">3</ac:parameter><ac:parameter ac:name="minLevel">1</ac:parameter></ac:structured-macro>`
	result, err := ParseConfluenceXML(input)
	require.NoError(t, err)
	require.Len(t, result.Segments, 1)
	macro := result.Segments[0].Macro
	assert.Equal(t, "toc", macro.Name)
	assert.Equal(t, "3", macro.Parameters["maxLevel"])
	assert.Equal(t, "1", macro.Parameters["minLevel"])
}

func TestParseConfluenceXML_PanelWithBody(t *testing.T) {
	input := `<ac:structured-macro ac:name="info" ac:schema-version="1"><ac:rich-text-body><p>Content</p></ac:rich-text-body></ac:structured-macro>`
	result, err := ParseConfluenceXML(input)
	require.NoError(t, err)
	require.Len(t, result.Segments, 1)
	macro := result.Segments[0].Macro
	assert.Equal(t, "info", macro.Name)
	assert.Contains(t, macro.Body, "Content")
}

func TestParseConfluenceXML_CodeWithCDATA(t *testing.T) {
	input := `<ac:structured-macro ac:name="code" ac:schema-version="1"><ac:parameter ac:name="language">python</ac:parameter><ac:plain-text-body><![CDATA[print("Hello")]]></ac:plain-text-body></ac:structured-macro>`
	result, err := ParseConfluenceXML(input)
	require.NoError(t, err)
	require.Len(t, result.Segments, 1)
	macro := result.Segments[0].Macro
	assert.Equal(t, "code", macro.Name)
	assert.Equal(t, "python", macro.Parameters["language"])
	assert.Contains(t, macro.Body, `print("Hello")`)
}

func TestParseConfluenceXML_NestedMacros(t *testing.T) {
	input := `<ac:structured-macro ac:name="info" ac:schema-version="1"><ac:rich-text-body><ac:structured-macro ac:name="toc" ac:schema-version="1"></ac:structured-macro></ac:rich-text-body></ac:structured-macro>`
	result, err := ParseConfluenceXML(input)
	require.NoError(t, err)
	require.Len(t, result.Segments, 1)
	macro := result.Segments[0].Macro
	assert.Equal(t, "info", macro.Name)
	require.Len(t, macro.Children, 1)
	assert.Equal(t, "toc", macro.Children[0].Name)
}

func TestParseConfluenceXML_WithSurroundingHTML(t *testing.T) {
	input := `<h1>Title</h1><ac:structured-macro ac:name="toc" ac:schema-version="1"></ac:structured-macro><p>Content</p>`
	result, err := ParseConfluenceXML(input)
	require.NoError(t, err)

	// Should have text, macro, text
	textCount := 0
	macroCount := 0
	for _, seg := range result.Segments {
		if seg.Type == SegmentText {
			textCount++
		} else {
			macroCount++
		}
	}
	assert.Equal(t, 1, macroCount)
	assert.Equal(t, 2, textCount)
}

// ==================== Segment Tests ====================

func TestParseResult_GetMacros(t *testing.T) {
	result := &ParseResult{}
	result.AddTextSegment("text1")
	result.AddMacroSegment(&MacroNode{Name: "toc"})
	result.AddTextSegment("text2")
	result.AddMacroSegment(&MacroNode{Name: "info"})

	macros := result.GetMacros()
	require.Len(t, macros, 2)
	assert.Equal(t, "toc", macros[0].Name)
	assert.Equal(t, "info", macros[1].Name)
}

func TestParseResult_MergeAdjacentText(t *testing.T) {
	result := &ParseResult{}
	result.AddTextSegment("hello ")
	result.AddTextSegment("world")

	// Should be merged into single segment
	require.Len(t, result.Segments, 1)
	assert.Equal(t, "hello world", result.Segments[0].Text)
}
```

**Step 2: Run parser tests**

Run: `go test -v ./pkg/md/ -run "TestParseBracket|TestParseConfluence|TestParseResult"`
Expected: All tests pass

**Step 3: Commit**

```bash
git add pkg/md/parser_test.go
git commit -m "test(md): add comprehensive parser tests

Tests cover both bracket and XML parsers:
- Simple macros, parameters, bodies
- Nested macros, multiple macros
- Unknown macros, malformed input
- Segment merging"
```

---

### Task 5: Run full test suite

**Step 1: Run all pkg/md tests**

Run: `go test -v ./pkg/md/...`
Expected: All tests pass

**Step 2: Run with race detection**

Run: `go test -race ./...`
Expected: All tests pass
