# Macro Parser Refactor - Phase 3

> **For Claude:** REQUIRED SUB-SKILL: Use ed3d-plan-and-execute:subagent-driven-development to implement this plan task-by-task.

**Goal:** Tokenize Confluence XML storage format for XHTML→MD direction

**Architecture:** Regex-based scanner for Confluence ac: namespaced elements

**Tech Stack:** Go, regexp, testify for assertions

**Scope:** Phase 3 of 6

**Codebase verified:** 2026-01-15

---

## Phase 3: Confluence XML Tokenizer

### Task 1: Create tokenizer_xml.go with TokenizeConfluenceXML function

**Files:**
- Create: `pkg/md/tokenizer_xml.go`

**Step 1: Create the tokenizer file**

```go
// tokenizer_xml.go implements tokenization for Confluence storage format XML.
package md

import (
	"fmt"
	"regexp"
	"strings"
)

// Regex patterns for Confluence XML elements
var (
	// Matches <ac:structured-macro ac:name="NAME" ...>
	macroOpenPattern = regexp.MustCompile(`<ac:structured-macro[^>]*ac:name="([^"]*)"[^>]*>`)
	// Matches </ac:structured-macro>
	macroClosePattern = regexp.MustCompile(`</ac:structured-macro>`)
	// Matches <ac:parameter ac:name="NAME">VALUE</ac:parameter>
	paramPattern = regexp.MustCompile(`<ac:parameter[^>]*ac:name="([^"]*)"[^>]*>([^<]*)</ac:parameter>`)
	// Matches <ac:rich-text-body> opening
	richTextBodyOpen = regexp.MustCompile(`<ac:rich-text-body>`)
	// Matches </ac:rich-text-body> closing
	richTextBodyClose = regexp.MustCompile(`</ac:rich-text-body>`)
	// Matches <ac:plain-text-body> opening
	plainTextBodyOpen = regexp.MustCompile(`<ac:plain-text-body>`)
	// Matches </ac:plain-text-body> closing
	plainTextBodyClose = regexp.MustCompile(`</ac:plain-text-body>`)
	// Matches CDATA content: <![CDATA[...]]>
	cdataPattern = regexp.MustCompile(`(?s)<!\[CDATA\[(.*?)\]\]>`)
)

