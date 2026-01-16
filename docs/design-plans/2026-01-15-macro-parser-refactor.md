# Macro Parser Refactor Design

## Overview

Replace regex-based macro parsing in `pkg/md/` with a proper tokenizer and parser architecture. This refactor enables trivial addition of new Confluence macros (one line in registry vs. new regex patterns) while maintaining all existing behavior.

**Goals:**
- Adding a new macro = adding one line to MacroRegistry
- Heavily tested foundation for future macro work
- Warn on malformed macros, pass through content unchanged
- All existing tests pass without modification

## Architecture

Custom tokenizer + parser for both conversion directions, unified by a shared `MacroNode` representation.

**Core data structures:**

```go
// MacroNode represents a parsed macro in either direction
type MacroNode struct {
    Name       string            // "toc", "info", "warning", etc.
    Parameters map[string]string // key-value pairs
    Body       string            // raw content (for rich-text-body macros)
    Children   []*MacroNode      // nested macros within body
}

// MacroType defines behavior for a specific macro
type MacroType struct {
    Name        string
    HasBody     bool   // true for panels, false for TOC
    BodyType    string // "rich-text" (HTML) or "plain-text" (CDATA)
    RenderFunc  func(node *MacroNode) string // optional custom rendering
}

// MacroRegistry - the extensibility point
var MacroRegistry = map[string]MacroType{...}
```

**Two tokenizers, one parser pattern:**

| Direction | Tokenizer | Input | Output |
|-----------|-----------|-------|--------|
| MD→XHTML | BracketTokenizer | `[INFO]...[/INFO]` | BracketToken stream |
| XHTML→MD | XMLTokenizer | `<ac:structured-macro>` | XMLToken stream |

Both tokenizers feed into a parser that builds `MacroNode` trees using stack-based depth tracking. The parser consults `MacroRegistry` to determine if a macro expects a body and what type.

**Data flow:**

```
MD→XHTML:
  input → BracketTokenizer → tokens → Parser → MacroNodes/Segments
  → placeholders inserted → goldmark → placeholders replaced with XHTML

XHTML→MD:
  input → XMLTokenizer → tokens → Parser → MacroNodes/Segments
  → placeholders inserted → html-to-markdown → placeholders replaced with [MACRO]
```

Placeholder system retained because goldmark and html-to-markdown don't understand macros. The parser makes extraction and reinsertion clean.

## Existing Patterns

Investigation found current implementation in `pkg/md/converter.go` and `pkg/md/from_html.go`:

**Current patterns being replaced:**
- Regex-based macro extraction with hardcoded panel types array (line 64)
- `findMatchingCloseTag()` using depth counting (already parser-like)
- Two-phase placeholder processing (preprocess → convert → postprocess)
- Parameter parsing via `parseKeyValueParams()` with quote handling

**Patterns being preserved:**
- Placeholder naming convention (`CFMACRO{n}END`, `CFMACROOPEN{n}`, `CFMACROCLOSE{n}`)
- External API unchanged (`ToConfluenceStorage`, `FromConfluenceStorageWithOptions`)
- `ConvertOptions` struct with `ShowMacros` flag
- Test file organization (`converter_test.go`, `from_html_test.go`)

**Pattern evolution:**
- Hardcoded `panelTypes := []string{"INFO", "WARNING"...}` → `MacroRegistry` map
- Regex extraction → Tokenizer + Parser
- String manipulation for nesting → Stack-based tree building

## Implementation Phases

### Phase 1: Core Data Structures

**Goal:** Define MacroNode, MacroType, MacroRegistry, and token types

**Components:**
- `pkg/md/macro.go` — MacroNode struct, MacroType struct, MacroRegistry with current macros (toc, info, warning, note, tip, expand, code)
- `pkg/md/tokens.go` — BracketToken, BracketTokenType, XMLToken, XMLTokenType definitions
- `pkg/md/macro_test.go` — Registry lookup tests, MacroNode construction tests

**Dependencies:** None (first phase)

**Done when:** Types compile, registry contains all current macros, basic unit tests pass

### Phase 2: Bracket Syntax Tokenizer

**Goal:** Tokenize `[MACRO]...[/MACRO]` bracket syntax for MD→XHTML direction

