package main

type OperationType int

const (
	OP_NOP = iota
	OP_NOT
	OP_AND
	OP_OR
	OP_RET
)

type Operation struct {
	result_var string
	op_type OperationType
	operands []string
}

type Environment struct {
	env map[string]bool
}

func (environment *Environment) get(name string) bool {
	switch name {
	case "1":
		return true
	case "0":
		return false
	}
	return environment.env[name]
}

func (environment *Environment) set(name string, value bool) {
	if name == "1" || name  == "0" {
		return
	}
	environment.env[name] = value
} 

func compile_and_build_environment(pre_compiled []Operation) ([]Operation, Environment) {
	var compiled []Operation
	var env Environment = Environment{
		env: make(map[string]bool),
	}

	var final_var string
	for _, e := range pre_compiled {
		if e.op_type != OP_NOP {
			 compiled = append(compiled, e)
		} else {
			env.set(e.result_var, false)
		}
		final_var = e.result_var
	}
	compiled = append(compiled, Operation{
		result_var: final_var,
		op_type: OP_RET,
		operands: []string{},
	})
	return compiled, env
}

func execute(ops []Operation, env Environment) bool {
	if len(ops) == 0 {
		panic("Nothing to execute!")
	}

	for _, op := range ops {
		switch op.op_type {
		case OP_NOT:
			env.set(op.result_var, !env.get(op.operands[0]))
		case OP_OR:
			initial := false
			for _, elem := range op.operands {
				initial = (initial || env.get(elem))
			}
			env.set(op.result_var, initial)
		case OP_AND:
			initial := true
			for _, elem := range op.operands {
				initial = (initial && env.get(elem))
			}
			env.set(op.result_var, initial)
		case OP_RET:
			return env.get(op.result_var)
		case OP_NOP:
			panic("Compile your code first!")
		}
	}

	panic("Unreachable")
}

func sat_solve(code []Operation, env Environment) (Environment, bool) {
	keys := make([]string, 0, len(env.env))
    for k := range env.env {
        keys = append(keys, k)
    }

	for i := 0; i < (1 << len(keys)); i++ {
		env_copy := Environment{env: make(map[string]bool)}
		for j, name := range keys {
			env_copy.set(name, (i>>j)&1 == 1)
		}

		if execute(code, env_copy) {
			return env_copy, true
		}
	}
	return env, false
}