// TokenizeConfluenceXML scans input for Confluence storage format macros and returns a token stream.
// This tokenizer produces a flat stream of tokens that the parser will assemble into a tree.
func TokenizeConfluenceXML(input string) ([]XMLToken, error) {
	var tokens []XMLToken
	pos := 0

	for pos < len(input) {
		// Try to find the next macro or body tag
		remaining := input[pos:]

		// Check for macro open tag
		if loc := macroOpenPattern.FindStringSubmatchIndex(remaining); loc != nil && loc[0] == 0 {
			macroName := remaining[loc[2]:loc[3]]
			tokens = append(tokens, XMLToken{
				Type:      XMLTokenOpenTag,
				MacroName: strings.ToLower(macroName),
				Position:  pos,
			})
			pos += loc[1]
			continue
		}

		// Check for macro close tag
		if loc := macroClosePattern.FindStringIndex(remaining); loc != nil && loc[0] == 0 {
			tokens = append(tokens, XMLToken{
				Type:     XMLTokenCloseTag,
				Position: pos,
			})
			pos += loc[1]
			continue
		}

		// Check for parameter
		if loc := paramPattern.FindStringSubmatchIndex(remaining); loc != nil && loc[0] == 0 {
			paramName := remaining[loc[2]:loc[3]]
			paramValue := remaining[loc[4]:loc[5]]
			tokens = append(tokens, XMLToken{
				Type:      XMLTokenParameter,
				ParamName: paramName,
				Value:     paramValue,
				Position:  pos,
			})
			pos += loc[1]
			continue
		}

		// Check for rich-text-body open
		if loc := richTextBodyOpen.FindStringIndex(remaining); loc != nil && loc[0] == 0 {
			tokens = append(tokens, XMLToken{
				Type:     XMLTokenBody,
				Value:    "rich-text",
				Position: pos,
			})
			pos += loc[1]
			continue
		}

		// Check for rich-text-body close
		if loc := richTextBodyClose.FindStringIndex(remaining); loc != nil && loc[0] == 0 {
			tokens = append(tokens, XMLToken{
				Type:     XMLTokenBodyEnd,
				Value:    "rich-text",
				Position: pos,
			})
			pos += loc[1]
			continue
		}

		// Check for plain-text-body open
		if loc := plainTextBodyOpen.FindStringIndex(remaining); loc != nil && loc[0] == 0 {
			tokens = append(tokens, XMLToken{
				Type:     XMLTokenBody,
				Value:    "plain-text",
				Position: pos,
			})
			pos += loc[1]
			continue
		}

		// Check for plain-text-body close
		if loc := plainTextBodyClose.FindStringIndex(remaining); loc != nil && loc[0] == 0 {
			tokens = append(tokens, XMLToken{
				Type:     XMLTokenBodyEnd,
				Value:    "plain-text",
				Position: pos,
			})
			pos += loc[1]
			continue
		}

		// Check for CDATA (inside plain-text-body)
		if loc := cdataPattern.FindStringSubmatchIndex(remaining); loc != nil && loc[0] == 0 {
			cdataContent := remaining[loc[2]:loc[3]]
			tokens = append(tokens, XMLToken{
				Type:     XMLTokenText,
				Text:     cdataContent,
				Position: pos,
			})
			pos += loc[1]
			continue
		}

		// Find the next macro-related tag
		nextMacroOpen := macroOpenPattern.FindStringIndex(remaining)
		nextMacroClose := macroClosePattern.FindStringIndex(remaining)
		nextParam := paramPattern.FindStringIndex(remaining)
		nextRichOpen := richTextBodyOpen.FindStringIndex(remaining)
		nextRichClose := richTextBodyClose.FindStringIndex(remaining)
		nextPlainOpen := plainTextBodyOpen.FindStringIndex(remaining)
		nextPlainClose := plainTextBodyClose.FindStringIndex(remaining)

		// Find minimum positive start position
		nextTagPos := len(remaining)
		for _, loc := range [][]int{nextMacroOpen, nextMacroClose, nextParam, nextRichOpen, nextRichClose, nextPlainOpen, nextPlainClose} {
			if loc != nil && loc[0] > 0 && loc[0] < nextTagPos {
				nextTagPos = loc[0]
			}
		}

		if nextTagPos > 0 {
			// Emit text up to next tag
			tokens = append(tokens, XMLToken{
				Type:     XMLTokenText,
				Text:     remaining[:nextTagPos],
				Position: pos,
			})
			pos += nextTagPos
		} else {
			// Single character fallback (shouldn't normally happen)
			pos++
		}
	}

	return tokens, nil
}

// ExtractCDATAContent extracts content from a CDATA section.
// Input: "<![CDATA[content]]>" Output: "content"
func ExtractCDATAContent(s string) string {
	if match := cdataPattern.FindStringSubmatch(s); match != nil {
		return match[1]
	}
	return s
}
```

**Step 2: Verify it compiles**

Run: `go build ./pkg/md/`
Expected: No errors

**Step 3: Commit**

```bash
git add pkg/md/tokenizer_xml.go
git commit -m "feat(md): add Confluence XML tokenizer

TokenizeConfluenceXML() scans <ac:structured-macro> syntax.
Handles macro open/close, parameters, rich-text/plain-text bodies, CDATA."
```

---

### Task 2: Create comprehensive tokenizer tests

**Files:**
- Create: `pkg/md/tokenizer_xml_test.go`

**Step 1: Create test file**

```go
package md

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenizeConfluenceXML_EmptyInput(t *testing.T) {
	tokens, err := TokenizeConfluenceXML("")
	require.NoError(t, err)
	assert.Empty(t, tokens)
}

func TestTokenizeConfluenceXML_PlainHTML(t *testing.T) {
	input := "<p>Hello world</p>"
	tokens, err := TokenizeConfluenceXML(input)
	require.NoError(t, err)
	require.Len(t, tokens, 1)
	assert.Equal(t, XMLTokenText, tokens[0].Type)
	assert.Equal(t, input, tokens[0].Text)
}

