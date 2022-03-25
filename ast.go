package markdown

import (
	"fmt"
	"strings"
)

type VisitFunc func(node DOM) (kontinue bool, err error)

func Walk(visit VisitFunc, tokens ...DOM) error {
	for _, node := range tokens {
		if kontinue, err := visit(node); err != nil {
			return err
		} else if kontinue {
			err = node.walkSubTree(visit)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func RenderDOM(token DOM) (kontinue bool, err error) {
	if _, ok := token.(*ContainerDOMToken); ok {
		return true, nil
	} else if token.Tag() == ":text" {
		token.(*DOMToken).renderedStr = string(token.Token())
		return false, nil
	}
	renderedValues := make([]string, 0)
	renderedValues = append(renderedValues, fmt.Sprintf("<%s", token.Tag()))
	for key, value := range token.Attrs() {
		renderedValues = append(renderedValues, fmt.Sprintf(" %s=\"%s\"", key, value))
	}
	renderedValues = append(renderedValues, ">")
	renderedValues = append(renderedValues, "%s<")
	renderedValues = append(renderedValues, fmt.Sprintf("/%s>", token.Tag()))
	token.(*DOMToken).renderedStr = strings.Join(renderedValues, "")
	return token.Next() != nil, nil
}

func Translate(content []rune) string {
	parser := &Parser{
		content: content,
		tokens:  make([]DOM, 0),
	}
	token := parser.Parse()
	Walk(RenderDOM, token)
	return token.String()
}
