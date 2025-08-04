package ini_test

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/DevReaper0/declarch/modules/config/ini"
)

func TestINIParser_GenerateWithInline(t *testing.T) {
	opts := ini.Options{
		AllowInlineComment: true,
	}
	parser := ini.NewParser(opts)
	testFile := "test_with_inline.conf"
	defer os.Remove(testFile)

	original := `
# Header comment
[config]
Key1 = Value1 # inline comment
Key2 = Value2
`
	os.WriteFile(testFile, []byte(original), 0o644)

	root, err := parser.Parse(testFile)
	assert.NoError(t, err)

	section := root.Children[2]

	assert.Equal(t, "Value1", section.Children[0].Value)
	assert.Equal(t, " # inline comment", section.Children[0].InlineComment)

	assert.Equal(t, "Value2", section.Children[1].Value)
	assert.Equal(t, "", section.Children[1].InlineComment)

	generated, err := parser.Generate(root)
	assert.NoError(t, err)

	expected := `
# Header comment
[config]
Key1 = Value1 # inline comment
Key2 = Value2
`
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(string(generated)))
}

func TestINIParser_GenerateWithoutInline(t *testing.T) {
	opts := ini.Options{
		AllowInlineComment: false,
	}
	parser := ini.NewParser(opts)
	testFile := "test_without_inline.conf"
	defer os.Remove(testFile)

	original := `
# Header comment
[config]
Key1 = Value1 # inline comment
Key2 = Value2
`
	os.WriteFile(testFile, []byte(original), 0o644)

	root, err := parser.Parse(testFile)
	assert.NoError(t, err)

	section := root.Children[2]

	assert.Equal(t, "Value1 # inline comment", section.Children[0].Value)
	assert.Equal(t, "", section.Children[0].InlineComment)

	assert.Equal(t, "Value2", section.Children[1].Value)
	assert.Equal(t, "", section.Children[1].InlineComment)

	generated, err := parser.Generate(root)
	assert.NoError(t, err)

	expected := `
# Header comment
[config]
Key1 = Value1 # inline comment
Key2 = Value2
`
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(string(generated)))
}

func TestINIParser_BooleanKeys(t *testing.T) {
	opts := ini.Options{
		AllowBooleanKeys:   true,
		AllowInlineComment: true,
	}
	parser := ini.NewParser(opts)
	testFile := "test_boolean_keys.conf"
	defer os.Remove(testFile)

	original := `
[booleans]
KeyWithoutEquals
KeyWithEquals = Something
`

	os.WriteFile(testFile, []byte(original), 0o644)
	root, err := parser.Parse(testFile)
	assert.NoError(t, err)

	// Verify generation preserves structure
	generated, err := parser.Generate(root)
	assert.NoError(t, err)
	assert.Equal(t, strings.TrimSpace(original), strings.TrimSpace(string(generated)))
}

func TestINIParser_Subsections(t *testing.T) {
	opts := ini.Options{
		AllowInlineComment: true,
		AllowBooleanKeys:   true,
	}
	parser := ini.NewParser(opts)
	testFile := "test_subsections.conf"
	defer os.Remove(testFile)

	original := `
[parent]
ParentKey = Value1

[parent.child]
ChildKey = Value2
ChildBool

[parent.child.grandchild]
GrandchildKey = Value3

[other]
OtherKey = Value4
`
	os.WriteFile(testFile, []byte(original), 0o644)

	root, err := parser.Parse(testFile)
	assert.NoError(t, err)

	expectedTree := `
Section: <root>
  Blank
  Section: parent
    Key: ParentKey = Value1
    Blank
  Section: parent.child
    Key: ChildKey = Value2
    Boolean: ChildBool
    Blank
  Section: parent.child.grandchild
    Key: GrandchildKey = Value3
    Blank
  Section: other
    Key: OtherKey = Value4
`

	assert.Equal(t, strings.TrimSpace(expectedTree), strings.TrimSpace(root.String()))

	// Verify generation preserves structure
	generated, err := parser.Generate(root)
	assert.NoError(t, err)
	assert.Equal(t, strings.TrimSpace(original), strings.TrimSpace(string(generated)))
}