func TestTokenizeConfluenceXML_SimpleMacro(t *testing.T) {
	input := `<ac:structured-macro ac:name="toc" ac:schema-version="1"></ac:structured-macro>`
	tokens, err := TokenizeConfluenceXML(input)
	require.NoError(t, err)
	require.Len(t, tokens, 2)

	assert.Equal(t, XMLTokenOpenTag, tokens[0].Type)
	assert.Equal(t, "toc", tokens[0].MacroName)

	assert.Equal(t, XMLTokenCloseTag, tokens[1].Type)
}

func TestTokenizeConfluenceXML_MacroWithParameter(t *testing.T) {
	input := `<ac:structured-macro ac:name="toc" ac:schema-version="1"><ac:parameter ac:name="maxLevel">3</ac:parameter></ac:structured-macro>`
	tokens, err := TokenizeConfluenceXML(input)
	require.NoError(t, err)
	require.Len(t, tokens, 3)

	assert.Equal(t, XMLTokenOpenTag, tokens[0].Type)
	assert.Equal(t, "toc", tokens[0].MacroName)

	assert.Equal(t, XMLTokenParameter, tokens[1].Type)
	assert.Equal(t, "maxLevel", tokens[1].ParamName)
	assert.Equal(t, "3", tokens[1].Value)

	assert.Equal(t, XMLTokenCloseTag, tokens[2].Type)
}

func TestTokenizeConfluenceXML_MacroWithMultipleParameters(t *testing.T) {
	input := `<ac:structured-macro ac:name="toc" ac:schema-version="1"><ac:parameter ac:name="maxLevel">3</ac:parameter><ac:parameter ac:name="minLevel">1</ac:parameter><ac:parameter ac:name="type">flat</ac:parameter></ac:structured-macro>`
	tokens, err := TokenizeConfluenceXML(input)
	require.NoError(t, err)

	// Open, 3 params, close = 5 tokens
	require.Len(t, tokens, 5)

	assert.Equal(t, XMLTokenParameter, tokens[1].Type)
	assert.Equal(t, "maxLevel", tokens[1].ParamName)
	assert.Equal(t, "3", tokens[1].Value)

	assert.Equal(t, XMLTokenParameter, tokens[2].Type)
	assert.Equal(t, "minLevel", tokens[2].ParamName)
	assert.Equal(t, "1", tokens[2].Value)

	assert.Equal(t, XMLTokenParameter, tokens[3].Type)
	assert.Equal(t, "type", tokens[3].ParamName)
	assert.Equal(t, "flat", tokens[3].Value)
}

func TestTokenizeConfluenceXML_PanelWithRichTextBody(t *testing.T) {
	input := `<ac:structured-macro ac:name="info" ac:schema-version="1"><ac:rich-text-body><p>Content</p></ac:rich-text-body></ac:structured-macro>`
	tokens, err := TokenizeConfluenceXML(input)
	require.NoError(t, err)

	// Open, body open, text, body close, close = 5 tokens
	require.Len(t, tokens, 5)

	assert.Equal(t, XMLTokenOpenTag, tokens[0].Type)
	assert.Equal(t, "info", tokens[0].MacroName)

	assert.Equal(t, XMLTokenBody, tokens[1].Type)
	assert.Equal(t, "rich-text", tokens[1].Value)

	assert.Equal(t, XMLTokenText, tokens[2].Type)
	assert.Equal(t, "<p>Content</p>", tokens[2].Text)

	assert.Equal(t, XMLTokenBodyEnd, tokens[3].Type)
	assert.Equal(t, "rich-text", tokens[3].Value)

	assert.Equal(t, XMLTokenCloseTag, tokens[4].Type)
}

