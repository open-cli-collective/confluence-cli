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
