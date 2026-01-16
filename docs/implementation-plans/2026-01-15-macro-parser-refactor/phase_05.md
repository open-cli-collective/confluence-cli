# Macro Parser Refactor - Phase 5

> **For Claude:** REQUIRED SUB-SKILL: Use ed3d-plan-and-execute:subagent-driven-development to implement this plan task-by-task.

**Goal:** Replace regex-based extraction in MD→XHTML direction with parser

**Architecture:** Wire ParseBracketMacros into preprocessMacros, use RenderMacroToXML for output

**Tech Stack:** Go, goldmark, testify for assertions

**Scope:** Phase 5 of 6

**Codebase verified:** 2026-01-15

---

## Phase 5: Integration with converter.go

### Task 1: Add MacroNode to XML rendering function

**Files:**
- Create: `pkg/md/render.go`

**Step 1: Create renderer for MacroNode to Confluence XML**

```go
// render.go provides functions to render MacroNodes to Confluence storage format.
package md

import (
	"fmt"
	"sort"
	"strings"
)

// RenderMacroToXML converts a MacroNode to Confluence XML storage format.
func RenderMacroToXML(node *MacroNode) string {
	var sb strings.Builder

	// Opening tag
	sb.WriteString(`<ac:structured-macro ac:name="`)
	sb.WriteString(node.Name)
	sb.WriteString(`" ac:schema-version="1">`)

	// Parameters (sorted for consistent output)
	keys := make([]string, 0, len(node.Parameters))
	for k := range node.Parameters {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := node.Parameters[key]
		sb.WriteString(`<ac:parameter ac:name="`)
		sb.WriteString(key)
		sb.WriteString(`">`)
		sb.WriteString(escapeXML(value))
		sb.WriteString(`</ac:parameter>`)
	}

	// Body content
	macroType, _ := LookupMacro(node.Name)
	if macroType.HasBody && node.Body != "" {
		switch macroType.BodyType {
		case BodyTypeRichText:
			sb.WriteString(`<ac:rich-text-body>`)
			sb.WriteString(node.Body)
			sb.WriteString(`</ac:rich-text-body>`)
		case BodyTypePlainText:
			sb.WriteString(`<ac:plain-text-body><![CDATA[`)
			sb.WriteString(node.Body)
			sb.WriteString(`]]></ac:plain-text-body>`)
		}
	}

	// Closing tag
	sb.WriteString(`</ac:structured-macro>`)

	return sb.String()
}

// escapeXML escapes special XML characters in a string.
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

// RenderMacroToBracket converts a MacroNode back to bracket syntax.
func RenderMacroToBracket(node *MacroNode) string {
	var sb strings.Builder

	// Opening tag
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

	// Body and close tag for macros with body
	macroType, _ := LookupMacro(node.Name)
	if macroType.HasBody {
		sb.WriteString(node.Body)
		sb.WriteString("[/")
		sb.WriteString(strings.ToUpper(node.Name))
		sb.WriteString("]")
	}

	return sb.String()
}

// FormatPlaceholder creates a macro placeholder string.
func FormatPlaceholder(id int) string {
	return fmt.Sprintf("%s%d%s", macroPlaceholderPrefix, id, macroPlaceholderSuffix)
}
```

**Step 2: Verify it compiles**

Run: `go build ./pkg/md/`
Expected: No errors

**Step 3: Commit**

```bash
git add pkg/md/render.go
git commit -m "feat(md): add MacroNode rendering functions

RenderMacroToXML() converts MacroNode to Confluence storage format.
RenderMacroToBracket() converts MacroNode back to bracket syntax.
Sorted parameters for consistent, deterministic output."
```

---

### Task 2: Create render_test.go with rendering tests

**Files:**
- Create: `pkg/md/render_test.go`

**Step 1: Create rendering tests**

```go
package md

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderMacroToXML_SimpleTOC(t *testing.T) {
	node := &MacroNode{Name: "toc"}
	xml := RenderMacroToXML(node)

	assert.Contains(t, xml, `ac:name="toc"`)
	assert.Contains(t, xml, `ac:schema-version="1"`)
	assert.Contains(t, xml, `</ac:structured-macro>`)
}

func TestRenderMacroToXML_TOCWithParams(t *testing.T) {
	node := &MacroNode{
		Name:       "toc",
		Parameters: map[string]string{"maxLevel": "3", "minLevel": "1"},
	}
	xml := RenderMacroToXML(node)

	assert.Contains(t, xml, `<ac:parameter ac:name="maxLevel">3</ac:parameter>`)
	assert.Contains(t, xml, `<ac:parameter ac:name="minLevel">1</ac:parameter>`)
}

func TestRenderMacroToXML_PanelWithBody(t *testing.T) {
	node := &MacroNode{
		Name: "info",
		Body: "<p>Content</p>",
	}
	xml := RenderMacroToXML(node)

	assert.Contains(t, xml, `ac:name="info"`)
	assert.Contains(t, xml, `<ac:rich-text-body>`)
	assert.Contains(t, xml, `<p>Content</p>`)
	assert.Contains(t, xml, `</ac:rich-text-body>`)
}

func TestRenderMacroToXML_CodeWithCDATA(t *testing.T) {
	node := &MacroNode{
		Name:       "code",
		Parameters: map[string]string{"language": "go"},
		Body:       `fmt.Println("hello")`,
	}
	xml := RenderMacroToXML(node)

	assert.Contains(t, xml, `ac:name="code"`)
	assert.Contains(t, xml, `<ac:plain-text-body><![CDATA[`)
	assert.Contains(t, xml, `fmt.Println("hello")`)
	assert.Contains(t, xml, `]]></ac:plain-text-body>`)
}

