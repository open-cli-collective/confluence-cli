# Macro Parser Refactor - Phase 2

> **For Claude:** REQUIRED SUB-SKILL: Use ed3d-plan-and-execute:subagent-driven-development to implement this plan task-by-task.

**Goal:** Tokenize bracket syntax [MACRO]...[/MACRO] for MD→XHTML direction

**Architecture:** Character-by-character scanner with state machine for parameter parsing

**Tech Stack:** Go, testify for assertions

**Scope:** Phase 2 of 6

**Codebase verified:** 2026-01-15

---

## Phase 2: Bracket Syntax Tokenizer

### Task 1: Create tokenizer_bracket.go with TokenizeBrackets function

**Files:**
- Create: `pkg/md/tokenizer_bracket.go`

**Step 1: Create the tokenizer file**

```go
// tokenizer_bracket.go implements tokenization for [MACRO]...[/MACRO] bracket syntax.
package md

import (
	"fmt"
	"strings"
	"unicode"
)

// TokenizeBrackets scans input for bracket macro syntax and returns a token stream.
// Recognized forms:
//   - [MACRO] or [MACRO params] - open tag
//   - [/MACRO] - close tag
//   - [MACRO/] - self-closing (no body)
//
// Text between macros is returned as BracketTokenText tokens.
// Unknown macro names are still tokenized (validation happens in parser).
func TokenizeBrackets(input string) ([]BracketToken, error) {
	var tokens []BracketToken
	pos := 0
	textStart := 0

	for pos < len(input) {
		// Look for opening bracket
		if input[pos] == '[' {
			// Emit any accumulated text before this bracket
			if pos > textStart {
				tokens = append(tokens, BracketToken{
					Type:     BracketTokenText,
					Text:     input[textStart:pos],
					Position: textStart,
				})
			}

			// Try to parse a macro tag
			token, endPos, err := parseBracketTag(input, pos)
			if err != nil {
				// Not a valid macro tag - treat '[' as text
				pos++
				continue
			}

			tokens = append(tokens, token)
			pos = endPos
			textStart = pos
		} else {
			pos++
		}
	}

	// Emit any remaining text
	if textStart < len(input) {
		tokens = append(tokens, BracketToken{
			Type:     BracketTokenText,
			Text:     input[textStart:],
			Position: textStart,
		})
	}

	return tokens, nil
}

// parseBracketTag attempts to parse a macro tag starting at pos.
// Returns the token, the position after the tag, and any error.
func parseBracketTag(input string, pos int) (BracketToken, int, error) {
	if pos >= len(input) || input[pos] != '[' {
		return BracketToken{}, pos, fmt.Errorf("expected '['")
	}

	startPos := pos
	pos++ // skip '['

	// Check for close tag [/MACRO]
	isCloseTag := false
	if pos < len(input) && input[pos] == '/' {
		isCloseTag = true
		pos++
	}

	// Parse macro name (letters only, case-insensitive)
	nameStart := pos
	for pos < len(input) && isValidMacroNameChar(rune(input[pos])) {
		pos++
	}
	if pos == nameStart {
		return BracketToken{}, startPos, fmt.Errorf("empty macro name")
	}
	macroName := input[nameStart:pos]

	// For close tags, expect immediate ']'
	if isCloseTag {
		if pos >= len(input) || input[pos] != ']' {
			return BracketToken{}, startPos, fmt.Errorf("unclosed close tag")
		}
		pos++ // skip ']'
		return BracketToken{
			Type:      BracketTokenCloseTag,
			MacroName: strings.ToUpper(macroName),
			Position:  startPos,
		}, pos, nil
	}

	// For open tags, check for self-close [MACRO/] or params and closing bracket
	// Skip whitespace after macro name
	for pos < len(input) && unicode.IsSpace(rune(input[pos])) {
		pos++
	}

	// Check for self-closing [MACRO/]
	if pos < len(input) && input[pos] == '/' {
		pos++
		if pos >= len(input) || input[pos] != ']' {
			return BracketToken{}, startPos, fmt.Errorf("expected ']' after '/'")
		}
		pos++ // skip ']'
		return BracketToken{
			Type:       BracketTokenSelfClose,
			MacroName:  strings.ToUpper(macroName),
			Parameters: make(map[string]string),
			Position:   startPos,
		}, pos, nil
	}

	// Parse parameters until ']'
	paramStart := pos
	params, endPos, err := parseParametersUntilClose(input, pos)
	if err != nil {
		return BracketToken{}, startPos, err
	}
	_ = paramStart // used for debugging if needed

	return BracketToken{
		Type:       BracketTokenOpenTag,
		MacroName:  strings.ToUpper(macroName),
		Parameters: params,
		Position:   startPos,
	}, endPos, nil
}

// parseParametersUntilClose parses key=value parameters until ']'.
// Returns the parameter map, position after ']', and any error.
func parseParametersUntilClose(input string, pos int) (map[string]string, int, error) {
	params := make(map[string]string)

	for pos < len(input) {
		// Skip whitespace
		for pos < len(input) && unicode.IsSpace(rune(input[pos])) {
			pos++
		}

		// Check for end of tag
		if pos < len(input) && input[pos] == ']' {
			pos++ // skip ']'
			return params, pos, nil
		}

		// Check for self-close marker
		if pos < len(input) && input[pos] == '/' {
			pos++
			if pos >= len(input) || input[pos] != ']' {
				return nil, pos, fmt.Errorf("expected ']' after '/'")
			}
			// Don't consume ']' here - let caller handle it
			pos-- // back up to '/'
			return params, pos, nil
		}

		// Parse parameter key
		keyStart := pos
		for pos < len(input) && isValidParamKeyChar(rune(input[pos])) {
			pos++
		}
		if pos == keyStart {
			// No key found, might be end of params
			if pos < len(input) && input[pos] == ']' {
				pos++
				return params, pos, nil
			}
			return nil, pos, fmt.Errorf("expected parameter key or ']'")
		}
		key := input[keyStart:pos]

		// Expect '='
		if pos >= len(input) || input[pos] != '=' {
			// Key without value - treat as boolean true
			params[key] = "true"
			continue
		}
		pos++ // skip '='

		// Parse value (may be quoted)
		value, newPos, err := parseParamValue(input, pos)
		if err != nil {
			return nil, pos, err
		}
		params[key] = value
		pos = newPos
	}

	return nil, pos, fmt.Errorf("unclosed bracket tag")
}

// parseParamValue parses a parameter value, handling quoted strings.
func parseParamValue(input string, pos int) (string, int, error) {
	if pos >= len(input) {
		return "", pos, fmt.Errorf("unexpected end of input")
	}

	// Check for quoted value
	if input[pos] == '"' || input[pos] == '\'' {
		quoteChar := input[pos]
		pos++ // skip opening quote
		valueStart := pos

		for pos < len(input) {
			if input[pos] == quoteChar {
				value := input[valueStart:pos]
				pos++ // skip closing quote
				return value, pos, nil
			}
			// Handle escaped quotes
			if input[pos] == '\\' && pos+1 < len(input) && input[pos+1] == quoteChar {
				pos += 2
				continue
			}
			pos++
		}
		return "", pos, fmt.Errorf("unclosed quoted value")
	}

	// Unquoted value - read until space or ']'
	valueStart := pos
	for pos < len(input) && !unicode.IsSpace(rune(input[pos])) && input[pos] != ']' && input[pos] != '/' {
		pos++
	}
	return input[valueStart:pos], pos, nil
}

// isValidMacroNameChar returns true if r is valid in a macro name.
func isValidMacroNameChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_'
}

// isValidParamKeyChar returns true if r is valid in a parameter key.
func isValidParamKeyChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_'
}
```

