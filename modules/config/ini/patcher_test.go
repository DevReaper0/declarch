package ini_test

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/DevReaper0/declarch/modules/config/ini"
)

func TestINIPatcher_ModifyKey(t *testing.T) {
	parser := ini.NewParser(ini.Options{AllowInlineComment: true})
	patcher := &ini.Patcher{}
	testFile := "test_modify_key.conf"
	defer os.Remove(testFile)

	original := `
[section]
Key1 = Value1
`
	os.WriteFile(testFile, []byte(original), 0o644)

	modifications := map[string]interface{}{
		"section": map[string]interface{}{
			"Key1": "NewValue1",
		},
	}

	err := patcher.Patch(parser, testFile, modifications)
	assert.NoError(t, err)

	expected := `
[section]
Key1 = NewValue1
`

	resultBytes, _ := os.ReadFile(testFile)
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(string(resultBytes)))
}

func TestINIPatcher_EmptyValue(t *testing.T) {
	parser := ini.NewParser(ini.Options{AllowInlineComment: true})
	patcher := &ini.Patcher{}
	testFile := "test_empty_value.conf"
	defer os.Remove(testFile)

	original := `
[section]
Key1 = Value1
`
	os.WriteFile(testFile, []byte(original), 0o644)

	modifications := map[string]interface{}{
		"section": map[string]interface{}{
			"Key1": "~EMPTY",
		},
	}

	err := patcher.Patch(parser, testFile, modifications)
	assert.NoError(t, err)

	expected := `
[section]
Key1 =
`

	resultBytes, _ := os.ReadFile(testFile)
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(string(resultBytes)))
}

func TestINIPatcher_RemoveKey(t *testing.T) {
	parser := ini.NewParser(ini.Options{AllowInlineComment: true})
	patcher := &ini.Patcher{}
	testFile := "test_remove_key.conf"
	defer os.Remove(testFile)

	original := `
[section]
Key1 = Value1
Key2 = Value2
`
	os.WriteFile(testFile, []byte(original), 0o644)

	modifications := map[string]interface{}{
		"section": map[string]interface{}{
			"Key2": "",
		},
	}

	err := patcher.Patch(parser, testFile, modifications)
	assert.NoError(t, err)

	expected := `
[section]
Key1 = Value1
`

	resultBytes, _ := os.ReadFile(testFile)
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(string(resultBytes)))
}

func TestINIPatcher_AddKey(t *testing.T) {
	parser := ini.NewParser(ini.Options{AllowInlineComment: true})
	patcher := &ini.Patcher{}
	testFile := "test_add_key.conf"
	defer os.Remove(testFile)

	original := `
[section]
Key1 = Value1
`
	os.WriteFile(testFile, []byte(original), 0o644)

	modifications := map[string]interface{}{
		"section": map[string]interface{}{
			"KeyNew": "Value3",
		},
	}

	err := patcher.Patch(parser, testFile, modifications)
	assert.NoError(t, err)

	expected := `
[section]
Key1 = Value1
KeyNew = Value3
`

	resultBytes, _ := os.ReadFile(testFile)
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(string(resultBytes)))
}

func TestINIPatcher_AddSection(t *testing.T) {
	parser := ini.NewParser(ini.Options{AllowInlineComment: true})
	patcher := &ini.Patcher{}
	testFile := "test_add_section.conf"
	defer os.Remove(testFile)

	original := `
[existing]
Key = Value
`
	os.WriteFile(testFile, []byte(original), 0o644)

	modifications := map[string]interface{}{
		"new_section": map[string]interface{}{
			"NewKey": "NewValue",
		},
	}

	err := patcher.Patch(parser, testFile, modifications)
	assert.NoError(t, err)

	expected := `
[existing]
Key = Value

[new_section]
NewKey = NewValue
`

	resultBytes, _ := os.ReadFile(testFile)
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(string(resultBytes)))
}

