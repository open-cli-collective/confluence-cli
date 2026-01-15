# Macro Parser Refactor Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use ed3d-plan-and-execute:subagent-driven-development to implement this plan task-by-task.

**Goal:** Define core data structures for unified macro representation

**Architecture:** MacroNode represents parsed macros in both directions, MacroType defines behavior per macro, MacroRegistry provides single extensibility point

**Tech Stack:** Go, testify for assertions

**Scope:** 6 phases from original design (this is phase 1 of 6)

**Codebase verified:** 2026-01-15

---

## Phase 1: Core Data Structures

### Task 1: Create MacroNode and MacroType structs

**Files:**
- Create: `pkg/md/macro.go`

**Step 1: Create the file with types**

```go
// macro.go defines the core data structures for macro parsing.
package md

// MacroNode represents a parsed macro in either direction (MD↔XHTML).
type MacroNode struct {
	Name       string            // "toc", "info", "warning", etc.
	Parameters map[string]string // key-value pairs from macro attributes
	Body       string            // raw content for body macros
	Children   []*MacroNode      // nested macros within body
}

// BodyType indicates how a macro's body content should be handled.
type BodyType string

const (
	BodyTypeNone      BodyType = ""          // no body (e.g., TOC)
	BodyTypeRichText  BodyType = "rich-text" // HTML content (e.g., panels)
	BodyTypePlainText BodyType = "plain-text" // CDATA content (e.g., code)
)

// MacroType defines the behavior for a specific macro.
type MacroType struct {
	Name     string   // canonical lowercase name
	HasBody  bool     // true for panels/expand/code, false for TOC
	BodyType BodyType // how to handle body content
}
```

**Step 2: Verify it compiles**

Run: `go build ./pkg/md/`
Expected: No errors

**Step 3: Commit**

```bash
git add pkg/md/macro.go
git commit -m "feat(md): add MacroNode and MacroType structs

Foundation for parser refactor - unified macro representation."
```

---

### Task 2: Create MacroRegistry with current macros

**Files:**
- Modify: `pkg/md/macro.go` (append to existing file)

**Step 1: Add registry to macro.go**

Append to `pkg/md/macro.go`:

```go
// MacroRegistry maps macro names to their type definitions.
// Adding a new macro = adding one entry here.
var MacroRegistry = map[string]MacroType{
	"toc": {
		Name:    "toc",
		HasBody: false,
	},
	"info": {
		Name:     "info",
		HasBody:  true,
		BodyType: BodyTypeRichText,
	},
	"warning": {
		Name:     "warning",
		HasBody:  true,
		BodyType: BodyTypeRichText,
	},
	"note": {
		Name:     "note",
		HasBody:  true,
		BodyType: BodyTypeRichText,
	},
	"tip": {
		Name:     "tip",
		HasBody:  true,
		BodyType: BodyTypeRichText,
	},
	"expand": {
		Name:     "expand",
		HasBody:  true,
		BodyType: BodyTypeRichText,
	},
	"code": {
		Name:     "code",
		HasBody:  true,
		BodyType: BodyTypePlainText,
	},
}

// LookupMacro returns the MacroType for a given name, normalizing to lowercase.
// Returns ok=false if macro is not registered.
func LookupMacro(name string) (MacroType, bool) {
	mt, ok := MacroRegistry[strings.ToLower(name)]
	return mt, ok
}
```

**Step 2: Add strings import at top of file**

Update imports at top of `pkg/md/macro.go`:

```go
package md

import "strings"
```

**Step 3: Verify it compiles**

Run: `go build ./pkg/md/`
Expected: No errors

**Step 4: Commit**

```bash
git add pkg/md/macro.go
git commit -m "feat(md): add MacroRegistry with current macros

Registry contains: toc, info, warning, note, tip, expand, code.
LookupMacro provides case-insensitive access."
```

---

### Task 3: Create token type definitions

**Files:**
- Create: `pkg/md/tokens.go`

**Step 1: Create tokens.go with all token types**

```go
// tokens.go defines token types for macro parsing in both directions.
package md

// BracketTokenType represents token types for bracket syntax [MACRO]...[/MACRO]
type BracketTokenType int

const (
	BracketTokenText       BracketTokenType = iota // plain text between macros
	BracketTokenOpenTag                            // [MACRO] or [MACRO params]
	BracketTokenCloseTag                           // [/MACRO]
	BracketTokenSelfClose                          // [MACRO/] (no body)
)

// BracketToken represents a single token from bracket syntax parsing.
type BracketToken struct {
	Type       BracketTokenType
	MacroName  string            // set for OpenTag, CloseTag, SelfClose
	Parameters map[string]string // set for OpenTag, SelfClose
	Text       string            // set for Text tokens
	Position   int               // byte offset in original input
}

// XMLTokenType represents token types for Confluence XML parsing.
type XMLTokenType int

const (
	XMLTokenText      XMLTokenType = iota // text/HTML between macros
	XMLTokenOpenTag                       // <ac:structured-macro ac:name="...">
	XMLTokenCloseTag                      // </ac:structured-macro>
	XMLTokenParameter                     // <ac:parameter ac:name="...">value</ac:parameter>
	XMLTokenBody                          // <ac:rich-text-body> or <ac:plain-text-body>
	XMLTokenBodyEnd                       // </ac:rich-text-body> or </ac:plain-text-body>
)

// XMLToken represents a single token from Confluence XML parsing.
type XMLToken struct {
	Type      XMLTokenType
	MacroName string   // set for OpenTag
	ParamName string   // set for Parameter
	Value     string   // parameter value or body type ("rich-text" or "plain-text")
	Text      string   // set for Text tokens
	Position  int      // byte offset in original input
}
```

