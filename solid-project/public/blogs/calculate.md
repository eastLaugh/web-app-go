# 计算器

表达式是多项式或单项式。

expr   := term { ('+' | '-') term }*

term   := factor { ('*' | '/') factor }*

factor := number | '(' expr ')'

```
expr: (1+(4+5+2)-3)+(6+8)
│
├─ term1: (1+(4+5+2)-3)  [factor: 括号表达式]
│  │
│  └─ expr: 1+(4+5+2)-3
│     │
│     ├─ term1: 1  [factor: 数字]
│     │
│     ├─ term2: (4+5+2)  [factor: 括号表达式]
│     │  │
│     │  └─ expr: 4+5+2
│     │     │
│     │     ├─ term1: 4  [factor: 数字]
│     │     ├─ term2: 5  [factor: 数字]
│     │     └─ term3: 2  [factor: 数字]
│     │
│     └─ term3: 3  [factor: 数字]
│
└─ term2: (6+8)  [factor: 括号表达式]
   │
   └─ expr: 6+8
      │
      ├─ term1: 6  [factor: 数字]
      └─ term2: 8  [factor: 数字]
```

```go
import (
	"unicode"

)

type parser struct {
	input string
	pos   int
}

func (p *parser) expr() int {
	A := p.term()
	p.skipSpace()
	for p.pos < len(p.input) && p.input[p.pos] != ')' {
		sign := p.input[p.pos]
		p.pos++
		switch sign {
		case '+':
			A += p.term()
		case '-':
			A -= p.term()
		}
        p.skipSpace()
	}
    return A
}

func (p *parser) term() (val int) {
	p.skipSpace()
	if p.input[p.pos] == '(' {
		p.pos++
		val = p.expr()
        p.pos++
        return
	}

	for p.pos < len(p.input) && unicode.IsDigit(rune(p.input[p.pos])) {
		ch := p.input[p.pos]
		val = val*10 + (int(ch) - '0')
		p.pos++
	}
	return
}

func (p *parser) skipSpace() {
	for p.pos < len(p.input) && unicode.IsSpace(rune(p.input[p.pos])) {
		p.pos++
	}
}

func calculate(s string) int {
	p := parser{
		input: s,
		pos:   0,
	}
	return p.expr()
}

```