package ini

func NewPacmanParser() *Parser {
	opts := Options{
		AllowInlineComment: true,
		AllowBooleanKeys:   true,
	}
	return NewParser(opts)
}
