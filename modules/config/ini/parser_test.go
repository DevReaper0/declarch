package ini

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestINIParser_GenerateWithInline(t *testing.T) {
	opts := Options{
		AllowInlineComment: true,
	}
	parser := NewParser(opts)
	testFile := "test_with_inline.conf"
	defer os.Remove(testFile)

	original := `
# Header comment
[config]
Key1 = Value1 # inline comment
Key2 = Value2
`
	os.WriteFile(testFile, []byte(original), 0644)

	root, err := parser.Parse(testFile)
	assert.NoError(t, err)

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
	opts := Options{
		AllowInlineComment: false,
	}
	parser := NewParser(opts)
	testFile := "test_without_inline.conf"
	defer os.Remove(testFile)

	original := `
# Header comment
[config]
Key1 = Value1 # inline comment to be removed
Key2 = Value2
`
	os.WriteFile(testFile, []byte(original), 0644)

	root, err := parser.Parse(testFile)
	assert.NoError(t, err)

	generated, err := parser.Generate(root)
	assert.NoError(t, err)

	expected := `
# Header comment
[config]
Key1 = Value1 # inline comment to be removed
Key2 = Value2
`
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(string(generated)))
}
