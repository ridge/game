package mg

import (
	"io"

	"github.com/ridge/game/task"
)

// SerialCtxDeps is a backward-compatible way to say ctx.Dep(fn1); ctx.Dep(fn2)...
func SerialCtxDeps(ctx task.Context, fns ...interface{}) {
	for _, fn := range fns {
		ctx.Dep(fn)
	}
}

// CtxDeps is a backward-compatibile way to say ctx.Dep(fns...).
func CtxDeps(ctx task.Context, fns ...interface{}) {
	ctx.Dep(fns...)
}

// Stdout is a backward-compatible way to say ctx.Stdout()
func Stdout(ctx task.Context) io.Writer {
	return ctx.Stdout()
}

// Stderr is a backward-compatible way to say ctx.Stderr()
func Stderr(ctx task.Context) io.Writer {
	return ctx.Stderr()
}
