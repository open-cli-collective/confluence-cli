package md

import (
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
