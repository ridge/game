package mg

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/magefile/mage/task"
)

var module string

// SetModule sets the module for build system. The name of the module will be trimmed off package names in output
func SetModule(mod string) {
	module = mod
}

var logger = log.New(os.Stderr, "", 0)

type taskMap struct {
	mu     *sync.Mutex
	nextID int
	m      map[string]*mgtask
}

func (tm *taskMap) Register(d dep) *mgtask {
	name := d.Identify()

	tm.mu.Lock()
	defer tm.mu.Unlock()

	if _, ok := tm.m[name]; !ok {
		tm.m[name] = &mgtask{
			id:          tm.nextID,
			fn:          d.Run,
			displayName: displayName(name),
		}
		tm.nextID++
	}
	return tm.m[name]
}

var tasks = &taskMap{
	mu: &sync.Mutex{},
	m:  map[string]*mgtask{},
}

type contextKey string

const (
	taskContextKey = contextKey("mage.task")
)

// Stdout returns a local stdout stream if assigned to the context, or os.Stdout otherwise
func Stdout(ctx task.Context) io.Writer {
	val := ctx.Value(taskContextKey)
	if val == nil {
		return os.Stdout
	}
	return val.(*mgtask).stdout
}

// Stderr returns a local stderr stream if assigned to the context, or os.Stderr otherwise
func Stderr(ctx task.Context) io.Writer {
	val := ctx.Value(taskContextKey)
	if val == nil {
		return os.Stderr
	}
	return val.(*mgtask).stderr
}

func ctxTask(ctx task.Context) *mgtask {
	val := ctx.Value(taskContextKey)
	if val == nil {
		return nil
	}
	return val.(*mgtask)
}

// SerialDeps is like Deps except it runs each dependency serially, instead of
// in parallel. This can be useful for resource intensive dependencies that
// shouldn't be run at the same time.
func SerialDeps(fns ...interface{}) {
	deps := wrapFns(fns)
	ctx := context.Background()
	for i := range deps {
		runDeps(ctx, deps[i:i+1])
	}
}

