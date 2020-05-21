package task

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
)

var (
	namespaceArg = []reflect.Value{reflect.ValueOf(struct{}{})}
	contextType  = reflect.TypeOf((*Context)(nil)).Elem()
)

func funcName(module string, i interface{}) string {
	name := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()

	splitByPackage := strings.Split(name, ".")
	if len(splitByPackage) == 2 && splitByPackage[0] == "main" {
		return splitByPackage[1]
	}
	return strings.TrimPrefix(name, module+"/")
}

type namedFunc struct {
	name string
}

// Functions are not comparable, so we compare them by their names
func (nf namedFunc) Identify() interface{} {
	return nf.name
}

type voidFn struct {
	namedFunc
	fn func()
}

func (vf voidFn) Run(ctx Context) {
	vf.fn()
}

type errorFn struct {
	namedFunc
	fn func() error
}

func (ef errorFn) Run(ctx Context) {
	if err := ef.fn(); err != nil {
		panic(err)
	}
}

type contextVoidFn struct {
	namedFunc
	fn func(Context)
}

func (cvf contextVoidFn) Run(ctx Context) {
	cvf.fn(ctx)
}

type namespaceVoidFn struct {
	namedFunc
	fn reflect.Value
}

func (nvf namespaceVoidFn) Run(ctx Context) {
	nvf.fn.Call(namespaceArg)
}

type namespaceContextVoidFn struct {
	namedFunc
	fn reflect.Value
}

func (ncvf namespaceContextVoidFn) Run(ctx Context) {
	ncvf.fn.Call([]reflect.Value{namespaceArg[0], reflect.ValueOf(ctx)})
}

func boringFunction(f string) bool {
	return f == "" || f == "runtime.Callers" || strings.HasPrefix(f, "github.com/ridge/game/")
}

// Returns a location of mg.Deps invocation where the error originates
func causeLocation() string {
	pcs := make([]uintptr, 10)
	if runtime.Callers(0, pcs) < 1 {
		return "<unknown>"
	}
	frames := runtime.CallersFrames(pcs)
	for {
		frame, more := frames.Next()
		if !boringFunction(frame.Function) {
			return fmt.Sprintf("%s %s:%d", frame.Function, frame.File, frame.Line)
		}
		if !more {
			return "<unknown>"
		}
	}
}

func invalidTypeError(fn interface{}) error {
	return fmt.Errorf("Invalid type for a task function: %T, must be func(), func(Context), "+
		"or the same method on an mg.Namespace @ %s", fn, causeLocation())
}

// funcToRunnable converts a function to a Runnable if its signature allows
func funcToRunnable(module string, fn interface{}) (Runnable, error) {
	if runnable, ok := fn.(Runnable); ok {
		// FIXME (misha): check that the fields are good
		return runnable, nil
	}

	t := reflect.TypeOf(fn)
	if t.Kind() != reflect.Func {
		return nil, invalidTypeError(fn)
	}

	name := funcName(module, fn)

	switch typedFn := fn.(type) {
	case func():
		return voidFn{namedFunc{name}, typedFn}, nil
	case func() error:
		return errorFn{namedFunc{name}, typedFn}, nil
	case func(Context):
		return contextVoidFn{namedFunc{name}, typedFn}, nil
	}

	// mg.Namespace methods

	switch t.NumIn() {
	case 2:
		if t.In(1) != contextType {
			return nil, invalidTypeError(fn)
		}
		fallthrough
	case 1:
		if t.In(0).Kind() != reflect.Struct || t.In(0).NumField() != 0 {
			return nil, invalidTypeError(fn)
		}
	default:
	}

	if t.NumOut() != 0 {
		return nil, invalidTypeError(fn)
	}

	v := reflect.ValueOf(fn)

	switch {
	case t.NumIn() == 1:
		return namespaceVoidFn{namedFunc{name}, v}, nil
	case t.NumIn() == 2:
		return namespaceContextVoidFn{namedFunc{name}, v}, nil
	default:
		return nil, invalidTypeError(fn)
	}
}

// mustFuncsToRunnable converts a list of functions to a list of Runnables
func mustFuncsToRunnable(module string, fns []interface{}) []Runnable {
	rr := make([]Runnable, 0, len(fns))
	for _, fn := range fns {
		runnable, err := funcToRunnable(module, fn)
		if err != nil {
			panic(err)
		}
		rr = append(rr, runnable)
	}
	return rr
}
