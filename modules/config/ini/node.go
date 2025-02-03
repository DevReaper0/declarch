package ini

type NodeType int

const (
	NodeUnknown NodeType = iota
	NodeComment
	NodeBlank
	NodeSection
	NodeKey
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
