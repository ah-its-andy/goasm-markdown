package markdown

import (
	"fmt"
	"strings"
)

type Tokenizer struct {
	buf       []rune
	isNewLine bool

	Position      int
	ParagraphMode bool

	tokens []DOM
}

func (tkn *Tokenizer) CurrentChar() rune {
	if tkn.Position == -1 {
		return 0
	}
	return tkn.buf[tkn.Position]
}

func (tkn *Tokenizer) Seek(pos int) int {
	if len(tkn.buf) > pos {
		tkn.Position = pos
		return pos
	}
	return -1
}
func (tkn *Tokenizer) Peek() rune {
	if pos := tkn.Next(); pos == -1 {
		return 0
	}
	return tkn.CurrentChar()
}

func (tkn *Tokenizer) Next() int {
	tkn.isNewLine = tkn.isLineBreak()
	pos := tkn.Seek(tkn.Position + 1)
	if pos == -1 {
		tkn.CurrentChar()
		return -1
	}

	return tkn.Position
}
func (tkn *Tokenizer) Reset() {
	tkn.Seek(0)
}

func (tkn *Tokenizer) Scan() *DOMToken {
	if tkn.CurrentChar() == 0 {
		pos := tkn.Next()
		if pos == -1 {
			return nil
		}
	}
	if skipPos := tkn.skipBlank(); skipPos == -1 {
		return nil
	}
	pos := tkn.Position
	tokenStr := ""
	lastCharVal := string([]rune{tkn.CurrentChar()})
	fmt.Print(lastCharVal)
	if tkn.CurrentChar() == rune('#') && tkn.isNewLine {
		lastPos := pos
		keyword := "#"
		for {
			if tkn.Peek() != '#' {
				lastPos = tkn.Position
				break
			}
			keyword += "#"
		}

		tokenStr := keyword + string(tkn.scanToEnd())
		tkn.Seek(lastPos)
		return &DOMToken{
			tag:      fmt.Sprintf("h%d", len(keyword)),
			token:    []rune(tokenStr),
			position: pos,
			length:   len(tokenStr),
		}
	}

	if tkn.CurrentChar() == '*' {
		keyword := "*"
		for {
			if tkn.Peek() != '*' {
				break
			}
			keyword += "*"
		}
		lpos := tkn.Position
		endPos, token := tkn.scanPair(keyword)
		if endPos == -1 {
			tokenStr += keyword
			tkn.Seek(lpos)
		} else {
			tkn.Seek(endPos)
			dToken := &DOMToken{
				tag:      "span",
				token:    token,
				position: pos,
				length:   len(tokenStr),
				attrs:    make(map[string]string),
			}
			if len(keyword) == 1 {
				dToken.attrs["style"] = "font-style: italic;"
			} else if len(keyword) == 2 {
				dToken.attrs["style"] = "font-weight: bold;"
			} else if len(keyword) == 3 {
				dToken.attrs["style"] = "font-style: italic; font-weight: bold;"
			}

			childTokenizer := &Tokenizer{
				buf:       token[len(keyword) : len(token)-len(keyword)],
				isNewLine: false,
			}
			childTokenizer.scanInto(dToken)

			return dToken
		}
	}
	if tkn.CurrentChar() == '`' {
		tkn.Seek(tkn.Position + 2)
		if tkn.CurrentChar() == '`' {
			tkn.ParagraphMode = true

			keyword := "```"
			lpos := tkn.Position
			endPos, token := tkn.scanPair(keyword)
			if endPos == -1 {
				tokenStr += keyword
				tkn.Seek(lpos)
			} else {
				tkn.Seek(endPos)
				return &DOMToken{
					tag:      "p",
					token:    token,
					position: pos,
					length:   len(tokenStr),
					attrs:    map[string]string{"class": "render-code"},
				}
			}
		} else {
			tkn.Seek(tkn.Position - 2)
			keyword := "`"
			lpos := tkn.Position
			endPos, token := tkn.scanPair(keyword)
			if endPos == -1 {
				tkn.Seek(lpos)
				tokenStr += keyword
			} else {
				tkn.Seek(endPos)
				return &DOMToken{
					tag:      "span",
					token:    token,
					position: pos,
					length:   len(tokenStr),
					attrs:    map[string]string{"class": "render-code"},
				}
			}
		}
	}
	if tkn.CurrentChar() == '~' {
		if tkn.Peek() == '~' {
			keyword := "~~"
			lpos := tkn.Position
			endPos, token := tkn.scanPair(keyword)
			if endPos == -1 {
				tokenStr += "~~"
				tkn.Seek(lpos)
			} else {
				tkn.Seek(endPos)
				return &DOMToken{
					tag:      "span",
					token:    token,
					position: pos,
					length:   len(tokenStr),
					attrs:    map[string]string{"style": "text-decoration: line-through;"},
				}
			}
		} else {
			tokenStr += "~~"
		}
	}

	token := []rune(tokenStr + string(tkn.scanToEnd()))
	tkn.Next()

	return &DOMToken{
		tag:      ":text",
		token:    token,
		position: pos,
		length:   len(token),
	}
}

