package parser

import (
	"os"
	"path/filepath"
	"strings"
)

type Section struct {
	Values    map[string][]string
	Sections  map[string][]*Section
	Variables map[string]string
}

func isAlphaNum(c byte) bool {
	return (c >= '0' && c <= '9') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= 'a' && c <= 'z') ||
		c == '_'
}

func formatLine(line string) string {
	line = strings.TrimSpace(line)

	if !strings.Contains(line, "#") {
		return line
	}
	for i, c := range line {
		if c == '#' {
			return line[:i]
		}
	}

	return line
}

func getLines(input string) ([]string, error) {
	lines := strings.Split(input, "\n")
	var processedLines []string

	for _, line := range lines {
		line = formatLine(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 && strings.TrimSpace(parts[0]) == "source" {
			sourcePath := strings.TrimSpace(parts[1])
			sourcePath, _ = filepath.Abs(sourcePath)

			sourcedContent, err := os.ReadFile(sourcePath)
			if err != nil {
				return nil, err
			}

			sourcedLines, err := getLines(string(sourcedContent))
			if err != nil {
				return nil, err
			}
			processedLines = append(processedLines, sourcedLines...)
		} else {
			processedLines = append(processedLines, line)
		}
	}

	return processedLines, nil
}

func ParseFile(path string) (*Section, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return Parse(string(content))
}

func Parse(input string) (*Section, error) {
	globalSection := &Section{
		Values:    make(map[string][]string),
		Sections:  make(map[string][]*Section),
		Variables: make(map[string]string),
	}

	var currentSection *Section
	var sectionStack []*Section

	lines, err := getLines(input)
	if err != nil {
		return nil, err
	}
	for _, line := range lines {
		line = formatLine(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "$") {
			// Variable
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				varName := strings.TrimPrefix(strings.TrimSpace(parts[0]), "$")
				varValue := strings.TrimSpace(parts[1])
				if currentSection == nil {
					globalSection.Variables[varName] = varValue
				} else {
					currentSection.Variables[varName] = varValue
				}
			}
		} else if strings.HasSuffix(line, "{") {
			// New section
			sectionName := strings.TrimSpace(strings.TrimSuffix(line, "{"))
			newSection := &Section{
				Values:    make(map[string][]string),
				Sections:  make(map[string][]*Section),
				Variables: make(map[string]string),
			}

			if currentSection == nil {
				globalSection.Sections[sectionName] = append(globalSection.Sections[sectionName], newSection)
			} else {
				currentSection.Sections[sectionName] = append(currentSection.Sections[sectionName], newSection)
			}
			sectionStack = append(sectionStack, currentSection)
			currentSection = newSection
		} else if line == "}" {
			// End of section
			if len(sectionStack) > 0 {
				currentSection = sectionStack[len(sectionStack)-1]
				sectionStack = sectionStack[:len(sectionStack)-1]
			}
		} else {
			// Key-value pair
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				if currentSection == nil {
					globalSection.Values[key] = append(globalSection.Values[key], value)
				} else {
					currentSection.Values[key] = append(currentSection.Values[key], value)
				}
			}
		}
	}

	globalSection.substituteVariables(make(map[string]string))

	return globalSection, nil
}

func (section *Section) substituteVariables(parentVariables map[string]string) {
	// Merge parent variables with current section variables
	variables := make(map[string]string)
	for k, v := range parentVariables {
		variables[k] = v
	}
	for k, v := range section.Variables {
		variables[k] = v
	}

	// Replace variables in values
	for k, v := range section.Values {
		for i, value := range v {
			for varName, varValue := range variables {
				wordToReplace := "$" + varName
				index := strings.Index(value, wordToReplace)
				for index != -1 {
					beforeOk := index == 0 || !isAlphaNum(value[index-1])
					afterPos := index + len(wordToReplace)
					afterOk := afterPos == len(value) || !isAlphaNum(value[afterPos])
					if beforeOk && afterOk {
						value = value[:index] + varValue + value[afterPos:]
						index = strings.Index(value, wordToReplace)
					} else {
						nextIndex := strings.Index(value[index+1:], wordToReplace)
						if nextIndex == -1 {
							break
						}
						index += 1 + nextIndex
					}
				}
			}
			section.Values[k][i] = value
		}
	}

	// Replace variables in sub-sections
	for _, v := range section.Sections {
		for _, subSection := range v {
			subSection.substituteVariables(variables)
		}
	}
}

func (section *Section) GetFirst(path string, defaultValue string) string {
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return defaultValue
	}

	if len(parts) == 1 {
		if values, ok := section.Values[parts[0]]; ok && len(values) > 0 {
			return values[0]
		}
		return defaultValue
	}

	subSectionName := parts[0]
	subSectionPath := strings.TrimPrefix(path, subSectionName+"/")
	if subSections, ok := section.Sections[subSectionName]; ok && len(subSections) > 0 {
		for _, subSection := range subSections {
			if value := subSection.GetFirst(subSectionPath, ""); value != "" {
				return value
			}
		}
	}

	return defaultValue
}

func (section *Section) GetAll(path string) []string {
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return []string{}
	}

	if len(parts) == 1 {
		if values, ok := section.Values[parts[0]]; ok {
			return values
		}
		return []string{}
	}

	subSectionName := parts[0]
	subSectionPath := strings.TrimPrefix(path, subSectionName+"/")

	values := []string{}
	if subSections, ok := section.Sections[subSectionName]; ok {
		for _, subSection := range subSections {
			values = append(values, subSection.GetAll(subSectionPath)...)
		}
	}

	return values
}

func SplitValues(value string) []string {
	values := strings.Split(value, ",")
	for i, v := range values {
		values[i] = strings.TrimSpace(v)
	}
	return values
}

func (section *Section) Marshal(indent int) string {
	output := ""
	indentStr := strings.Repeat("  ", indent)
	for k, v := range section.Values {
		for _, value := range v {
			output += indentStr + k + " = " + value + "\n"
		}
	}
	for k, v := range section.Sections {
		for _, subSection := range v {
			output += indentStr + k + " {\n"
			output += subSection.Marshal(indent + 1)
			output += indentStr + "}\n"
		}
	}
	return output
}