func TestTokenizeConfluenceXML_PanelWithTitleAndBody(t *testing.T) {
	input := `<ac:structured-macro ac:name="warning" ac:schema-version="1"><ac:parameter ac:name="title">Watch Out</ac:parameter><ac:rich-text-body><p>Warning content</p></ac:rich-text-body></ac:structured-macro>`
	tokens, err := TokenizeConfluenceXML(input)
	require.NoError(t, err)

	// Open, param, body open, text, body close, close = 6 tokens
	require.Len(t, tokens, 6)

	assert.Equal(t, "warning", tokens[0].MacroName)
	assert.Equal(t, "title", tokens[1].ParamName)
	assert.Equal(t, "Watch Out", tokens[1].Value)
	assert.Equal(t, XMLTokenBody, tokens[2].Type)
	assert.Contains(t, tokens[3].Text, "Warning content")
}

func TestTokenizeConfluenceXML_CodeMacroWithCDATA(t *testing.T) {
	input := `<ac:structured-macro ac:name="code" ac:schema-version="1"><ac:parameter ac:name="language">python</ac:parameter><ac:plain-text-body><![CDATA[print("Hello")]]></ac:plain-text-body></ac:structured-macro>`
	tokens, err := TokenizeConfluenceXML(input)
	require.NoError(t, err)

	// Open, param, body open, text (CDATA content), body close, close = 6 tokens
	require.Len(t, tokens, 6)

	assert.Equal(t, "code", tokens[0].MacroName)
	assert.Equal(t, "language", tokens[1].ParamName)
	assert.Equal(t, "python", tokens[1].Value)

	assert.Equal(t, XMLTokenBody, tokens[2].Type)
	assert.Equal(t, "plain-text", tokens[2].Value)

	assert.Equal(t, XMLTokenText, tokens[3].Type)
	assert.Equal(t, `print("Hello")`, tokens[3].Text)

	assert.Equal(t, XMLTokenBodyEnd, tokens[4].Type)
	assert.Equal(t, "plain-text", tokens[4].Value)
}

func TestTokenizeConfluenceXML_NestedMacros(t *testing.T) {
	input := `<ac:structured-macro ac:name="info" ac:schema-version="1"><ac:rich-text-body><p>Before</p><ac:structured-macro ac:name="toc" ac:schema-version="1"></ac:structured-macro><p>After</p></ac:rich-text-body></ac:structured-macro>`
	tokens, err := TokenizeConfluenceXML(input)
	require.NoError(t, err)

	// Count token types
	openCount := 0
	closeCount := 0
	for _, tok := range tokens {
		if tok.Type == XMLTokenOpenTag {
			openCount++
		}
		if tok.Type == XMLTokenCloseTag {
			closeCount++
		}
	}

	assert.Equal(t, 2, openCount, "should have 2 macro opens (info and toc)")
	assert.Equal(t, 2, closeCount, "should have 2 macro closes")
}

func TestTokenizeConfluenceXML_WithSurroundingHTML(t *testing.T) {
	input := `<h1>Title</h1><ac:structured-macro ac:name="toc" ac:schema-version="1"></ac:structured-macro><p>Content</p>`
	tokens, err := TokenizeConfluenceXML(input)
	require.NoError(t, err)

	// text, open, close, text = 4 tokens
	require.Len(t, tokens, 4)

	assert.Equal(t, XMLTokenText, tokens[0].Type)
	assert.Equal(t, "<h1>Title</h1>", tokens[0].Text)

	assert.Equal(t, XMLTokenOpenTag, tokens[1].Type)
	assert.Equal(t, "toc", tokens[1].MacroName)

	assert.Equal(t, XMLTokenCloseTag, tokens[2].Type)

	assert.Equal(t, XMLTokenText, tokens[3].Type)
	assert.Equal(t, "<p>Content</p>", tokens[3].Text)
}

func TestTokenizeConfluenceXML_AllPanelTypes(t *testing.T) {
	panelTypes := []string{"info", "warning", "note", "tip", "expand"}

	for _, pt := range panelTypes {
		t.Run(pt, func(t *testing.T) {
			input := `<ac:structured-macro ac:name="` + pt + `" ac:schema-version="1"><ac:rich-text-body><p>Content</p></ac:rich-text-body></ac:structured-macro>`
			tokens, err := TokenizeConfluenceXML(input)
			require.NoError(t, err)
			require.GreaterOrEqual(t, len(tokens), 2)
			assert.Equal(t, pt, tokens[0].MacroName)
		})
	}
}