func (tkn *Tokenizer) scanInto(r DOM) {
	if container, ok := r.(*ContainerDOMToken); ok {
		for {
			token := tkn.Scan()
			if token == nil || token.position == -1 {
				break
			}
			if prev := tkn.findPrev(token); prev != nil {
				prev.SetNext(token)
			} else {
				container.children = append(container.children, token)
			}
			tkn.tokens = append(tkn.tokens, token)
		}
	} else if node, ok := r.(*DOMToken); ok {
		for {
			token := tkn.Scan()
			if token == nil || token.position == -1 {
				break
			}
			node.SetNext(token)
			node = token
			tkn.tokens = append(tkn.tokens, token)
		}
	}
}

func (tkn *Tokenizer) findPrev(token DOM) DOM {
	for _, t := range tkn.tokens {
		if t.Position() < token.Position() && t.Length()+t.Position() >= token.Length()+token.Position() {
			return t
		}
	}
	return nil
}

func (tkn *Tokenizer) scanPair(keyword string) (endPos int, token []rune) {
	curPos := tkn.Position
	if tkn.ParagraphMode {
		paragraph := keyword
		endPos = -1
		for {
			line := string(tkn.scanToEnd())
			paragraph += string(tkn.scanToEnd())
			endPos = strings.Index(line, keyword)
			if endPos != -1 {
				break
			}
		}
		tkn.ParagraphMode = false
		return curPos + endPos + len(keyword), []rune(paragraph)
	} else {
		line := string(tkn.scanToEnd())
		endPos := strings.Index(line, keyword)
		if endPos == -1 {
			return -1, []rune(keyword + line)
		} else {
			return curPos + endPos + len(keyword), []rune(keyword + line[:endPos+len(keyword)])
		}
	}
}

func (tkn *Tokenizer) skipBlank() int {
	ch := tkn.CurrentChar()
	pos := tkn.Position
	for ch == ' ' || ch == '\n' || ch == '\r' || ch == '\t' {
		if pos = tkn.Next(); pos == -1 {
			break
		}
		ch = tkn.CurrentChar()
	}
	return pos
}

func (tkn *Tokenizer) isWhiteSpace() bool {
	return tkn.CurrentChar() == ' '
}

func (tkn *Tokenizer) isLineBreak() bool {
	lastCharVal := string([]rune{tkn.CurrentChar()})
	fmt.Print(lastCharVal)
	flag := tkn.CurrentChar() == '\\'
	if flag {
		flag = tkn.Peek() == 'n'
		if !flag {
			tkn.Seek(tkn.Position - 1)
		}
	}
	if !flag {
		flag = tkn.CurrentChar() == '\n'
	}
	return flag
}

func (tkn *Tokenizer) scanToEnd() []rune {
	ret := make([]rune, 0)
	for {
		if tkn.isLineBreak() {
			break
		}
		ret = append(ret, tkn.CurrentChar())
		if pos := tkn.Next(); pos == -1 {
			break
		}
	}
	return ret
}

type DOM interface {
	Tag() string
	Token() []rune
	Position() int
	Length() int
	Attrs() map[string]string
	Prev() DOM
	Next() DOM
	SetPrev(DOM)
	SetNext(DOM)
	IsLeaf() bool
	String() string

	walkSubTree(visit VisitFunc) error
}

