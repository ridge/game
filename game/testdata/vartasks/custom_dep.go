//+build game

package main

import (
	"fmt"

	"github.com/ridge/game/task"
)

type ParameterizedDep struct {
	i int
}

func (pd ParameterizedDep) Run(ctx task.Context) {
	fmt.Printf("%d\n", pd.i)
}

// Dep var dep
var Dep = ParameterizedDep{0}

// noDep
var noDep = ParameterizedDep{1}

var alsoNoDep = 42