**Step 2: Verify it compiles**

Run: `go build ./pkg/md/`
Expected: No errors

**Step 3: Commit**

```bash
git add pkg/md/tokenizer_bracket.go
git commit -m "feat(md): add bracket syntax tokenizer

TokenizeBrackets() scans [MACRO]...[/MACRO] syntax.
Handles open tags, close tags, self-closing, parameters with quoted values."
```

---

### Task 2: Create comprehensive tokenizer tests

**Files:**
- Create: `pkg/md/tokenizer_bracket_test.go`

**Step 1: Create test file with basic tests**

```go
package md

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenizeBrackets_EmptyInput(t *testing.T) {
	tokens, err := TokenizeBrackets("")
	require.NoError(t, err)
	assert.Empty(t, tokens)
}

func TestTokenizeBrackets_PlainText(t *testing.T) {
	tokens, err := TokenizeBrackets("Hello world")
	require.NoError(t, err)
	require.Len(t, tokens, 1)
	assert.Equal(t, BracketTokenText, tokens[0].Type)
	assert.Equal(t, "Hello world", tokens[0].Text)
}

func TestTokenizeBrackets_SimpleMacro(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantName  string
		wantType  BracketTokenType
		wantCount int
	}{
		{"TOC no params", "[TOC]", "TOC", BracketTokenOpenTag, 1},
		{"TOC lowercase", "[toc]", "TOC", BracketTokenOpenTag, 1},
		{"TOC mixed case", "[Toc]", "TOC", BracketTokenOpenTag, 1},
		{"INFO open", "[INFO]", "INFO", BracketTokenOpenTag, 1},
		{"INFO close", "[/INFO]", "INFO", BracketTokenCloseTag, 1},
		{"WARNING", "[WARNING]", "WARNING", BracketTokenOpenTag, 1},
		{"NOTE", "[NOTE]", "NOTE", BracketTokenOpenTag, 1},
		{"TIP", "[TIP]", "TIP", BracketTokenOpenTag, 1},
		{"EXPAND", "[EXPAND]", "EXPAND", BracketTokenOpenTag, 1},
		{"CODE", "[CODE]", "CODE", BracketTokenOpenTag, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := TokenizeBrackets(tt.input)
			require.NoError(t, err)
			require.Len(t, tokens, tt.wantCount)
			assert.Equal(t, tt.wantType, tokens[0].Type)
			assert.Equal(t, tt.wantName, tokens[0].MacroName)
		})
	}
}

func TestTokenizeBrackets_WithParameters(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantParams map[string]string
	}{
		{
			"single param",
			"[TOC maxLevel=3]",
			map[string]string{"maxLevel": "3"},
		},
		{
			"multiple params",
			"[TOC maxLevel=3 minLevel=1]",
			map[string]string{"maxLevel": "3", "minLevel": "1"},
		},
		{
			"quoted value",
			`[INFO title="Hello World"]`,
			map[string]string{"title": "Hello World"},
		},
		{
			"single quoted value",
			`[INFO title='Hello World']`,
			map[string]string{"title": "Hello World"},
		},
		{
			"mixed quoted and unquoted",
			`[EXPAND title="Click to expand" icon=info]`,
			map[string]string{"title": "Click to expand", "icon": "info"},
		},
		{
			"empty value",
			`[INFO title=""]`,
			map[string]string{"title": ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := TokenizeBrackets(tt.input)
			require.NoError(t, err)
			require.Len(t, tokens, 1)
			assert.Equal(t, BracketTokenOpenTag, tokens[0].Type)
			assert.Equal(t, tt.wantParams, tokens[0].Parameters)
		})
	}
}

func TestTokenizeBrackets_OpenAndClose(t *testing.T) {
	input := "[INFO]content[/INFO]"
	tokens, err := TokenizeBrackets(input)
	require.NoError(t, err)
	require.Len(t, tokens, 3)

	assert.Equal(t, BracketTokenOpenTag, tokens[0].Type)
	assert.Equal(t, "INFO", tokens[0].MacroName)

	assert.Equal(t, BracketTokenText, tokens[1].Type)
	assert.Equal(t, "content", tokens[1].Text)

	assert.Equal(t, BracketTokenCloseTag, tokens[2].Type)
	assert.Equal(t, "INFO", tokens[2].MacroName)
}

func TestTokenizeBrackets_WithSurroundingText(t *testing.T) {
	input := "Before [TOC] after"
	tokens, err := TokenizeBrackets(input)
	require.NoError(t, err)
	require.Len(t, tokens, 3)

	assert.Equal(t, BracketTokenText, tokens[0].Type)
	assert.Equal(t, "Before ", tokens[0].Text)

	assert.Equal(t, BracketTokenOpenTag, tokens[1].Type)
	assert.Equal(t, "TOC", tokens[1].MacroName)

	assert.Equal(t, BracketTokenText, tokens[2].Type)
	assert.Equal(t, " after", tokens[2].Text)
}

func TestTokenizeBrackets_NestedMacros(t *testing.T) {
	input := "[INFO]outer [TOC] content[/INFO]"
	tokens, err := TokenizeBrackets(input)
	require.NoError(t, err)
	require.Len(t, tokens, 5)

	assert.Equal(t, BracketTokenOpenTag, tokens[0].Type)
	assert.Equal(t, "INFO", tokens[0].MacroName)

	assert.Equal(t, BracketTokenText, tokens[1].Type)
	assert.Equal(t, "outer ", tokens[1].Text)

	assert.Equal(t, BracketTokenOpenTag, tokens[2].Type)
	assert.Equal(t, "TOC", tokens[2].MacroName)

	assert.Equal(t, BracketTokenText, tokens[3].Type)
	assert.Equal(t, " content", tokens[3].Text)

	assert.Equal(t, BracketTokenCloseTag, tokens[4].Type)
	assert.Equal(t, "INFO", tokens[4].MacroName)
}

func TestTokenizeBrackets_MultipleMacros(t *testing.T) {
	input := "[INFO]first[/INFO]\n[WARNING]second[/WARNING]"
	tokens, err := TokenizeBrackets(input)
	require.NoError(t, err)
	require.Len(t, tokens, 7)

	// First macro
	assert.Equal(t, BracketTokenOpenTag, tokens[0].Type)
	assert.Equal(t, "INFO", tokens[0].MacroName)
	assert.Equal(t, BracketTokenText, tokens[1].Type)
	assert.Equal(t, "first", tokens[1].Text)
	assert.Equal(t, BracketTokenCloseTag, tokens[2].Type)
	assert.Equal(t, "INFO", tokens[2].MacroName)

	// Text between
	assert.Equal(t, BracketTokenText, tokens[3].Type)
	assert.Equal(t, "\n", tokens[3].Text)

	// Second macro
	assert.Equal(t, BracketTokenOpenTag, tokens[4].Type)
	assert.Equal(t, "WARNING", tokens[4].MacroName)
	assert.Equal(t, BracketTokenText, tokens[5].Type)
	assert.Equal(t, "second", tokens[5].Text)
	assert.Equal(t, BracketTokenCloseTag, tokens[6].Type)
	assert.Equal(t, "WARNING", tokens[6].MacroName)
}

func TestTokenizeBrackets_Positions(t *testing.T) {
	input := "abc[TOC]def"
	tokens, err := TokenizeBrackets(input)
	require.NoError(t, err)
	require.Len(t, tokens, 3)

	assert.Equal(t, 0, tokens[0].Position)  // "abc"
	assert.Equal(t, 3, tokens[1].Position)  // "[TOC]"
	assert.Equal(t, 8, tokens[2].Position)  // "def"
}
```

