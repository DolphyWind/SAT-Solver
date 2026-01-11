package main

import "fmt"

var temp_env map[string]bool = map[string]bool{"1": true, "0": false};


type Parser struct {
	tokens []Token
	current_pos int
}

// PARSE
func (p *Parser) expect(t TokenType) {
	tt := p.tokens[p.current_pos].token_type
	if tt == t {
		p.current_pos++
		return
	}
	panic(fmt.Errorf("Expected token type \"%v\". Got \"%v\"!", t, tt))
}

func (p *Parser) is_finished() bool {
	return p.current_pos >= len(p.tokens) || p.tokens[p.current_pos].token_type == TOK_EOF
}

func (p *Parser) parse() Expression {
	return p.parse_expr()
}

func (p *Parser) parse_expr() Expression {
	return p.parse_conj()
}

func (p *Parser) parse_conj() Expression {
	e1 := p.parse_disj()
	if p.is_finished() {
		return e1
	}

	if p.tokens[p.current_pos].token_type == TOK_OR {
		p.current_pos++
		e2 := p.parse_expr()
		return &OrListExpr {
			exprs: []Expression{e1, e2},
		}
	}
	return e1
}

func (p *Parser) parse_disj() Expression {
	e1 := p.parse_other()
	if p.is_finished() {
		return e1
	}

	if p.tokens[p.current_pos].token_type == TOK_AND {
		p.current_pos++
		e2 := p.parse_expr()
		return &AndListExpr {
			exprs: []Expression{e1, e2},
		}
	}
	return e1
}

func (p *Parser) parse_other() Expression {
	t := p.tokens[p.current_pos]
	switch t.token_type {
	case TOK_NOT:
		p.current_pos++
		return &NotExpr{
			expr: p.parse_expr(),
		}
	case TOK_LPAREN:
		p.current_pos++
		expr := p.parse_expr()
		p.expect(TOK_RPAREN)
		return &ParenExpr{
			expr: expr,
		}
	case TOK_VARIABLE, TOK_TRUE, TOK_FALSE:
		p.current_pos++
		temp_env[t.lexime] = (t.token_type == TOK_TRUE)
		return &VarExpr{
			name: t.lexime,
		}
	default:
		panic(fmt.Errorf("Unexpected token type \"%v\"!", t.token_type))
	}
}
