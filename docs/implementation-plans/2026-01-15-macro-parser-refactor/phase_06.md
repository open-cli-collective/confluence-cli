# Macro Parser Refactor - Phase 6

> **For Claude:** REQUIRED SUB-SKILL: Use ed3d-plan-and-execute:subagent-driven-development to implement this plan task-by-task.

**Goal:** Replace regex-based extraction in XHTML→MD direction with parser

**Architecture:** Wire ParseConfluenceXML into processConfluenceMacrosWithPlaceholders, use RenderMacroToBracket for output

**Tech Stack:** Go, html-to-markdown, testify for assertions

**Scope:** Phase 6 of 6

**Codebase verified:** 2026-01-15

---

## Phase 6: Integration with from_html.go

### Task 1: Refactor processConfluenceMacrosWithPlaceholders to use parser

**Files:**
- Modify: `pkg/md/from_html.go`

**Step 1: Read current from_html.go**

Read the file to understand the current implementation before modification.

**Step 2: Update processConfluenceMacrosWithPlaceholders**

Replace the internal implementation to use `ParseConfluenceXML()` while maintaining the same signature and behavior:

```go
// processConfluenceMacrosWithPlaceholders processes Confluence macros in HTML.
// When showMacros is true, macros are converted to bracket syntax.
// When showMacros is false, macros are stripped from output.
func processConfluenceMacrosWithPlaceholders(html string, showMacros bool) (string, map[int]macroPlaceholder) {
	// Convert code blocks first (special handling)
	html = convertCodeBlockMacros(html)

	macroMap := make(map[int]macroPlaceholder)

	if !showMacros {
		// Strip all macros without placeholders
		return stripConfluenceMacros(html), macroMap
	}

	// Parse the XML to extract macros
	result, err := ParseConfluenceXML(html)
	if err != nil {
		return html, macroMap
	}

	// Build output with placeholders
	var output strings.Builder
	counter := 0

	for _, seg := range result.Segments {
		switch seg.Type {
		case SegmentText:
			output.WriteString(seg.Text)
		case SegmentMacro:
			// Convert macro to bracket syntax placeholders
			placeholder := renderMacroToPlaceholders(seg.Macro, counter)
			macroMap[counter] = placeholder
			output.WriteString(placeholderOpenPrefix + strconv.Itoa(counter))
			if placeholder.closeTag != "" {
				// Body content goes between placeholders
				output.WriteString(seg.Macro.Body)
				output.WriteString(placeholderClosePrefix + strconv.Itoa(counter))
			}
			counter++
		}
	}

	return output.String(), macroMap
}

// renderMacroToPlaceholders creates a macroPlaceholder from a MacroNode.
func renderMacroToPlaceholders(node *MacroNode, id int) macroPlaceholder {
	macroType, _ := LookupMacro(node.Name)

	openTag := RenderMacroToBracketOpen(node)

	var closeTag string
	if macroType.HasBody {
		closeTag = "[/" + strings.ToUpper(node.Name) + "]"
	}

	return macroPlaceholder{
		openTag:  openTag,
		closeTag: closeTag,
	}
}

// stripConfluenceMacros removes all Confluence structured macros from HTML.
func stripConfluenceMacros(html string) string {
	result, err := ParseConfluenceXML(html)
	if err != nil {
		return html
	}

	var output strings.Builder
	for _, seg := range result.Segments {
		if seg.Type == SegmentText {
			output.WriteString(seg.Text)
		}
		// Macros are silently dropped
	}
	return output.String()
}
```

**Step 3: Add strconv import if needed**

The placeholder ID formatting uses `strconv.Itoa()`.

**Step 4: Verify all tests pass**

Run: `go test -v ./pkg/md/...`
Expected: All existing tests in from_html_test.go pass unchanged

**Step 5: Commit**

```bash
git add pkg/md/from_html.go
git commit -m "refactor(md): integrate parser into from_html.go

Replace regex-based processConfluenceMacrosWithPlaceholders with ParseConfluenceXML.
Existing API and ShowMacros behavior unchanged - all tests pass."
```

---

### Task 2: Add RenderMacroToBracketOpen to render.go

