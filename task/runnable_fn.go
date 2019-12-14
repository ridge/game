package task

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"strings"
)

var (
	namespaceArg = []reflect.Value{reflect.ValueOf(struct{}{})}
	errorType    = reflect.TypeOf((*error)(nil)).Elem()
	contextType  = reflect.TypeOf((*context.Context)(nil)).Elem()
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

func (nf namedFunc) Name() string {
	return nf.name
}

type voidFn struct {
	namedFunc
	fn func()
}

func (vf voidFn) Run(ctx context.Context) error {
	vf.fn()
	return nil
}

type errorFn struct {
	namedFunc
	fn func() error
}

func (ef errorFn) Run(ctx context.Context) error {
	return ef.fn()
}

type contextVoidFn struct {
	namedFunc
	fn func(context.Context)
}

func (cvf contextVoidFn) Run(ctx context.Context) error {
	cvf.fn(ctx)
	return nil
}

type contextErrorFn struct {
	namedFunc
	fn func(context.Context) error
}

func (cef contextErrorFn) Run(ctx context.Context) error {
	return cef.fn(ctx)
}

func errorRet(ret []reflect.Value) error {
	val := ret[0].Interface()
	if val == nil {
		return nil
	}
	return val.(error)
}

type namespaceVoidFn struct {
	namedFunc
	fn reflect.Value
}

func (nvf namespaceVoidFn) Run(ctx context.Context) error {
	nvf.fn.Call(namespaceArg)
	return nil
}

type namespaceErrorFn struct {
	namedFunc
	fn reflect.Value
}

func (nef namespaceErrorFn) Run(ctx context.Context) error {
	return errorRet(nef.fn.Call(namespaceArg))
}

type namespaceContextVoidFn struct {
	namedFunc
	fn reflect.Value
}

func (ncvf namespaceContextVoidFn) Run(ctx context.Context) error {
	ncvf.fn.Call([]reflect.Value{namespaceArg[0], reflect.ValueOf(ctx)})
	return nil
}

type namespaceContextErrorFn struct {
	namedFunc
	fn reflect.Value
}

func (ncef namespaceContextErrorFn) Run(ctx context.Context) error {
	return errorRet(ncef.fn.Call([]reflect.Value{namespaceArg[0], reflect.ValueOf(ctx)}))
}

func boringFunction(f string) bool {
	return f == "" || f == "runtime.Callers" || strings.HasPrefix(f, "github.com/magefile/mage/")
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
	return fmt.Errorf("Invalid type for a task function: %T, must be func(), func() error, func(context.Context), "+
		"func(context.Context) error, or the same method on an mg.Namespace @ %s", fn, causeLocation())
}

// FuncToRunnable converts a function to a Runnable if its signature allows
func FuncToRunnable(module string, fn interface{}) (Runnable, error) {
	if runnable, ok := fn.(Runnable); ok {
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
	case func(context.Context):
		return contextVoidFn{namedFunc{name}, typedFn}, nil
	case func(context.Context) error:
		return contextErrorFn{namedFunc{name}, typedFn}, nil
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

	switch t.NumOut() {
	case 0:
	case 1:
		if t.Out(0) != errorType {
			return nil, invalidTypeError(fn)
		}
	default:
		return nil, invalidTypeError(fn)
	}

	v := reflect.ValueOf(fn)

	switch {
	case t.NumIn() == 1 && t.NumOut() == 0:
		return namespaceVoidFn{namedFunc{name}, v}, nil
	case t.NumIn() == 1 && t.NumOut() == 1:
		return namespaceErrorFn{namedFunc{name}, v}, nil
	case t.NumIn() == 2 && t.NumOut() == 0:
		return namespaceContextVoidFn{namedFunc{name}, v}, nil
	case t.NumIn() == 2 && t.NumOut() == 1:
		return namespaceContextErrorFn{namedFunc{name}, v}, nil
	default:
		return nil, invalidTypeError(fn)
	}
}

// MustFuncToRunnable converts a function to a Runnable
func MustFuncToRunnable(module string, fn interface{}) Runnable {
	runnable, err := FuncToRunnable(module, fn)
	if err != nil {
		panic(err)
	}
	return runnable
}

// MustFuncsToRunnable converts a list of functions to a list of Runnables
func MustFuncsToRunnable(module string, fns []interface{}) []Runnable {
	rr := make([]Runnable, 0, len(fns))
	for _, fn := range fns {
		rr = append(rr, MustFuncToRunnable(module, fn))
	}
	return rr
}
