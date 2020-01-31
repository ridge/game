//+build game

package main

import (
	"fmt"

	"github.com/ridge/game/mg"
	"github.com/ridge/game/task"
)

type ParameterizedDep struct {
	i int
}

func (pd ParameterizedDep) Run(ctx task.Context) {
	fmt.Printf("%d\n", pd.i)
}

func Main(ctx task.Context) {
	mg.CtxDeps(ctx,
		ParameterizedDep{1},
		ParameterizedDep{2},
		ParameterizedDep{3},
		ParameterizedDep{4},
		ParameterizedDep{5},
		ParameterizedDep{6},
		ParameterizedDep{1},
		ParameterizedDep{1},
		ParameterizedDep{3},
		ParameterizedDep{6},
		ParameterizedDep{2},
	)
	mg.SerialCtxDeps(ctx,
		ParameterizedDep{1},
		ParameterizedDep{2},
		ParameterizedDep{5},
	)
}

var Default = Main
