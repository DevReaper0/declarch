package modules

import (
	"fmt"
	"strings"

	"github.com/fatih/color"

	"github.com/DevReaper0/declarch/parser"
)

// TagSet represents a collection of include/exclude tags
type TagSet struct {
	tags []string
}

// NewTagSet creates a new TagSet with the given initial tags
func NewTagSet(initialTags ...string) *TagSet {
	return &TagSet{tags: initialTags}
}

// AddTag adds a new tag to the TagSet
func (ts *TagSet) AddTag(tag string) {
	ts.tags = append(ts.tags, tag)
}

// AddTags adds multiple tags to the TagSet
func (ts *TagSet) AddTags(tags []string) {
	ts.tags = append(ts.tags, tags...)
}

// HasTag checks if the TagSet contains a specific tag
func (ts *TagSet) HasTag(tag string) bool {
	for _, t := range ts.tags {
		if t == tag {
			return true
		}
	}
	return false
}

// GetAll returns all values for a key that match the current tag set
// If an item has spaces, it will be split into multiple items.
func (ts *TagSet) GetAll(section *parser.Section, key string) []string {
	items := section.GetAll(key)
	result := []string{}
	for i := 0; i < len(items); i++ {
		included := true

		parts := strings.Split(items[i], ",")

		// Check for tagging, e.g., "pkg, +!bare" etc.
		if len(parts) > 1 {
			tagPart := strings.TrimSpace(parts[1])
			linkedTags := strings.Fields(tagPart)

			for _, linkedTag := range linkedTags {
				if !strings.HasPrefix(linkedTag, "+") {
					color.Set(color.FgRed)
					fmt.Printf("Invalid tag format in '%s': %s\n", key, linkedTag)
					color.Unset()
					continue
				}

				tagName := strings.TrimPrefix(linkedTag, "+")
				isRequired := false
				if strings.HasPrefix(tagName, "!") {
					included = false
					tagName = strings.TrimPrefix(tagName, "!")
					isRequired = true
				}

				for _, tag := range ts.tags {
					if tag == "+"+tagName || (!isRequired && tag == "+default") {
						included = true
					} else if tag == "-"+tagName || (!isRequired && tag == "-default") {
						included = false
					}
				}
			}
		} else {
			for _, tag := range ts.tags {
				switch tag {
				case "+default":
					included = true
				case "-default":
					included = false
				}
			}
		}

		if included {
			valuesPart := strings.TrimSpace(parts[0])
			values := strings.Fields(valuesPart)

			result = append(result, values...)
		}
	}
	return result
}