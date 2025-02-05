package ini

import (
	"os"
	"sort"
	"strings"
)

type Patcher struct {
	ReplaceComments bool
}

// Patch performs in-memory modifications by parsing the file into nodes,
// updating those nodes recursively, then generating the updated file content.
func (p *Patcher) Patch(parser *Parser, filePath string, modifications map[string]interface{}) error {
	root, err := parser.Parse(filePath)
	if err != nil {
		return err
	}

	p.applyModifications(parser, root, modifications)

	content, err := parser.Generate(root)
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, content, 0o644)
}

// applyModifications applies updates to the given node based on modifications.
// If the modification value is a string, it updates/removes a key in the current node.
// If the modification value is a map, it recurses into the corresponding section.
func (p *Patcher) applyModifications(parser *Parser, node *Node, mods map[string]interface{}) {
	commentChar := parser.options.CommentChar
	// Uncomment sections
	if p.ReplaceComments {
		// Note: Commented out sections will be extracted in findOrCreateSectionNode.
	}

	// First pass: process modifications for keys already present.
	for key, mod := range mods {
		if val, ok := mod.(string); ok { // key-value update
			if p.hasKey(node, key) {
				if val == "" {
					p.removeKey(node, key)
				} else {
					p.modifyExistingKey(node, key, val)
				}
			} else if p.ReplaceComments && p.hasCommentedKey(node, key, commentChar) {
				p.replaceCommentedKey(node, key, val, commentChar)
			} else {
				// Skip new keys; process in second pass.
			}
		}
	}
	// Second pass: collect new keys (for insertion) and sort them.
	var newKeys []string
	for key, mod := range mods {
		if _, ok := mod.(string); ok {
			if !p.hasKey(node, key) && !(p.ReplaceComments && p.hasCommentedKey(node, key, commentChar)) {
				newKeys = append(newKeys, key)
			}
		}
	}
	// Lexicographical order ensures deterministic insertion order.
	sort.Strings(newKeys)
	for _, key := range newKeys {
		val := mods[key].(string)
		if val == "" {
			// Removing non-existent key: ignore.
			continue
		}
		if parser.options.AllowBooleanKeys && strings.HasSuffix(val, "~BOOL") {
			p.insertBooleanBeforeBlankLines(node, key)
		} else {
			p.insertKeyBeforeBlankLines(node, key, val)
		}
	}
	// Then, process section modifications (map values)
	for key, mod := range mods {
		if secMods, ok := mod.(map[string]interface{}); ok { // section update
			secNode := p.findOrCreateSectionNode(node, key, commentChar)
			p.applyModifications(parser, secNode, secMods)
		}
	}
}

func (p *Patcher) modifyExistingKey(sectionNode *Node, key, value string) {
	for _, child := range sectionNode.Children {
		if (child.Type == NodeKey || child.Type == NodeBoolean) && child.Key == key {
			child.Value = value
			child.Raw = "" // mark as modified so that new formatting is applied
		}
	}
}

// Insert the new key before any trailing blank/comment at the end of the section.
func (p *Patcher) insertKeyBeforeBlankLines(sectionNode *Node, key, value string) {
	idx := len(sectionNode.Children)
	for i := len(sectionNode.Children) - 1; i >= 0; i-- {
		c := sectionNode.Children[i]
		if c.Type != NodeBlank && c.Type != NodeComment {
			idx = i + 1
			break
		}
	}
	newKey := NewNode(NodeKey, key, value)
	sectionNode.Children = append(
		sectionNode.Children[:idx],
		append([]*Node{newKey}, sectionNode.Children[idx:]...)...,
	)
}

// Insert a boolean key at the end, before blank lines.
func (p *Patcher) insertBooleanBeforeBlankLines(sectionNode *Node, key string) {
	idx := len(sectionNode.Children)
	for i := len(sectionNode.Children) - 1; i >= 0; i-- {
		c := sectionNode.Children[i]
		if c.Type != NodeBlank && c.Type != NodeComment {
			idx = i + 1
			break
		}
	}
	newBool := NewNode(NodeBoolean, key, "")
	sectionNode.Children = append(
		sectionNode.Children[:idx],
		append([]*Node{newBool}, sectionNode.Children[idx:]...)...,
	)
}

