package ini

import (
	"fmt"
	"strings"
)

type NodeType int

const (
	NodeBlank NodeType = iota
	NodeComment
	NodeSection
	NodeKey
	NodeBoolean
)

type Node struct {
	Type          NodeType
	Key           string
	Value         string
	InlineComment string
	Children      []*Node
}

func NewNode(t NodeType, key, value string, children ...*Node) *Node {
	return &Node{
		Type:     t,
		Key:      key,
		Value:    value,
		Children: children,
	}
}

func (n *Node) String() string {
	return n.debugString(0)
}

func (n *Node) debugString(level int) string {
	indent := strings.Repeat("  ", level)
	var sb strings.Builder

	// Write node type
	sb.WriteString(indent)
	sb.WriteString(n.Type.String())

	// Add content based on node type
	switch n.Type {
	case NodeKey:
		sb.WriteString(": ")
		sb.WriteString(n.Key)
		sb.WriteString(" = ")
		sb.WriteString(n.Value)
	case NodeSection:
		sb.WriteString(": ")
		sb.WriteString(n.Key)
	case NodeBoolean:
		sb.WriteString(": ")
		sb.WriteString(n.Key)
	case NodeComment, NodeBlank:
		if n.Key != "" {
			sb.WriteString(" ")
			sb.WriteString(n.Key)
		}
	}

	// Write inline comment if present
	if n.InlineComment != "" {
		sb.WriteString(n.InlineComment)
	}
	sb.WriteString("\n")

	// Write children recursively
	for _, child := range n.Children {
		sb.WriteString(child.debugString(level + 1))
	}

	return sb.String()
}

func (t NodeType) String() string {
	switch t {
	case NodeComment:
		return "Comment"
	case NodeBlank:
		return "Blank"
	case NodeSection:
		return "Section"
	case NodeKey:
		return "Key"
	case NodeBoolean:
		return "Boolean"
	default:
		return fmt.Sprintf("NodeType(%d)", t)
	}
}
