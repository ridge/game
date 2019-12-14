package mg

import (
	"io"

	"github.com/magefile/mage/task"
)

var module string

// SetModule sets the module for build system. The name of the module will be trimmed off package names in output
func SetModule(mod string) {
	module = mod
}

// SerialCtxDeps is like CtxDeps except that it runs dependencies sequentially.
//
// An error in a subtask stops the execution.
func SerialCtxDeps(ctx task.Context, fns ...interface{}) {
	for _, t := range task.All.Register(task.MustFuncsToRunnable(module, fns)) {
		task.RunSubtasks(ctx, []*task.Task{t})
	}
}

// CtxDeps runs the given functions as dependencies of the calling function.
// Dependencies must only be of type:
//     func()
//     func() error
//     func(task.Context)
//     func(task.Context) error
// Or a similar method on a mg.Namespace type.
//
// The function calling Deps is guaranteed that all dependent functions will be
// run exactly once when Deps returns.  Dependent functions may in turn declare
// their own dependencies using Deps. Each dependency is run in their own
// goroutines. Each function is given the context provided if the function
// prototype allows for it.
func CtxDeps(ctx task.Context, fns ...interface{}) {
	task.RunSubtasks(ctx, task.All.Register(task.MustFuncsToRunnable(module, fns)))
}

// Stdout returns a stdout writer associated with the current task
func Stdout(ctx task.Context) io.Writer {
	return task.Stdout(ctx)
}

// Stderr returns a stderr writer associated with the current task
func Stderr(ctx task.Context) io.Writer {
	return task.Stderr(ctx)
}