**Files:**
- Modify: `pkg/md/render.go`

**Step 1: Add the function**

Add `RenderMacroToBracketOpen()` to render.go:

```go
// RenderMacroToBracketOpen renders just the opening bracket tag (without body or close).
func RenderMacroToBracketOpen(node *MacroNode) string {
	var sb strings.Builder
	sb.WriteString("[")
	sb.WriteString(strings.ToUpper(node.Name))

	// Parameters (sorted for consistent output)
	keys := make([]string, 0, len(node.Parameters))
	for k := range node.Parameters {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := node.Parameters[key]
		sb.WriteString(" ")
		sb.WriteString(key)
		sb.WriteString("=")
		if strings.ContainsAny(value, " \t\n\"") {
			sb.WriteString(`"`)
			sb.WriteString(strings.ReplaceAll(value, `"`, `\"`))
			sb.WriteString(`"`)
		} else {
			sb.WriteString(value)
		}
	}
	sb.WriteString("]")
	return sb.String()
}
```

**Step 2: Add test for the function**

Append to `pkg/md/render_test.go`:

```go
func TestRenderMacroToBracketOpen_SimpleTOC(t *testing.T) {
	node := &MacroNode{Name: "toc"}
	bracket := RenderMacroToBracketOpen(node)
	assert.Equal(t, "[TOC]", bracket)
}

func TestRenderMacroToBracketOpen_WithParams(t *testing.T) {
	node := &MacroNode{
		Name:       "info",
		Parameters: map[string]string{"title": "Hello World"},
	}
	bracket := RenderMacroToBracketOpen(node)
	assert.Contains(t, bracket, "[INFO")
	assert.Contains(t, bracket, `title="Hello World"`)
	assert.True(t, strings.HasSuffix(bracket, "]"))
}
```

**Step 3: Verify it compiles and tests pass**

Run: `go build ./pkg/md/ && go test -v ./pkg/md/ -run TestRenderMacroToBracketOpen`
Expected: No errors, tests pass

**Step 4: Commit**

```bash
git add pkg/md/render.go pkg/md/render_test.go
git commit -m "feat(md): add RenderMacroToBracketOpen function

Renders only the opening bracket tag for use with placeholders."
```

---

### Task 3: Remove dead code from from_html.go

**Files:**
- Modify: `pkg/md/from_html.go`

**Step 1: Identify dead code**

After the refactor, these become unused:
- `findMatchingCloseTag()` function
- `findInnermostMacro()` function
- Old regex patterns for macro matching
- `convertMacroToPlaceholders()` old implementation (if exists)
- `extractMacroParams()` (if replaced)

**Step 2: Remove dead code**

Remove the identified unused functions. Keep `convertCodeBlockMacros()` as it handles code blocks specially.

**Step 3: Verify tests still pass**

Run: `go test -v ./pkg/md/...`
Expected: All tests pass

**Step 4: Commit**

```bash
git add pkg/md/from_html.go
git commit -m "refactor(md): remove dead code from from_html.go

Remove old functions replaced by parser:
- findMatchingCloseTag, findInnermostMacro
- Old regex patterns and helper functions"
```

---

### Task 4: Run full test suite and verify behavior

**Step 1: Run all pkg/md tests**

Run: `go test -v ./pkg/md/...`
Expected: All tests pass

**Step 2: Run all tests with race detection**

Run: `go test -race ./...`
Expected: All tests pass

**Step 3: Verify ShowMacros behavior**

Test both modes:
```bash
# ShowMacros=true should preserve macros as bracket syntax
# ShowMacros=false should strip macros
go test -v ./pkg/md/ -run "TestFromConfluence.*ShowMacros"
```

Expected: Tests verify both behaviors work correctly

---

### Task 5: Add roundtrip tests

**Files:**
- Create: `pkg/md/roundtrip_test.go`

**Step 1: Create roundtrip test file**

```go
package md

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRoundtrip verifies that macros survive MD→XHTML→MD conversion.
func TestRoundtrip_TOC(t *testing.T) {
	input := "[TOC maxLevel=3]"

	// MD → XHTML
	xhtml, err := ToConfluenceStorage([]byte(input))
	require.NoError(t, err)
	assert.Contains(t, xhtml, `ac:name="toc"`)
	assert.Contains(t, xhtml, `maxLevel`)

	// XHTML → MD
	md, err := FromConfluenceStorageWithOptions(xhtml, ConvertOptions{ShowMacros: true})
	require.NoError(t, err)
	assert.Contains(t, strings.ToUpper(md), "[TOC")
	assert.Contains(t, md, "maxLevel")
}