func TestINIPatcher_AddSubsection(t *testing.T) {
	parser := ini.NewParser(ini.Options{AllowInlineComment: true})
	patcher := &ini.Patcher{}
	testFile := "test_add_subsection.conf"
	defer os.Remove(testFile)

	original := `
[existing]
Key = Value
`
	os.WriteFile(testFile, []byte(original), 0o644)

	modifications := map[string]interface{}{
		"existing": map[string]interface{}{
			"subsection": map[string]interface{}{
				"SubKey": "SubValue",
			},
		},
	}

	err := patcher.Patch(parser, testFile, modifications)
	assert.NoError(t, err)

	expected := `
[existing]
Key = Value

[existing.subsection]
SubKey = SubValue
`

	resultBytes, _ := os.ReadFile(testFile)
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(string(resultBytes)))
}

func TestINIPatcher_PreserveComments(t *testing.T) {
	parser := ini.NewParser(ini.Options{AllowInlineComment: true})
	patcher := &ini.Patcher{}
	testFile := "test_preserve_comments.conf"
	defer os.Remove(testFile)

	original := `
# Header comment
[section]
Key1 = Value1 # inline comment
Key2 = Value2

# Section comment
[other]
KeyA = ValueA
`
	os.WriteFile(testFile, []byte(original), 0o644)

	modifications := map[string]interface{}{
		"section": map[string]interface{}{
			"Key1": "NewValue1",
		},
	}

	err := patcher.Patch(parser, testFile, modifications)
	assert.NoError(t, err)

	expected := `
# Header comment
[section]
Key1 = NewValue1 # inline comment
Key2 = Value2

# Section comment
[other]
KeyA = ValueA
`

	resultBytes, _ := os.ReadFile(testFile)
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(string(resultBytes)))
}

func TestINIPatcher_BooleanKeys(t *testing.T) {
	parser := ini.NewParser(ini.Options{AllowBooleanKeys: true})
	patcher := &ini.Patcher{}
	testFile := "test_boolean_keys.conf"
	defer os.Remove(testFile)

	original := `
[section]
ExistingBool
Key = Value
`
	os.WriteFile(testFile, []byte(original), 0o644)

	modifications := map[string]interface{}{
		"section": map[string]interface{}{
			"ExistingBool": "",
			"NewBool":      "~BOOL",
			"Key":          "NewValue",
		},
	}

	err := patcher.Patch(parser, testFile, modifications)
	assert.NoError(t, err)

	expected := `
[section]
Key = NewValue
NewBool
`

	resultBytes, _ := os.ReadFile(testFile)
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(string(resultBytes)))
}

func TestINIPatcher_ReplaceCommentedKey(t *testing.T) {
	parser := ini.NewParser(ini.Options{AllowInlineComment: true})
	patcher := &ini.Patcher{ReplaceComments: true}
	testFile := "test_replace_commented_key.conf"
	defer os.Remove(testFile)

	original := `
[section]
#CommentedKey = OldValue
ActiveKey = Value
`
	os.WriteFile(testFile, []byte(original), 0o644)

	modifications := map[string]interface{}{
		"section": map[string]interface{}{
			"CommentedKey": "NewValue",
		},
	}

	err := patcher.Patch(parser, testFile, modifications)
	assert.NoError(t, err)

	expected := `
[section]
CommentedKey = NewValue
ActiveKey = Value
`

	resultBytes, _ := os.ReadFile(testFile)
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(string(resultBytes)))
}

func TestINIPatcher_UncommentSection(t *testing.T) {
	parser := ini.NewParser(ini.Options{AllowInlineComment: true})
	patcher := &ini.Patcher{ReplaceComments: true}
	testFile := "test_uncomment_section.conf"
	defer os.Remove(testFile)

	original := `
[active]
Key = Value

#[qwe]
#Qwe = 123

# [commented]
#Key1 = Value1
#Key2 = Value2

#[asd]
#Asd = 456

[zxc]
Zxc = 789
`
	os.WriteFile(testFile, []byte(original), 0o644)

	modifications := map[string]interface{}{
		"commented": map[string]interface{}{
			"Key1": "NewValue1",
		},
	}

	err := patcher.Patch(parser, testFile, modifications)
	assert.NoError(t, err)

	expected := `
[active]
Key = Value

#[qwe]
#Qwe = 123

[commented]
Key1 = NewValue1
#Key2 = Value2

#[asd]
#Asd = 456

[zxc]
Zxc = 789
`

	resultBytes, _ := os.ReadFile(testFile)
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(string(resultBytes)))
}