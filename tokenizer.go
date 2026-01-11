package main

import "fmt"

type TokenType int

const (
	TOK_NOT TokenType = iota
	TOK_OR
	TOK_AND
	TOK_TRUE
	TOK_FALSE
	TOK_LPAREN
	TOK_RPAREN
	TOK_VARIABLE
	TOK_EOF
)

type Token struct {
	token_type TokenType
	lexime     string
	position   int
}

type Tokenizer struct {
	str         string
	current_pos int
}

func is_valid_first_char(c byte) bool {
	return ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z') || c == '_'
}

func is_valid_char(c byte) bool {
	return is_valid_first_char(c) || ('0' <= c && c <= '9')
}

func is_whitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n'
}

func (t *Tokenizer) check(s string) (bool, string, error) {
	if t.current_pos+len(s) > len(t.str) {
		return false, "a", fmt.Errorf("Expected token \"%v\" is too long!", s)
	}
	substr := t.str[t.current_pos : t.current_pos+len(s)]
	return substr == s, substr, nil
}

func (t *Tokenizer) consume(l int) {
	t.current_pos += l
}

func (t *Tokenizer) expect(s string) error {
	got, substr, err := t.check(s)
	if err != nil {
		return err
	}
	if !got {
		return fmt.Errorf("Expected string \"%v\". Got \"%v\"!", s, substr)
	}
	t.consume(len(s))
	return nil
}

func (t *Tokenizer) tokenize() []Token {
	tokens := make([]Token, 0)

	for t.current_pos < len(t.str) {
		c := t.str[t.current_pos]
		if c == ' ' || c == '\t' || c == '\n' {
			t.current_pos++
		} else if got, _, _ := t.check("NOT"); got {
			t.expect("NOT")
			tokens = append(tokens, Token{
				token_type: TOK_NOT,
				lexime:     "NOT",
			})
		} else if got, _, _ := t.check("OR"); got {
			t.expect("OR")
			tokens = append(tokens, Token{
				token_type: TOK_OR,
				lexime:     "OR",
			})
		} else if got, _, _ := t.check("AND"); got {
			t.expect("AND")
			tokens = append(tokens, Token{
				token_type: TOK_AND,
				lexime:     "AND",
			})
		} else if c == '1' {
			tokens = append(tokens, Token{
				token_type: TOK_TRUE,
				lexime:     "1",
			})
			t.consume(1)
		} else if c == '0' {
			tokens = append(tokens, Token{
				token_type: TOK_FALSE,
				lexime:     "0",
			})
			t.consume(1)
		} else if c == '(' {
			t.current_pos++
			tokens = append(tokens, Token{
				token_type: TOK_LPAREN,
				lexime:     "(",
			})
		} else if c == ')' {
			t.current_pos++
			tokens = append(tokens, Token{
				token_type: TOK_RPAREN,
				lexime:     ")",
			})
		} else {
			if is_valid_first_char(c) {
				start_pos := t.current_pos
				t.current_pos++
				for t.current_pos < len(t.str) {
					cc := t.str[t.current_pos]
					if !is_valid_char(cc) {
						break
					}
					t.current_pos++
				}
				end_pos := t.current_pos
				tokens = append(tokens, Token{
					token_type: TOK_VARIABLE,
					lexime:     string(t.str[start_pos:end_pos]),
				})
			} else {
				panic(fmt.Errorf("Variable names cannot start with \"%v\"", string(c)))
			}
		}
	}

	tokens = append(tokens, Token{
		token_type: TOK_EOF,
		lexime:     "",
	})

	return tokens
}