func TestRenderMacroToXML_EscapesXML(t *testing.T) {
	node := &MacroNode{
		Name:       "toc",
		Parameters: map[string]string{"title": "A & B <test>"},
	}
	xml := RenderMacroToXML(node)

	assert.Contains(t, xml, `A &amp; B &lt;test&gt;`)
}

func TestRenderMacroToBracket_SimpleTOC(t *testing.T) {
	node := &MacroNode{Name: "toc"}
	bracket := RenderMacroToBracket(node)

	assert.Equal(t, "[TOC]", bracket)
}

func TestRenderMacroToBracket_TOCWithParams(t *testing.T) {
	node := &MacroNode{
		Name:       "toc",
		Parameters: map[string]string{"maxLevel": "3"},
	}
	bracket := RenderMacroToBracket(node)

	assert.Contains(t, bracket, "[TOC")
	assert.Contains(t, bracket, "maxLevel=3")
	assert.Contains(t, bracket, "]")
}

func TestRenderMacroToBracket_PanelWithBody(t *testing.T) {
	node := &MacroNode{
		Name:       "info",
		Parameters: map[string]string{"title": "Important"},
		Body:       "Content here",
	}
	bracket := RenderMacroToBracket(node)

	assert.Contains(t, bracket, "[INFO")
	assert.Contains(t, bracket, `title="Important"`)
	assert.Contains(t, bracket, "Content here")
	assert.Contains(t, bracket, "[/INFO]")
}

func TestRenderMacroToBracket_QuotedValues(t *testing.T) {
	node := &MacroNode{
		Name:       "info",
		Parameters: map[string]string{"title": "Hello World"},
	}
	bracket := RenderMacroToBracket(node)

	assert.Contains(t, bracket, `title="Hello World"`)
}

func TestFormatPlaceholder(t *testing.T) {
	assert.Equal(t, "CFMACRO0END", FormatPlaceholder(0))
	assert.Equal(t, "CFMACRO42END", FormatPlaceholder(42))
}
```

**Step 2: Run tests**

Run: `go test -v ./pkg/md/ -run TestRender`
Expected: All tests pass

**Step 3: Commit**

```bash
git add pkg/md/render_test.go
git commit -m "test(md): add MacroNode rendering tests

Tests cover XML and bracket rendering:
- Simple macros, parameters, bodies
- Code with CDATA, XML escaping
- Placeholder formatting"
```

---

### Task 3: Refactor preprocessMacros to use parser

**Files:**
- Modify: `pkg/md/converter.go`

**Step 1: Read current converter.go**

Read the file to understand the current implementation before modification.

**Step 2: Replace preprocessMacros internals**

Update `preprocessMacros` function to use the new parser while maintaining the same signature and placeholder format. The key changes:

1. Replace regex pattern matching with `ParseBracketMacros()`
2. Iterate over result segments
3. For each macro segment, use `RenderMacroToXML()` with body conversion
4. Insert placeholders and collect macros in the map

The new implementation should:
- Call `ParseBracketMacros(input)`
- Build output by iterating segments
- For text segments: append to output
- For macro segments: convert body markdown to HTML, render to XML, insert placeholder
- Return `(processedMarkdown, macroMap)`

**Step 3: Update postprocessMacros to use FormatPlaceholder**

Simplify postprocessMacros to use the `FormatPlaceholder()` helper function.

**Step 4: Add bytes import if needed**

The body conversion needs `bytes.Buffer` for goldmark.

**Step 5: Verify all tests pass**

Run: `go test -v ./pkg/md/...`
Expected: All existing tests in converter_test.go pass unchanged

**Step 6: Commit**

```bash
git add pkg/md/converter.go
git commit -m "refactor(md): integrate parser into converter.go

Replace regex-based preprocessMacros with ParseBracketMacros.
Existing API and behavior unchanged - all tests pass."
```

---

### Task 4: Remove dead code from converter.go

**Files:**
- Modify: `pkg/md/converter.go`

**Step 1: Identify dead code**

After the refactor, these become unused:
- `convertTOCMacro()` function
- `convertPanelMacro()` function
- `panelTypes` variable
- Regex pattern variables for macro matching

**Step 2: Remove dead code**

Remove the identified unused functions and variables.

**Step 3: Verify tests still pass**

Run: `go test -v ./pkg/md/...`
Expected: All tests pass

**Step 4: Commit**

```bash
git add pkg/md/converter.go
git commit -m "refactor(md): remove dead code from converter.go

Remove old regex-based conversion functions replaced by parser:
- convertTOCMacro, convertPanelMacro
- panelTypes array, regex patterns"
```

---

### Task 5: Run full test suite and verify behavior

**Step 1: Run all pkg/md tests**

Run: `go test -v ./pkg/md/...`
Expected: All tests pass

**Step 2: Run all tests with race detection**

Run: `go test -race ./...`
Expected: All tests pass

**Step 3: Manual verification (optional)**

Test roundtrip conversion manually if desired:
```bash
echo '[TOC maxLevel=3]' | go run ./cmd/cfl page create --space TEST --title "Test" --dry-run
```

Expected: Should produce valid Confluence XML with TOC macro
