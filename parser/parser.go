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

func getLines(input string) []string {
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
			sourcePath, _ = filepath.EvalSymlinks(sourcePath)

			sourceContent, err := os.ReadFile(sourcePath)
			if err != nil {
				panic(err)
			}

			processedLines = append(processedLines, getLines(string(sourceContent))...)
		} else {
			processedLines = append(processedLines, line)
		}
	}

	return processedLines
}

func ParseFile(path string) *Section {
	path, _ = filepath.Abs(path)
	path, _ = filepath.EvalSymlinks(path)
	os.Chdir(filepath.Dir(path))

	content, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	return Parse(string(content))
}

func Parse(input string) *Section {
	globalSection := &Section{
		Values:    make(map[string][]string),
		Sections:  make(map[string][]*Section),
		Variables: make(map[string]string),
	}

	var currentSection *Section
	var sectionStack []*Section

	lines := getLines(input)
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

	return globalSection
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
				value = strings.ReplaceAll(value, "$"+varName, varValue)
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