type DOMToken struct {
	tag   string
	attrs map[string]string

	token    []rune
	position int
	length   int

	prev DOM
	next DOM

	renderedStr string
}

func (t *DOMToken) String() string {
	if t.IsLeaf() {
		return t.renderedStr
	}
	return fmt.Sprintf(t.renderedStr, t.Next().String())
}

func (t *DOMToken) walkSubTree(visit VisitFunc) error {
	if t.Next() == nil {
		return nil
	}
	kontinue, err := visit(t.Next())
	if err != nil {
		return err
	}
	if kontinue {
		return t.Next().walkSubTree(visit)
	}
	return nil
}

func (t *DOMToken) Tag() string {
	return t.tag
}
func (t *DOMToken) Token() []rune {
	return t.token
}
func (t *DOMToken) Position() int {
	return t.position
}
func (t *DOMToken) Length() int {
	return t.length
}
func (t *DOMToken) Attrs() map[string]string {
	return t.attrs
}
func (t *DOMToken) Prev() DOM {
	return t.prev
}
func (t *DOMToken) Next() DOM {
	return t.next
}
func (t *DOMToken) SetPrev(v DOM) {
	t.prev = v
}
func (t *DOMToken) SetNext(v DOM) {
	t.next = v
}

func (t *DOMToken) IsLeaf() bool {
	return t.next == nil
}

type ContainerDOMToken struct {
	tag   string
	attrs map[string]string

	token    []rune
	position int
	length   int

	prev DOM

	children []DOM

	renderedStr string
}

func (t *ContainerDOMToken) String() string {
	buffer := make([]string, len(t.children))
	for i, child := range t.children {
		buffer[i] = child.String()
	}
	return strings.Join(buffer, "\n")
}

func (t *ContainerDOMToken) walkSubTree(visit VisitFunc) error {
	if len(t.children) == 0 {
		return nil
	}

	for _, child := range t.children {
		kontinue, err := visit(child)
		if err != nil {
			return err
		}
		if kontinue {
			err = child.walkSubTree(visit)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (t *ContainerDOMToken) Tag() string {
	return t.tag
}
func (t *ContainerDOMToken) Token() []rune {
	return t.token
}
func (t *ContainerDOMToken) Position() int {
	return t.position
}
func (t *ContainerDOMToken) Length() int {
	return t.length
}
func (t *ContainerDOMToken) Attrs() map[string]string {
	return t.attrs
}
func (t *ContainerDOMToken) Prev() DOM {
	return t.prev
}
func (t *ContainerDOMToken) Next() DOM {
	return nil
}
func (t *ContainerDOMToken) SetPrev(v DOM) {
	t.prev = v
}
func (t *ContainerDOMToken) SetNext(v DOM) {
}
func (t *ContainerDOMToken) IsLeaf() bool {
	return true
}

type DOMLinkedList struct {
	first DOM
	last  DOM
}

func (list *DOMLinkedList) AddLast(token DOM) {
	if list.first == nil {
		list.first = token
		list.last = token
	} else {
		list.last.SetNext(token)
		token.SetPrev(list.last)
		list.last = token
	}
}

func (list *DOMLinkedList) AddFirst(token DOM) {
	if list.first == nil {
		list.first = token
		list.last = token
	} else {
		list.first.SetPrev(token)
		token.SetNext(list.first)
		list.first = token
	}
}

func (list *DOMLinkedList) AddBefore(token DOM, before DOM) {
	if before == list.first {
		list.AddFirst(token)
	} else {
		before.Prev().SetNext(token)
		token.SetPrev(before.Prev())
		before.SetPrev(token)
		token.SetNext(before)
	}
}

func (list *DOMLinkedList) AddAfter(token DOM, after DOM) {
	if after == list.last {
		list.AddLast(token)
	} else {
		after.Next().SetPrev(token)
		token.SetNext(after.Next())
		after.SetNext(token)
		token.SetPrev(after)
	}
}

func (list *DOMLinkedList) Remove(token DOM) {
	if token == list.first {
		list.first = token.Next()
	} else if token == list.last {
		list.last = token.Prev()
	} else {
		token.Prev().SetNext(token.Next())
		token.Next().SetPrev(token.Prev())
	}
}