// SerialCtxDeps is like CtxDeps except it runs each dependency serially,
// instead of in parallel. This can be useful for resource intensive
// dependencies that shouldn't be run at the same time.
func SerialCtxDeps(ctx task.Context, fns ...interface{}) {
	deps := wrapFns(fns)
	for i := range deps {
		runDeps(ctx, deps[i:i+1])
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
	deps := wrapFns(fns)
	runDeps(ctx, deps)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func plural(name string, count int) string {
	if count == 1 {
		return name
	}

	return name + "s"
}

const shortTaskThreshold = 10 * time.Millisecond

// runDeps assumes you've already called wrapFns.
func runDeps(ctx task.Context, deps []dep) {
	finishedCh := make(chan *mgtask, len(deps))

	for _, dep := range deps {
		t := tasks.Register(dep)

		go func() {
			t.stdout, t.stderr = newStreamLineWriters(taskOutputCollector{task: t})
			if err := t.run(context.WithValue(ctx, taskContextKey, t)); err != nil {
				t.stderr.Flush()
				fmt.Fprintf(t.stderr, "FAILURE | %v\n", err)
			}
			finishedCh <- t
		}()
	}

	t := ctxTask(ctx)
	start := time.Now()

	failedSubtasks := []*mgtask{}
	cumulativeExitStatus := 1
	for i := 0; i < len(deps); i++ {
		subt := <-finishedCh
		if subt.err != nil {
			subtaskExitStatus := 1
			if err, ok := subt.err.(exitStatus); ok {
				subtaskExitStatus = err.ExitStatus()
			}
			failedSubtasks = append(failedSubtasks, subt)
			cumulativeExitStatus = max(cumulativeExitStatus, subtaskExitStatus)
		}
	}

	if t != nil {
		t.subtasksDuration += time.Since(start)
	}

	if len(failedSubtasks) > 0 {
		sort.Slice(failedSubtasks, func(i, j int) bool {
			return failedSubtasks[i].id < failedSubtasks[j].id
		})
		msgs := []string{}
		for _, subtask := range failedSubtasks {
			msgs = append(msgs, subtask.String())
		}
		panic(Fatal(cumulativeExitStatus, fmt.Sprintf("Failed %s: %s", plural("subtask", len(failedSubtasks)),
			strings.Join(msgs, ", "))))
	}
}

func wrapFns(fns []interface{}) []dep {
	deps := make([]dep, len(fns))
	for i, f := range fns {
		d, err := wrapFn(f)
		if err != nil {
			panic(err)
		}
		deps[i] = d
	}
	return deps
}

// Deps runs the given functions in parallel, exactly once. Dependencies must
// only be of type:
//     func()
//     func() error
//     func(task.Context)
//     func(task.Context) error
// Or a similar method on a mg.Namespace type.
//
// This is a way to build up a tree of dependencies with each dependency
// defining its own dependencies.  Functions must have the same signature as a
// Mage target, i.e. optional context argument, optional error return.
func Deps(fns ...interface{}) {
	CtxDeps(context.Background(), fns...)
}

type dep interface {
	Identify() string
	Run(task.Context) error
}

type voidFn func()

func (vf voidFn) Identify() string {
	return name(vf)
}

func (vf voidFn) Run(ctx task.Context) error {
	vf()
	return nil
}

type errorFn func() error

func (ef errorFn) Identify() string {
	return name(ef)
}

func (ef errorFn) Run(ctx task.Context) error {
	return ef()
}

type contextVoidFn func(task.Context)

func (cvf contextVoidFn) Identify() string {
	return name(cvf)
}

func (cvf contextVoidFn) Run(ctx task.Context) error {
	cvf(ctx)
	return nil
}

type contextErrorFn func(task.Context) error

func (cef contextErrorFn) Identify() string {
	return name(cef)
}

func (cef contextErrorFn) Run(ctx task.Context) error {
	return cef(ctx)
}

var namespaceArg = []reflect.Value{reflect.ValueOf(struct{}{})}

func errorRet(ret []reflect.Value) error {
	val := ret[0].Interface()
	if val == nil {
		return nil
	}
	return val.(error)
}

type namespaceVoidFn struct {
	fn interface{}
}

func (nvf namespaceVoidFn) Identify() string {
	return name(nvf.fn)
}

func (nvf namespaceVoidFn) Run(ctx task.Context) error {
	v := reflect.ValueOf(nvf.fn)
	v.Call(namespaceArg)
	return nil
}

type namespaceErrorFn struct {
	fn interface{}
}

func (nef namespaceErrorFn) Identify() string {
	return name(nef.fn)
}

func (nef namespaceErrorFn) Run(ctx task.Context) error {
	v := reflect.ValueOf(nef.fn)
	return errorRet(v.Call(namespaceArg))
}

type namespaceContextVoidFn struct {
	fn interface{}
}

func (ncvf namespaceContextVoidFn) Identify() string {
	return name(ncvf.fn)
}

func (ncvf namespaceContextVoidFn) Run(ctx task.Context) error {
	v := reflect.ValueOf(ncvf.fn)
	v.Call(append(namespaceArg, reflect.ValueOf(ctx)))
	return nil
}

type namespaceContextErrorFn struct {
	fn interface{}
}

func (ncef namespaceContextErrorFn) Identify() string {
	return name(ncef.fn)
}

func (ncef namespaceContextErrorFn) Run(ctx task.Context) error {
	v := reflect.ValueOf(ncef.fn)
	return errorRet(v.Call(append(namespaceArg, reflect.ValueOf(ctx))))
}

func name(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

func displayName(name string) string {
	splitByPackage := strings.Split(name, ".")
	if len(splitByPackage) == 2 && splitByPackage[0] == "main" {
		return splitByPackage[len(splitByPackage)-1]
	}
	return strings.TrimPrefix(name, module+"/")
}

type mgtask struct {
	id int

	once sync.Once
	fn   func(task.Context) error

	stdout           flushWriter
	stderr           flushWriter
	err              error
	duration         time.Duration
	subtasksDuration time.Duration

	displayName string
}

func (t *mgtask) String() string {
	return fmt.Sprintf("#%04d %s", t.id, t.displayName)
}

const smallTaskThreshold = 10 * time.Millisecond

func (t *mgtask) run(ctx task.Context) error {
	t.once.Do(func() {
		start := time.Now()
		defer func() {
			t.duration = time.Since(start)
			if t.duration > smallTaskThreshold {
				t.stdout.Flush()
				fmt.Fprintf(t.stdout, "FINISHED | time=%.02fs, self=%.02fs, subtasks=%.02fs\n",
					t.duration.Seconds(), (t.duration - t.subtasksDuration).Seconds(),
					t.subtasksDuration.Seconds())
			}
		}()

		defer func() {
			if v := recover(); v != nil {
				if err, ok := v.(error); ok {
					t.err = err
				} else {
					t.err = fmt.Errorf("%v", v)
				}
			}
		}()

		if Verbose() {
			logger.Println("Running dependency:", t.displayName)
		}
		t.err = t.fn(ctx)
	})
	return t.err
}

// Returns a location of mg.Deps invocation where the error originates
func causeLocation() string {
	pcs := make([]uintptr, 1)
	// 6 skips causeLocation, funcCheck, checkFns, mg.CtxDeps, mg.Deps in stacktrace
	if runtime.Callers(6, pcs) != 1 {
		return "<unknown>"
	}
	frames := runtime.CallersFrames(pcs)
	frame, _ := frames.Next()
	if frame.Function == "" && frame.File == "" && frame.Line == 0 {
		return "<unknown>"
	}
	return fmt.Sprintf("%s %s:%d", frame.Function, frame.File, frame.Line)
}

// wrapFn tests if a function is one of funcType and wraps in into a corresponding dep type
func wrapFn(fn interface{}) (dep, error) {
	if d, ok := fn.(dep); ok {
		return d, nil
	}

	switch typedFn := fn.(type) {
	case func():
		return voidFn(typedFn), nil
	case func() error:
		return errorFn(typedFn), nil
	case func(task.Context):
		return contextVoidFn(typedFn), nil
	case func(task.Context) error:
		return contextErrorFn(typedFn), nil
	}

	err := fmt.Errorf("Invalid type for dependent function: %T. Dependencies must be func(), func() error, func(task.Context), func(task.Context) error, or the same method on an mg.Namespace @ %s", fn, causeLocation())

	// ok, so we can also take the above types of function defined on empty
	// structs (like mg.Namespace). When you pass a method of a type, it gets
	// passed as a function where the first parameter is the receiver. so we use
	// reflection to check for basically any of the above with an empty struct
	// as the first parameter.

	t := reflect.TypeOf(fn)
	if t.Kind() != reflect.Func {
		return nil, err
	}

	if t.NumOut() > 1 {
		return nil, err
	}
	if t.NumOut() == 1 && t.Out(0) == reflect.TypeOf(err) {
		return nil, err
	}

	// 1 or 2 argumments, either just the struct, or struct and context.
	if t.NumIn() == 0 || t.NumIn() > 2 {
		return nil, err
	}

	// first argument has to be an empty struct
	arg := t.In(0)
	if arg.Kind() != reflect.Struct {
		return nil, err
	}
	if arg.NumField() != 0 {
		return nil, err
	}
	if t.NumIn() == 1 {
		if t.NumOut() == 0 {
			return namespaceVoidFn{fn}, nil
		}
		return namespaceErrorFn{fn}, nil
	}
	ctxType := reflect.TypeOf(context.Background())
	if t.In(1) == ctxType {
		return nil, err
	}

	if t.NumOut() == 0 {
		return namespaceContextVoidFn{fn}, nil
	}
	return namespaceContextErrorFn{fn}, nil
}
