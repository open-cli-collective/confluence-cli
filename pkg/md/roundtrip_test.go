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
