package ini

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestINIPatcher_VariedModifications(t *testing.T) {
	opts := Options{
		AllowInlineComment: true,
	}
	parser := NewParser(opts)
	patcher := &Patcher{}
	testFile := "test_patch.conf"
	defer os.Remove(testFile)

	original := `
# Global comment
[alpha]
Key1 = Value1 # comment1
Key2 = Value2
Key3 = Value3


[beta]
KeyA = A1
KeyB = B1
`
	os.WriteFile(testFile, []byte(original), 0644)

	modifications := map[string]interface{}{
		"alpha": map[string]interface{}{
			"Key1":   "NewValue1", // modify
			"Key2":   "",          // remove
			"KeyNew": "AddedValue",
			"subalpha": map[string]interface{}{ // new subsection
				"KeySub": "SubValue",
			},
		},
		"gamma": map[string]interface{}{ // new section
			"KeyX": "ValueX",
		},
	}

	err := patcher.Patch(parser, testFile, modifications)
	assert.NoError(t, err)

	resultBytes, _ := os.ReadFile(testFile)
	result := strings.TrimSpace(string(resultBytes))

	expected := `
# Global comment
[alpha]
Key1 = NewValue1 # comment1
Key3 = Value3
KeyNew = AddedValue

[alpha.subalpha]
KeySub = SubValue


[beta]
KeyA = A1
KeyB = B1

[gamma]
KeyX = ValueX
`
	assert.Equal(t, strings.TrimSpace(expected), result)
}

func TestINIPatcher_NoChanges(t *testing.T) {
	opts := Options{
		AllowInlineComment: true,
	}
	parser := NewParser(opts)
	patcher := &Patcher{}
	testFile := "test_no_change.conf"
	defer os.Remove(testFile)

	original := `
[section]
Key1 = Value1
`
	os.WriteFile(testFile, []byte(original), 0644)

	modifications := map[string]interface{}{}

	err := patcher.Patch(parser, testFile, modifications)
	assert.NoError(t, err)

	resultBytes, _ := os.ReadFile(testFile)
	result := strings.TrimSpace(string(resultBytes))
	assert.Equal(t, strings.TrimSpace(original), result)
}