func (p *Patcher) findOrCreateSectionNode(root *Node, name string, commentChar string) *Node {
	// Determine full section name based on parent's type and name
	var fullName string
	if root.Type == NodeSection && root.Key != rootMarker {
		fullName = root.Key + "." + name
	} else {
		fullName = name
	}
	// Attempt to find an existing section node with the full name.
	for _, child := range root.Children {
		if child.Type == NodeSection && child.Key == fullName {
			return child
		}
	}
	// If ReplaceComments is enabled, look for a commented section header within immediate child sections.
	if p.ReplaceComments {
		for _, sectionNode := range root.Children {
			if sectionNode.Type != NodeSection {
				continue
			}
			for idx, child := range sectionNode.Children {
				if child.Type == NodeComment {
					trimmedLine := strings.TrimSpace(child.Key)
					if strings.HasPrefix(trimmedLine, commentChar) {
						uncommented := strings.TrimSpace(trimmedLine[len(commentChar):])
						if strings.HasPrefix(uncommented, "[") && strings.HasSuffix(uncommented, "]") {
							secName := strings.TrimSuffix(strings.TrimPrefix(uncommented, "["), "]")
							if secName == name {
								// Found commented section header.
								// Move all nodes from this header until the end.
								newSectionBlock := sectionNode.Children[idx:]
								// Remove them from current section.
								sectionNode.Children = sectionNode.Children[:idx]
								// Create a new section node.
								sec := NewNode(NodeSection, fullName, "")
								// Process newSectionBlock: skip header, then trim spaces from each comment.
								for i, n := range newSectionBlock {
									if i == 0 { // skip header comment
										continue
									}
									if n.Type == NodeComment {
										uncommented := strings.TrimSpace(strings.TrimPrefix(n.Key, commentChar))
										if strings.Contains(uncommented, "=") {
											parts := strings.SplitN(uncommented, "=", 2)
											k := strings.TrimSpace(parts[0])
											v := strings.TrimSpace(parts[1])
											newKey := NewNode(NodeKey, k, v)
											sec.Children = append(sec.Children, newKey)
										} else {
											newBool := NewNode(NodeBoolean, uncommented, "")
											sec.Children = append(sec.Children, newBool)
										}
									} else {
										sec.Children = append(sec.Children, n)
									}
								}
								// Insert the new section node into root.
								idxRoot := len(root.Children)
								for i := len(root.Children) - 1; i >= 0; i-- {
									if root.Children[i].Type != NodeBlank && root.Children[i].Type != NodeComment {
										idxRoot = i + 1
										break
									}
								}
								root.Children = append(root.Children[:idxRoot], append([]*Node{sec}, root.Children[idxRoot:]...)...)
								return sec
							}
						}
					}
				}
			}
		}
	}
	// Find insertion index before trailing blank/comment nodes.
	idx := len(root.Children)
	for i := len(root.Children) - 1; i >= 0; i-- {
		if root.Children[i].Type != NodeBlank && root.Children[i].Type != NodeComment {
			idx = i + 1
			break
		}
	}
	// Ensure there's a blank line before the new section.
	if idx > 0 && root.Children[idx-1].Type != NodeBlank {
		blankNode := NewNode(NodeBlank, "", "")
		root.Children = append(root.Children[:idx], append([]*Node{blankNode}, root.Children[idx:]...)...)
		idx++
	}
	// Create and insert the new section node.
	sec := NewNode(NodeSection, fullName, "")
	root.Children = append(root.Children[:idx], append([]*Node{sec}, root.Children[idx:]...)...)
	return sec
}

func (p *Patcher) removeKey(sectionNode *Node, key string) {
	newChildren := make([]*Node, 0, len(sectionNode.Children))
	for _, child := range sectionNode.Children {
		if !((child.Type == NodeKey || child.Type == NodeBoolean) && child.Key == key) {
			newChildren = append(newChildren, child)
		}
	}
	sectionNode.Children = newChildren
}

func (p *Patcher) hasKey(sectionNode *Node, key string) bool {
	for _, child := range sectionNode.Children {
		if (child.Type == NodeKey || child.Type == NodeBoolean) && child.Key == key {
			return true
		}
	}
	return false
}

func (p *Patcher) hasCommentedKey(sectionNode *Node, key, commentChar string) bool {
	for _, child := range sectionNode.Children {
		if child.Type == NodeComment {
			trimmed := strings.TrimSpace(child.Key)
			if strings.HasPrefix(trimmed, commentChar) {
				uncommented := strings.TrimSpace(trimmed[len(commentChar):])
				if strings.HasPrefix(uncommented, key) {
					rest := strings.TrimPrefix(uncommented, key)
					if rest == "" || strings.HasPrefix(rest, " ") || strings.HasPrefix(rest, "=") {
						return true
					}
				}
			}
		}
	}
	return false
}

func (p *Patcher) replaceCommentedKey(sectionNode *Node, key, value, commentChar string) {
	for i, child := range sectionNode.Children {
		if child.Type == NodeComment {
			trimmed := strings.TrimSpace(child.Key)
			if strings.HasPrefix(trimmed, commentChar) {
				uncommented := strings.TrimSpace(trimmed[len(commentChar):])
				if strings.HasPrefix(uncommented, key) {
					rest := strings.TrimPrefix(uncommented, key)
					if rest == "" || strings.HasPrefix(rest, " ") || strings.HasPrefix(rest, "=") {
						if value == "~BOOL" {
							sectionNode.Children[i] = NewNode(NodeBoolean, key, "")
						} else {
							// New key node is created without existing raw formatting.
							sectionNode.Children[i] = NewNode(NodeKey, key, value)
						}
						break
					}
				}
			}
		}
	}
}
