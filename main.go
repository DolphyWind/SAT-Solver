package main

import (
	"bufio"
	"fmt"
	"os"
);

func main() {
	reader := bufio.NewReader(os.Stdin)
    fmt.Print("> ")
    input, err := reader.ReadString('\n')
    if err != nil {
        fmt.Println("Error reading input:", err)
        return
    }

	// TOKENIZE & PARSE
	tokenizer := Tokenizer{
		str: input,
		current_pos: 0,
	}
	tokens := tokenizer.tokenize()
	p := Parser{
		tokens: tokens,
		current_pos: 0,
	}
	expr_tree := p.parse()

	// OPTIMIZE
	has_change := true
	for has_change {
		expr_tree, has_change = expr_tree.optimize_stage1()
		if has_change {
			continue
		}
		expr_tree, has_change = expr_tree.optimize_stage2()
	}

	// COMPILE
	pre_compiled, _ := expr_tree.pre_compile(0)
	compiled, environment := compile_and_build_environment(pre_compiled)

	// SAT SOLVE
	true_env, sat := sat_solve(compiled, environment)
	if sat {
		fmt.Println("Given expression is satisfiable with these values:")
		for var_name := range true_env.env {
			if var_name[0] != '*' {
				fmt.Printf("%v = %v\n", var_name, true_env.get(var_name))
			}
		}
	} else {
		fmt.Println("Given expression is unsatisfiable.")
	}
}
