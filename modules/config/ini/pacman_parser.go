package ini

func NewPacmanParser() *Parser {
	opts := Options{
		AllowInlineComment: true,
	}
	return NewParser(opts)
}