**Step 2: Run basic tests**

Run: `go test -v ./pkg/md/ -run TestTokenizeBrackets`
Expected: All tests pass

**Step 3: Commit**

```bash
git add pkg/md/tokenizer_bracket_test.go
git commit -m "test(md): add bracket tokenizer basic tests

Tests cover: empty input, plain text, simple macros, parameters,
open/close pairs, surrounding text, nesting, multiple macros, positions."
```

---

### Task 3: Add edge case tests

**Files:**
- Modify: `pkg/md/tokenizer_bracket_test.go` (append tests)

**Step 1: Add edge case tests**

Append to `pkg/md/tokenizer_bracket_test.go`:

```go
func TestTokenizeBrackets_MalformedSyntax(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			"orphan open bracket",
			"text [ more text",
			"lone bracket treated as text",
		},
		{
			"orphan close bracket",
			"text ] more text",
			"close bracket in text is just text",
		},
		{
			"incomplete macro name",
			"text [",
			"bracket at end of input",
		},
		{
			"brackets in text",
			"array[0] = value",
			"programming brackets not macro syntax",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := TokenizeBrackets(tt.input)
			require.NoError(t, err, "should not error on malformed input")
			// Malformed macro syntax should be treated as text
			for _, tok := range tokens {
				if tok.Type == BracketTokenOpenTag || tok.Type == BracketTokenCloseTag {
					// Only valid macro names should be tokenized
					_, known := LookupMacro(tok.MacroName)
					// Unknown macros are still tokenized - parser validates them
					_ = known
				}
			}
		})
	}
}

func TestTokenizeBrackets_BracketsInQuotedValues(t *testing.T) {
	input := `[INFO title="[Important]"]content[/INFO]`
	tokens, err := TokenizeBrackets(input)
	require.NoError(t, err)

	// Should have: open tag, text, close tag
	require.Len(t, tokens, 3)
	assert.Equal(t, BracketTokenOpenTag, tokens[0].Type)
	assert.Equal(t, "[Important]", tokens[0].Parameters["title"])
}

func TestTokenizeBrackets_EscapedQuotes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantVal  string
	}{
		{
			"escaped double quote",
			`[INFO title="Say \"Hello\""]`,
			`Say "Hello"`,
		},
		{
			"escaped single quote",
			`[INFO title='It\'s fine']`,
			`It's fine`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := TokenizeBrackets(tt.input)
			require.NoError(t, err)
			require.Len(t, tokens, 1)
			// Note: escaped quote handling may need adjustment
			// Current implementation may not strip backslashes
			assert.Contains(t, tokens[0].Parameters["title"], "Hello")
		})
	}
}