func TestRoundtrip_InfoPanel(t *testing.T) {
	input := `[INFO title="Important"]
This is important content.
[/INFO]`

	// MD → XHTML
	xhtml, err := ToConfluenceStorage([]byte(input))
	require.NoError(t, err)
	assert.Contains(t, xhtml, `ac:name="info"`)
	assert.Contains(t, xhtml, `<ac:rich-text-body>`)

	// XHTML → MD
	md, err := FromConfluenceStorageWithOptions(xhtml, ConvertOptions{ShowMacros: true})
	require.NoError(t, err)
	assert.Contains(t, strings.ToUpper(md), "[INFO")
	assert.Contains(t, md, "[/INFO]")
	assert.Contains(t, md, "important content")
}

func TestRoundtrip_NestedMacros(t *testing.T) {
	input := `[INFO]
Content with [TOC] inside.
[/INFO]`

	// MD → XHTML
	xhtml, err := ToConfluenceStorage([]byte(input))
	require.NoError(t, err)
	assert.Contains(t, xhtml, `ac:name="info"`)
	assert.Contains(t, xhtml, `ac:name="toc"`)

	// XHTML → MD
	md, err := FromConfluenceStorageWithOptions(xhtml, ConvertOptions{ShowMacros: true})
	require.NoError(t, err)
	assert.Contains(t, strings.ToUpper(md), "[INFO")
	assert.Contains(t, strings.ToUpper(md), "[TOC")
}

func TestRoundtrip_AllPanelTypes(t *testing.T) {
	panelTypes := []string{"INFO", "WARNING", "NOTE", "TIP", "EXPAND"}

	for _, pt := range panelTypes {
		t.Run(pt, func(t *testing.T) {
			input := "[" + pt + "]Content[/" + pt + "]"

			xhtml, err := ToConfluenceStorage([]byte(input))
			require.NoError(t, err)
			assert.Contains(t, xhtml, `ac:name="`+strings.ToLower(pt)+`"`)

			md, err := FromConfluenceStorageWithOptions(xhtml, ConvertOptions{ShowMacros: true})
			require.NoError(t, err)
			assert.Contains(t, strings.ToUpper(md), "["+pt)
			assert.Contains(t, strings.ToUpper(md), "[/"+pt+"]")
		})
	}
}
```

**Step 2: Run roundtrip tests**

Run: `go test -v ./pkg/md/ -run TestRoundtrip`
Expected: All tests pass

**Step 3: Commit**

```bash
git add pkg/md/roundtrip_test.go
git commit -m "test(md): add macro roundtrip tests

Verify MD→XHTML→MD preserves:
- TOC with parameters
- Panel macros with body
- Nested macros
- All panel types (info, warning, note, tip, expand)"
```

---

### Task 6: Final verification

**Step 1: Run complete test suite**

Run: `go test -v ./...`
Expected: All tests pass

**Step 2: Run with race detection**

Run: `go test -race ./...`
Expected: All tests pass

**Step 3: Run linter**

Run: `make lint`
Expected: No lint errors

**Step 4: Commit any final fixes**

If any issues found, fix and commit.

**Step 5: Final commit for phase completion**

```bash
git add -A
git commit -m "feat(md): complete macro parser refactor

All 6 phases complete:
- Phase 1: Core data structures (MacroNode, MacroType, MacroRegistry)
- Phase 2: Bracket syntax tokenizer
- Phase 3: Confluence XML tokenizer
- Phase 4: Parser implementation (both directions)
- Phase 5: Integration with converter.go
- Phase 6: Integration with from_html.go

Adding new macros now requires one line in MacroRegistry."
```
