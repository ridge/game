//+build game

package main

import (
	"fmt"
	"time"

	"github.com/ridge/game/task"
)

// Returns a non-nil error.
func TakesContextNoError(ctx task.Context) {
	deadline, _ := ctx.Deadline()
	fmt.Printf("Context timeout: %v\n", deadline)
}

func Timeout(ctx task.Context) {
	select {
	case <-time.After(200 * time.Millisecond):
	case <-ctx.Done():
		panic(ctx.Err())
	}
}

func CtxDeps(ctx task.Context) {
	ctx.Dep(TakesContextNoError)
}
