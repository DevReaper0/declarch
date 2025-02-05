package ini

func NewPacmanParser() *Parser {
	opts := Options{
		AllowInlineComment: true,
		AllowBooleanKeys:   true,
		CommentChar:        "#",
	}
	return NewParser(opts)
}