**Components:**
- `pkg/md/tokenizer_bracket.go` — `TokenizeBrackets(input string) ([]BracketToken, error)`
- `pkg/md/tokenizer_bracket_test.go` — Comprehensive tokenizer tests

**Test coverage required:**
- Single macros: `[TOC]`, `[INFO]...[/INFO]`
- Parameters: quoted, unquoted, mixed, escaped quotes, empty values
- Nesting: 2-3 levels deep
- Malformed: unclosed tags, mismatched names, orphan close tags
- Edge cases: brackets in quoted values, brackets in body text

**Dependencies:** Phase 1 (token type definitions)

**Done when:** Tokenizer correctly splits all bracket syntax variations into typed tokens, all edge case tests pass

### Phase 3: Confluence XML Tokenizer

**Goal:** Tokenize `<ac:structured-macro>` XML for XHTML→MD direction

**Components:**
- `pkg/md/tokenizer_xml.go` — `TokenizeConfluenceXML(input string) ([]XMLToken, error)`
- `pkg/md/tokenizer_xml_test.go` — Comprehensive tokenizer tests

**Test coverage required:**
- Single macros with parameters
- Nested macros (depth 2, 3, 4)
- CDATA sections in plain-text-body (including CDATA end sequence in content)
- Rich-text-body with HTML content
- Whitespace variations, entity handling
- Malformed: unclosed tags, mismatched names

**Dependencies:** Phase 1 (token type definitions)

**Done when:** Tokenizer correctly splits all Confluence XML variations into typed tokens, all edge case tests pass

### Phase 4: Parser Implementation

**Goal:** Build MacroNode trees from token streams for both directions

**Components:**
- `pkg/md/parser.go` — `ParseResult`, `Segment`, parser interface and implementation
- `pkg/md/parser_bracket.go` — `ParseBracketMacros(input string) (*ParseResult, error)`
- `pkg/md/parser_xml.go` — `ParseConfluenceXML(input string) (*ParseResult, error)`
- `pkg/md/parser_test.go` — Parser tests for tree building

**Test coverage required:**
- Single macro → correct MacroNode with parameters
- Nested macros → Children populated correctly
- Multiple top-level macros → multiple Nodes
- Segments interleave correctly (text, macro, text, macro)
- Unknown macros → warning logged, passthrough as text
- Malformed input → warning logged, graceful degradation

**Dependencies:** Phases 2 and 3 (tokenizers)

**Done when:** Parsers build correct MacroNode trees from token streams, nesting works, error handling works

### Phase 5: Integration with converter.go

**Goal:** Replace regex-based extraction in MD→XHTML direction with parser

**Components:**
- `pkg/md/converter.go` — Replace `preprocessMacros()` internals with `ParseBracketMacros()`, replace `postprocessMacros()` with segment-based rendering

**Constraints:**
- External API unchanged (`ToConfluenceStorage` signature)
- All existing tests in `converter_test.go` must pass unchanged
- Placeholder naming convention preserved for compatibility

**Dependencies:** Phase 4 (parser)

**Done when:** All existing `converter_test.go` tests pass, MD→XHTML conversion works identically to before

### Phase 6: Integration with from_html.go

**Goal:** Replace regex-based extraction in XHTML→MD direction with parser

**Components:**
- `pkg/md/from_html.go` — Replace `processConfluenceMacrosWithPlaceholders()` internals with `ParseConfluenceXML()`, replace placeholder replacement with segment-based rendering

**Constraints:**
- External API unchanged (`FromConfluenceStorageWithOptions` signature)
- All existing tests in `from_html_test.go` must pass unchanged
- `ShowMacros` option behavior preserved

**Dependencies:** Phase 4 (parser)

**Done when:** All existing `from_html_test.go` tests pass, XHTML→MD conversion works identically to before

## Additional Considerations

**Error handling:** Malformed macros log a warning (via standard library `log` package) and pass through content unchanged. This matches the user-specified "warn and continue" strategy.

**Roundtrip stability:** After both integrations complete, add explicit roundtrip tests in `pkg/md/roundtrip_test.go` covering each registered macro type with various parameter and body combinations.

**Code cleanup:** After integration phases complete, dead code from old regex approach can be removed. The `findMatchingCloseTag()` function and regex patterns become unused.

**Future macros:** With this architecture in place, subsequent PRs add macros by:
1. Adding one line to `MacroRegistry`
2. Adding tests for the new macro's roundtrip behavior
