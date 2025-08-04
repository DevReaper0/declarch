package ini

import (
	"os"
	"slices"
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
	// First pass: process modifications for keys already present.
	for key, mod := range mods {
		if val, ok := mod.(string); ok { // key-value update
			if p.hasKey(node, key) {
				if val == "" {
					p.removeKey(node, key)
				} else {
					p.modifyExistingKey(node, key, val)
				}
			} else if p.ReplaceComments && p.hasCommentedKey(node, key, parser.options) {
				p.replaceCommentedKey(node, key, val, parser.options)
			}
		}
	}
	// Second pass: collect new keys (for insertion) and sort them.
	var newKeys []string
	for key, mod := range mods {
		if _, ok := mod.(string); ok {
			if !p.hasKey(node, key) && !(p.ReplaceComments && p.hasCommentedKey(node, key, parser.options)) {
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
		if val == "~EMPTY" {
			val = ""
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
			secNode := p.findOrCreateSectionNode(node, key, parser.options)
			p.applyModifications(parser, secNode, secMods)
		}
	}
}

func (p *Patcher) modifyExistingKey(sectionNode *Node, key, value string) {
	if value == "~EMPTY" {
		value = ""
	}
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

func (p *Patcher) findOrCreateSectionNode(root *Node, name string, opts Options) *Node {
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
		sectionNodes := []*Node{}
		for _, child := range root.Children {
			if child.Type == NodeSection {
				sectionNodes = append(sectionNodes, child)
			}
		}

		for _, sectionNode := range sectionNodes {
			var currentSectionNode *Node
			for idx, child := range sectionNode.Children {
				if child.Type == NodeComment {
					trimmedLine := strings.TrimSpace(child.Key)
					if strings.HasPrefix(trimmedLine, opts.CommentChar) {
						uncommented := strings.TrimSpace(trimmedLine[len(opts.CommentChar):])
						if strings.HasPrefix(uncommented, "[") && strings.HasSuffix(uncommented, "]") {
							secName := strings.TrimSuffix(strings.TrimPrefix(uncommented, "["), "]")
							if secName == name {
								newSectionBlock := sectionNode.Children[idx+1:]
								sectionNode.Children = sectionNode.Children[:idx]

								currentSectionNode = NewNode(NodeSection, secName, "")
								currentSectionNode.Children = newSectionBlock
								idxRoot := slices.Index(root.Children, sectionNode)
								root.Children = append(root.Children[:idxRoot+1], append([]*Node{currentSectionNode}, root.Children[idxRoot+1:]...)...)

								return currentSectionNode
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

func (p *Patcher) hasCommentedKey(sectionNode *Node, key string, opts Options) bool {
	for _, child := range sectionNode.Children {
		if child.Type == NodeComment {
			trimmed := strings.TrimSpace(child.Key)
			if strings.HasPrefix(trimmed, opts.CommentChar) {
				uncommented := strings.TrimSpace(trimmed[len(opts.CommentChar):])
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

func (p *Patcher) replaceCommentedKey(sectionNode *Node, key, value string, opts Options) {
	for i, child := range sectionNode.Children {
		if child.Type == NodeComment {
			trimmed := strings.TrimSpace(child.Key)
			if strings.HasPrefix(trimmed, opts.CommentChar) {
				uncommented := strings.TrimSpace(trimmed[len(opts.CommentChar):])
				if strings.HasPrefix(uncommented, key) {
					rest := strings.TrimPrefix(uncommented, key)
					if rest == "" || strings.HasPrefix(rest, " ") || strings.HasPrefix(rest, "=") {
						if opts.AllowBooleanKeys && value == "~BOOL" {
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