package task

import (
	"reflect"
	"sync"
	"time"
)

// Stream is the standard I/O stream type
type Stream int

const (
	// StdoutStream is a marker of stdout stream
	StdoutStream Stream = 1
	// StderrStream is a marker of stderr stream
	StderrStream Stream = 2
)

// LogLine is struct that unites line to be printed with type of stream (stdout or stderr)
type LogLine struct {
	Stream Stream
	Line   string
}

// Reporter reports events
type Reporter interface {
	Started(t *Task)
	Finished(t *Task)
	Dependencies(dependent *Task, dependees []*Task, sequential bool)
	// OutputLine will send string to any of output streams defined by line.Stream
	// as well as store it into task persistent storage
	// line.Line should include end-of-line character
	OutputLine(t *Task, time time.Time, line LogLine)
}

// Registry registers runnables for execution
type Registry struct {
	reporters []Reporter
	module    string

	mu     sync.Mutex
	tasks  map[interface{}]*Task
	nextID int
}

// Runnable is a named piece of runnable code
type Runnable interface {
	Run(ctx Context)
}

type identifiable interface {
	Identify() interface{}
}

type taskID struct {
	Type reflect.Type
	ID   interface{}
}

func identify(r Runnable) interface{} {
	if i, ok := r.(identifiable); ok {
		return taskID{Type: reflect.TypeOf(r), ID: i.Identify()}
	}
	return r
}

// Register registers runnable as a task
func (r *Registry) Register(fns []interface{}) []*Task {
	runnables := mustFuncsToRunnable(r.module, fns)

	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]*Task, 0, len(runnables))
	for _, runnable := range runnables {
		identity := identify(runnable)

		if _, exists := r.tasks[identity]; !exists {
			r.tasks[identity] = &Task{
				ID:        r.nextID,
				Runnable:  runnable,
				reporters: r.reporters,
			}
			r.nextID++
		}
		out = append(out, r.tasks[identity])
	}
	return out
}

// Tasks returns tasks
func (r *Registry) Tasks() []*Task {
	ts := []*Task{}
	for _, task := range r.tasks {
		ts = append(ts, task)
	}
	return ts
}

// All is a registry of all tasks
var All = Registry{
	tasks: map[interface{}]*Task{},
}

// AddReporter adds the reporter
func AddReporter(reporter Reporter) {
	All.reporters = append(All.reporters, reporter)
}

// SetModule sets the code module
func SetModule(module string) {
	All.module = module
}