func TestTokenizeConfluenceXML_Positions(t *testing.T) {
	input := `abc<ac:structured-macro ac:name="toc" ac:schema-version="1"></ac:structured-macro>def`
	tokens, err := TokenizeConfluenceXML(input)
	require.NoError(t, err)
	require.Len(t, tokens, 4)

	assert.Equal(t, 0, tokens[0].Position)  // "abc"
	assert.Equal(t, 3, tokens[1].Position)  // macro open
	// Close and "def" positions will follow
}
```

**Step 2: Run tests**

Run: `go test -v ./pkg/md/ -run TestTokenizeConfluenceXML`
Expected: All tests pass

**Step 3: Commit**

```bash
git add pkg/md/tokenizer_xml_test.go
git commit -m "test(md): add Confluence XML tokenizer tests

Tests cover: empty input, plain HTML, simple macros, parameters,
panel macros with bodies, code with CDATA, nesting, positions."
```

---

### Task 3: Add edge case tests

**Files:**
- Modify: `pkg/md/tokenizer_xml_test.go` (append tests)

**Step 1: Add edge case tests**

Append to `pkg/md/tokenizer_xml_test.go`:

```go
func TestTokenizeConfluenceXML_CDATAWithSpecialChars(t *testing.T) {
	input := `<ac:structured-macro ac:name="code" ac:schema-version="1"><ac:plain-text-body><![CDATA[if x < 10 && y > 5 {
    fmt.Println("test")
}]]></ac:plain-text-body></ac:structured-macro>`
	tokens, err := TokenizeConfluenceXML(input)
	require.NoError(t, err)

	// Find the CDATA content token
	var cdataToken *XMLToken
	for i := range tokens {
		if tokens[i].Type == XMLTokenText && strings.Contains(tokens[i].Text, "x < 10") {
			cdataToken = &tokens[i]
			break
		}
	}

	require.NotNil(t, cdataToken, "should find CDATA content")
	assert.Contains(t, cdataToken.Text, "x < 10")
	assert.Contains(t, cdataToken.Text, "&&")
	assert.Contains(t, cdataToken.Text, "y > 5")
}

func TestTokenizeConfluenceXML_MultilineCDATA(t *testing.T) {
	input := `<ac:structured-macro ac:name="code" ac:schema-version="1"><ac:plain-text-body><![CDATA[
line1
line2
line3
]]></ac:plain-text-body></ac:structured-macro>`
	tokens, err := TokenizeConfluenceXML(input)
	require.NoError(t, err)

	// Find CDATA content
	var found bool
	for _, tok := range tokens {
		if tok.Type == XMLTokenText && strings.Contains(tok.Text, "line1") {
			found = true
			assert.Contains(t, tok.Text, "line2")
			assert.Contains(t, tok.Text, "line3")
			assert.Contains(t, tok.Text, "\n")
		}
	}
	assert.True(t, found, "should find multiline CDATA content")
}

func TestTokenizeConfluenceXML_DeeplyNestedMacros(t *testing.T) {
	input := `<ac:structured-macro ac:name="info" ac:schema-version="1"><ac:rich-text-body><ac:structured-macro ac:name="warning" ac:schema-version="1"><ac:rich-text-body><ac:structured-macro ac:name="note" ac:schema-version="1"><ac:rich-text-body><p>Deep</p></ac:rich-text-body></ac:structured-macro></ac:rich-text-body></ac:structured-macro></ac:rich-text-body></ac:structured-macro>`
	tokens, err := TokenizeConfluenceXML(input)
	require.NoError(t, err)

	// Count opens and closes
	openCount := 0
	closeCount := 0
	macroNames := []string{}
	for _, tok := range tokens {
		if tok.Type == XMLTokenOpenTag {
			openCount++
			macroNames = append(macroNames, tok.MacroName)
		}
		if tok.Type == XMLTokenCloseTag {
			closeCount++
		}
	}

	assert.Equal(t, 3, openCount, "should have 3 macro opens")
	assert.Equal(t, 3, closeCount, "should have 3 macro closes")
	assert.Contains(t, macroNames, "info")
	assert.Contains(t, macroNames, "warning")
	assert.Contains(t, macroNames, "note")
}

