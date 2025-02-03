package ini

import (
	"os"
)

type Patcher struct{}

// Patch performs in-memory modifications by parsing the file into nodes,
// updating those nodes recursively, then generating the updated file content.
func (p *Patcher) Patch(parser *Parser, filePath string, modifications map[string]interface{}) error {
	root, err := parser.Parse(filePath)
	if err != nil {
		return err
	}

	p.applyModifications(root, modifications)

	content, err := parser.Generate(root)
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, content, 0644)
}

// applyModifications applies updates to the given node based on modifications.
// If the modification value is a string, it updates/removes a key in the current node.
// If the modification value is a map, it recurses into the corresponding section.
func (p *Patcher) applyModifications(node *Node, mods map[string]interface{}) {
	for key, mod := range mods {
		if val, ok := mod.(string); ok { // key-value update
			if val == "" {
				p.removeKey(node, key)
			} else {
				if p.hasKey(node, key) {
					p.modifyExistingKey(node, key, val)
				} else {
					p.insertKeyBeforeBlankLines(node, key, val)
				}
			}
		} else if secMods, ok := mod.(map[string]interface{}); ok { // section update
			secNode := p.findOrCreateSectionNode(node, key)
			p.applyModifications(secNode, secMods)
		}
	}
}

func (p *Patcher) modifyExistingKey(sectionNode *Node, key, value string) {
	for _, child := range sectionNode.Children {
		if child.Type == NodeKey && child.Key == key {
			child.Value = value
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

func (p *Patcher) findOrCreateSectionNode(root *Node, name string) *Node {
	// Determine full section name based on parent's type.
	var fullName string
	if root.Type == NodeSection {
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
		if !(child.Type == NodeKey && child.Key == key) {
			newChildren = append(newChildren, child)
		}
	}
	sectionNode.Children = newChildren
}

func (p *Patcher) hasKey(sectionNode *Node, key string) bool {
	for _, child := range sectionNode.Children {
		if child.Type == NodeKey && child.Key == key {
			return true
		}
	}
	return false
}
