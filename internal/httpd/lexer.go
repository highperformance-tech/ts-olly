package httpd

import (
	"fmt"
	lex "github.com/timtadh/lexmachine"
	"github.com/timtadh/lexmachine/machines"
	"strings"
)

func newLexer() (*lexer, error) {
	l := &lexer{}
	l.initTokens()
	l.Lexer = lex.NewLexer()
	for _, lit := range l.Literals {
		r := "\\" + strings.Join(strings.Split(lit, ""), "\\")
		l.Add([]byte(r), l.token(lit))
	}
	for _, name := range l.Keywords {
		l.Add([]byte(name), l.token(name))
	}
	l.Add([]byte(`#([^#][^\n]*)`), l.token("COMMENT"))
	l.Add([]byte("( |\t|\n|\r)+"), skip)
	l.Add([]byte(`([a-z]|[A-Z])([a-z]|[A-Z]|_)*`), l.token("ID"))
	l.Add([]byte(`"`),
		func(scan *lex.Scanner, match *machines.Match) (interface{}, error) {
			str := make([]byte, 0, 10)
			str = append(str, match.Bytes...)
			match.EndLine = match.StartLine
			match.EndColumn = match.StartColumn
			inEscape := false
			for tc := scan.TC; tc < len(scan.Text); tc++ {
				str = append(str, scan.Text[tc])
				match.EndColumn += 1
				if !inEscape && (scan.Text[tc] == '"' || scan.Text[tc] == '\n') {
					match.TC = scan.TC
					scan.TC = tc + 1
					match.Bytes = str
					return l.token("VALUE")(scan, match)
				}
				if scan.Text[tc] == '\n' {
					match.EndLine += 1
				}
				if scan.Text[tc] == '\\' {
					inEscape = !inEscape
				} else {
					inEscape = false
				}
			}
			return nil,
				fmt.Errorf("unclosed string literal starting at byte %d, (line %d, col %d)",
					match.TC, match.StartLine, match.StartColumn)
		},
	)
	l.Add([]byte("."), skip)
	err := l.Compile()
	if err != nil {
		return nil, err
	}
	return l, nil
}

type lexer struct {
	Literals []string
	Keywords []string
	Tokens   []string
	TokenIds map[string]int
	*lex.Lexer
}

func (l *lexer) initTokens() {
	l.Literals = []string{
		"[",
		"]",
		"{",
		"}",
		"=",
		",",
		";",
		":",
		"->",
		"--",
		"^",
		"-",
	}
	l.Keywords = []string{
		"Define",
		"LogFormat",
		"CustomLog",
		"ErrorLog",
	}
	l.Tokens = []string{
		"ID",
		"VALUE",
		"COMMENT",
	}
	l.Tokens = append(l.Tokens, l.Keywords...)
	l.Tokens = append(l.Tokens, l.Literals...)
	l.TokenIds = make(map[string]int)
	for i, token := range l.Tokens {
		l.TokenIds[token] = i
	}
}

func (l *lexer) token(name string) lex.Action {
	return func(s *lex.Scanner, m *machines.Match) (interface{}, error) {
		return s.Token(l.TokenIds[name], string(m.Bytes), m), nil
	}
}

func skip(*lex.Scanner, *machines.Match) (interface{}, error) {
	return nil, nil
}