func TestTokenizeConfluenceXML_WhitespaceInMacro(t *testing.T) {
	input := `<ac:structured-macro ac:name="toc" ac:schema-version="1">
    <ac:parameter ac:name="maxLevel">3</ac:parameter>
</ac:structured-macro>`
	tokens, err := TokenizeConfluenceXML(input)
	require.NoError(t, err)

	// Should still find open, param, close (whitespace becomes text tokens)
	var foundParam bool
	for _, tok := range tokens {
		if tok.Type == XMLTokenParameter && tok.ParamName == "maxLevel" {
			foundParam = true
			assert.Equal(t, "3", tok.Value)
		}
	}
	assert.True(t, foundParam, "should find maxLevel parameter")
}

func TestTokenizeConfluenceXML_EmptyParameter(t *testing.T) {
	input := `<ac:structured-macro ac:name="toc" ac:schema-version="1"><ac:parameter ac:name="title"></ac:parameter></ac:structured-macro>`
	tokens, err := TokenizeConfluenceXML(input)
	require.NoError(t, err)

	var foundParam bool
	for _, tok := range tokens {
		if tok.Type == XMLTokenParameter && tok.ParamName == "title" {
			foundParam = true
			assert.Equal(t, "", tok.Value)
		}
	}
	assert.True(t, foundParam, "should find empty parameter")
}

func TestTokenizeConfluenceXML_EmptyRichTextBody(t *testing.T) {
	input := `<ac:structured-macro ac:name="info" ac:schema-version="1"><ac:rich-text-body></ac:rich-text-body></ac:structured-macro>`
	tokens, err := TokenizeConfluenceXML(input)
	require.NoError(t, err)

	// Open, body open, body close, close = 4 tokens (no text between body tags)
	bodyOpenCount := 0
	bodyCloseCount := 0
	for _, tok := range tokens {
		if tok.Type == XMLTokenBody {
			bodyOpenCount++
		}
		if tok.Type == XMLTokenBodyEnd {
			bodyCloseCount++
		}
	}
	assert.Equal(t, 1, bodyOpenCount)
	assert.Equal(t, 1, bodyCloseCount)
}

func TestTokenizeConfluenceXML_MacroNameCaseInsensitive(t *testing.T) {
	inputs := []string{
		`<ac:structured-macro ac:name="TOC" ac:schema-version="1"></ac:structured-macro>`,
		`<ac:structured-macro ac:name="Toc" ac:schema-version="1"></ac:structured-macro>`,
		`<ac:structured-macro ac:name="toc" ac:schema-version="1"></ac:structured-macro>`,
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			tokens, err := TokenizeConfluenceXML(input)
			require.NoError(t, err)
			require.GreaterOrEqual(t, len(tokens), 1)
			// All should normalize to lowercase
			assert.Equal(t, "toc", tokens[0].MacroName)
		})
	}
}

func TestExtractCDATAContent(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"<![CDATA[hello]]>", "hello"},
		{"<![CDATA[multi\nline]]>", "multi\nline"},
		{"<![CDATA[x < 10 && y > 5]]>", "x < 10 && y > 5"},
		{"<![CDATA[]]>", ""},
		{"not cdata", "not cdata"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ExtractCDATAContent(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
```

**Step 2: Run all XML tokenizer tests**

Run: `go test -v ./pkg/md/ -run "TestTokenizeConfluenceXML|TestExtractCDATA"`
Expected: All tests pass

**Step 3: Commit**

```bash
git add pkg/md/tokenizer_xml_test.go
git commit -m "test(md): add Confluence XML tokenizer edge case tests

Edge cases: special chars in CDATA, multiline CDATA, deep nesting,
whitespace, empty parameters, empty body, case insensitivity."
```

---

### Task 4: Run full test suite

**Step 1: Run all pkg/md tests**

Run: `go test -v ./pkg/md/...`
Expected: All tests pass

**Step 2: Run with race detection**

Run: `go test -race ./...`
Expected: All tests pass
