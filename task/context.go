package task

import (
	"context"
	"io"
)

// Context is a task context
type Context struct {
	context.Context
}

// Dep runs the given tasks as subtasks of the curent task. Dependencies must
// only be of type func(task.Context) or the method of the same signature on a
// mg.Namespace type.
//
// The task calling Dep is guaranteed that all dependent tasks will be run
// exactly once when Dep returns. Dependent functions may in turn run their own
// dependencies using Dep. Each dependency is run in their own goroutine. Each
// function is given the subtask context.
func (ctx Context) Dep(fns ...interface{}) {
	runSubtasks(ctx, All.Register(fns))
}

// SeqDep is similar to Dep, with the only difference that subtasks will be run
// one after another, not simultaneously.
//
// Same as for Dep, if one of subtasks fails, the rest will be run anyway.
//
// This function is useful in a very limited number of cases, mostly when there
// is a number of checks that have to be performed in order and all erorrs need
// to be conveyed back to user.
func (ctx Context) SeqDep(fns ...interface{}) {
	runSubtasksSequential(ctx, All.Register(fns))
}

// Stdout returns a stdout writer associated with the current task
func (ctx Context) Stdout() io.Writer {
	return Stdout(ctx)
}

// Stderr returns a stdout writer associated with the current task
func (ctx Context) Stderr() io.Writer {
	return Stderr(ctx)
}
