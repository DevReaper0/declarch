package ini

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const rootMarker = "<root>"

type Options struct {
	AllowInlineComment bool
	AllowBooleanKeys   bool
	CommentChar        string
}

type Parser struct {
	options Options
}

func NewParser(opts Options) *Parser {
	if opts.CommentChar == "" {
		opts.CommentChar = "#"
	}
	return &Parser{options: opts}
}

func (p *Parser) Parse(filePath string) (*Node, error) {
	root := NewNode(NodeSection, rootMarker, "")
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var currentSection *Node

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		switch {
		case trimmed == "":
			blankNode := NewNode(NodeBlank, line, "")
			if currentSection == nil {
				root.Children = append(root.Children, blankNode)
			} else {
				currentSection.Children = append(currentSection.Children, blankNode)
			}
		case strings.HasPrefix(trimmed, p.options.CommentChar):
			if currentSection == nil {
				root.Children = append(root.Children, NewNode(NodeComment, line, ""))
			} else {
				currentSection.Children = append(currentSection.Children, NewNode(NodeComment, line, ""))
			}
		case strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]"):
			name := trimmed[1 : len(trimmed)-1]
			sec := NewNode(NodeSection, name, "")
			root.Children = append(root.Children, sec)
			currentSection = sec
		default:
			if p.options.AllowBooleanKeys && !strings.Contains(line, "=") {
				boolNode := NewNode(NodeBoolean, trimmed, "")
				if currentSection == nil {
					root.Children = append(root.Children, boolNode)
				} else {
					currentSection.Children = append(currentSection.Children, boolNode)
				}
				continue
			}
			key, val, inlineComment := p.splitKeyValueWithComment(line)
			keyNode := NewNode(NodeKey, key, val)
			keyNode.InlineComment = inlineComment
			keyNode.Raw = line // Preserve original formatting
			if currentSection == nil {
				root.Children = append(root.Children, keyNode)
			} else {
				currentSection.Children = append(currentSection.Children, keyNode)
			}
		}
	}

	return root, nil
}

func (p *Parser) Generate(root *Node) ([]byte, error) {
	lines := p.buildLines(root)
	output := strings.Join(lines, "\n")
	return []byte(output), nil
}

func (p *Parser) splitKeyValue(line string) (string, string) {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return strings.TrimSpace(line), ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

func (p *Parser) splitKeyValueWithComment(line string) (string, string, string) {
	var inlineComment string
	if p.options.AllowInlineComment {
		if idx := strings.Index(line, p.options.CommentChar); idx != -1 {
			start := idx
			for start > 0 && (line[start-1] == ' ' || line[start-1] == '\t') {
				start--
			}
			inlineComment = line[start:]
			line = strings.TrimSpace(line[:start])
		}
	}
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return strings.TrimSpace(line), "", inlineComment
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), inlineComment
}

func (p *Parser) buildLines(node *Node) []string {
	var lines []string
	for _, child := range node.Children {
		switch child.Type {
		case NodeComment:
			lines = append(lines, child.Key)
		case NodeBlank:
			lines = append(lines, child.Key)
		case NodeSection:
			// Don't output section marker for root node
			if child.Key != rootMarker {
				lines = append(lines, fmt.Sprintf("[%s]", child.Key))
			}
			secLines := p.buildLines(child)
			lines = append(lines, secLines...)
		case NodeKey:
			if child.Raw != "" {
				lines = append(lines, child.Raw)
			} else {
				line := fmt.Sprintf("%s = %s", child.Key, child.Value)
				if child.InlineComment != "" {
					line += child.InlineComment
				}
				lines = append(lines, line)
			}
		case NodeBoolean:
			lines = append(lines, child.Key)
		default:
		}
	}
	return lines
}