**Step 2: Verify it compiles**

Run: `go build ./pkg/md/`
Expected: No errors

**Step 3: Commit**

```bash
git add pkg/md/tokens.go
git commit -m "feat(md): add token type definitions for both directions

BracketToken for MD→XHTML ([MACRO] syntax)
XMLToken for XHTML→MD (<ac:structured-macro> syntax)"
```

---

### Task 4: Create macro_test.go with registry tests

**Files:**
- Create: `pkg/md/macro_test.go`

**Step 1: Create test file**

```go
package md

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMacroRegistry_ContainsExpectedMacros(t *testing.T) {
	expectedMacros := []string{"toc", "info", "warning", "note", "tip", "expand", "code"}

	for _, name := range expectedMacros {
		t.Run(name, func(t *testing.T) {
			mt, ok := MacroRegistry[name]
			assert.True(t, ok, "MacroRegistry should contain %q", name)
			assert.Equal(t, name, mt.Name)
		})
	}
}

func TestLookupMacro_CaseInsensitive(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		found    bool
	}{
		{"toc", "toc", true},
		{"TOC", "toc", true},
		{"Toc", "toc", true},
		{"INFO", "info", true},
		{"Info", "info", true},
		{"unknown", "", false},
		{"UNKNOWN", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			mt, ok := LookupMacro(tt.input)
			assert.Equal(t, tt.found, ok)
			if tt.found {
				assert.Equal(t, tt.expected, mt.Name)
			}
		})
	}
}

func TestMacroType_BodyConfiguration(t *testing.T) {
	tests := []struct {
		name     string
		hasBody  bool
		bodyType BodyType
	}{
		{"toc", false, BodyTypeNone},
		{"info", true, BodyTypeRichText},
		{"warning", true, BodyTypeRichText},
		{"note", true, BodyTypeRichText},
		{"tip", true, BodyTypeRichText},
		{"expand", true, BodyTypeRichText},
		{"code", true, BodyTypePlainText},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mt, ok := MacroRegistry[tt.name]
			assert.True(t, ok)
			assert.Equal(t, tt.hasBody, mt.HasBody)
			assert.Equal(t, tt.bodyType, mt.BodyType)
		})
	}
}

func TestMacroNode_Construction(t *testing.T) {
	// Test basic construction
	node := &MacroNode{
		Name:       "info",
		Parameters: map[string]string{"title": "Important"},
		Body:       "This is the content",
		Children:   nil,
	}

	assert.Equal(t, "info", node.Name)
	assert.Equal(t, "Important", node.Parameters["title"])
	assert.Equal(t, "This is the content", node.Body)
	assert.Nil(t, node.Children)
}

func TestMacroNode_WithChildren(t *testing.T) {
	// Test nested structure
	child := &MacroNode{
		Name:       "code",
		Parameters: map[string]string{"language": "go"},
		Body:       "fmt.Println(\"hello\")",
	}

	parent := &MacroNode{
		Name:       "expand",
		Parameters: map[string]string{"title": "Show code"},
		Body:       "",
		Children:   []*MacroNode{child},
	}

	assert.Equal(t, "expand", parent.Name)
	assert.Len(t, parent.Children, 1)
	assert.Equal(t, "code", parent.Children[0].Name)
	assert.Equal(t, "go", parent.Children[0].Parameters["language"])
}
```

**Step 2: Run tests**

Run: `go test -v ./pkg/md/ -run TestMacro`
Expected: All tests pass

**Step 3: Commit**

```bash
git add pkg/md/macro_test.go
git commit -m "test(md): add macro registry and MacroNode tests

Tests cover:
- Registry contains all expected macros
- Case-insensitive lookup
- Body type configuration per macro
- MacroNode construction and nesting"
```

---

### Task 5: Run full test suite to verify no regressions

**Step 1: Run all pkg/md tests**

Run: `go test -v ./pkg/md/...`
Expected: All tests pass (existing + new)

**Step 2: Run all tests with race detection**

Run: `go test -race ./...`
Expected: All tests pass
