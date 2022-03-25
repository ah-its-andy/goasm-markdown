package markdown

type Parser struct {
	content []rune
	tokens  []DOM
}

func (p *Parser) Parse() DOM {
	body := &ContainerDOMToken{
		tag:      "body",
		children: make([]DOM, 0),
	}
	tkn := &Tokenizer{
		buf:       p.content,
		isNewLine: true,
	}
	tkn.scanInto(body)
	return body
}