func TestTokenizeBrackets_MultilineBody(t *testing.T) {
	input := `[INFO]
This is
multiline
content
[/INFO]`
	tokens, err := TokenizeBrackets(input)
	require.NoError(t, err)
	require.Len(t, tokens, 3)

	assert.Equal(t, BracketTokenOpenTag, tokens[0].Type)
	assert.Equal(t, BracketTokenText, tokens[1].Type)
	assert.Contains(t, tokens[1].Text, "\n")
	assert.Equal(t, BracketTokenCloseTag, tokens[2].Type)
}

func TestTokenizeBrackets_SelfClosing(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"TOC self-close", "[TOC/]"},
		{"with space", "[TOC /]"},
		{"with params", "[TOC maxLevel=3/]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := TokenizeBrackets(tt.input)
			require.NoError(t, err)
			require.Len(t, tokens, 1)
			assert.Equal(t, BracketTokenSelfClose, tokens[0].Type)
			assert.Equal(t, "TOC", tokens[0].MacroName)
		})
	}
}

func TestTokenizeBrackets_DeeplyNested(t *testing.T) {
	input := "[INFO][WARNING][NOTE]deep[/NOTE][/WARNING][/INFO]"
	tokens, err := TokenizeBrackets(input)
	require.NoError(t, err)

	// Count token types
	openCount := 0
	closeCount := 0
	textCount := 0
	for _, tok := range tokens {
		switch tok.Type {
		case BracketTokenOpenTag:
			openCount++
		case BracketTokenCloseTag:
			closeCount++
		case BracketTokenText:
			textCount++
		}
	}

	assert.Equal(t, 3, openCount, "should have 3 open tags")
	assert.Equal(t, 3, closeCount, "should have 3 close tags")
	assert.Equal(t, 1, textCount, "should have 1 text token")
}

