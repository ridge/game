// +build game

package main

import (
	"context"
	"fmt"

	"github.com/ridge/game/mg"
	"github.com/ridge/game/task"
)

// This should work as a default - even if it's in a different file
var Default = ReturnsNilError

// this should not be a target because it returns a string
func ReturnsString() string {
	fmt.Println("more stuff")
	return ""
}

func ReturnsVoid() {
	mg.Deps(f)
}

func f() {}

func TakesContextReturnsVoid(ctx context.Context) {

}

func TakesContextReturnsError(ctx context.Context) error {
	return nil
}

type Ru struct {
}

func (Ru) Run(ctx task.Context) {
}

var varUnexported = Ru{}

var VarWrongType = 42

var VarNoInterface = struct{}{}

var VarTarget = Ru{}