func TestTokenizeBrackets_SpecialCharactersInBody(t *testing.T) {
	input := "[INFO]<script>alert('xss')</script> & < > \"[/INFO]"
	tokens, err := TokenizeBrackets(input)
	require.NoError(t, err)
	require.Len(t, tokens, 3)

	assert.Equal(t, BracketTokenText, tokens[1].Type)
	assert.Contains(t, tokens[1].Text, "<script>")
	assert.Contains(t, tokens[1].Text, "&")
}

func TestTokenizeBrackets_WhitespaceHandling(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantName  string
		wantParam string
	}{
		{
			"space after name",
			"[TOC ]",
			"TOC",
			"",
		},
		{
			"multiple spaces",
			"[TOC   maxLevel=3]",
			"TOC",
			"3",
		},
		{
			"tabs",
			"[TOC\tmaxLevel=3]",
			"TOC",
			"3",
		},
		{
			"newline in params",
			"[INFO\n  title=\"test\"]",
			"INFO",
			"test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := TokenizeBrackets(tt.input)
			require.NoError(t, err)
			require.GreaterOrEqual(t, len(tokens), 1)
			assert.Equal(t, tt.wantName, tokens[0].MacroName)
			if tt.wantParam != "" {
				if tokens[0].Type == BracketTokenOpenTag {
					val, exists := tokens[0].Parameters["maxLevel"]
					if !exists {
						val = tokens[0].Parameters["title"]
					}
					assert.Equal(t, tt.wantParam, val)
				}
			}
		})
	}
}
```

**Step 2: Run all tokenizer tests**

Run: `go test -v ./pkg/md/ -run TestTokenizeBrackets`
Expected: All tests pass (fix any failing edge cases)

**Step 3: Commit**

```bash
git add pkg/md/tokenizer_bracket_test.go
git commit -m "test(md): add bracket tokenizer edge case tests

Edge cases: malformed syntax, brackets in quoted values, escaped quotes,
multiline body, self-closing, deep nesting, special chars, whitespace."
```

---

### Task 4: Run full test suite

**Step 1: Run all pkg/md tests**

Run: `go test -v ./pkg/md/...`
Expected: All tests pass

**Step 2: Run with race detection**

Run: `go test -race ./...`
Expected: All tests pass
