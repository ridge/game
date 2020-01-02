package task

import (
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

// Reporter reports events
type Reporter interface {
	Started(t *Task)
	Finished(t *Task)
	Dependencies(dependent *Task, dependees []*Task)
	OutputLine(t *Task, time time.Time, stream Stream, line string)
}

// Registry registers runnables for execution
type Registry struct {
	reporter Reporter
	module   string

	mu     sync.Mutex
	tasks  map[interface{}]*Task
	nextID int
}

// Runnable is a named piece of runnable code
type Runnable interface {
	Run(ctx Context)
}

type identifiable interface {
	Identify() string
}

func identify(r Runnable) interface{} {
	if i, ok := r.(identifiable); ok {
		return i.Identify()
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
				ID:       r.nextID,
				Runnable: runnable,
				reporter: r.reporter,
			}
			r.nextID++
		}
		out = append(out, r.tasks[identity])
	}
	return out
}

// All is a registry of all tasks
var All = Registry{
	tasks: map[interface{}]*Task{},
}

// SetReporter sets the reporter
func SetReporter(reporter Reporter) {
	All.reporter = reporter
}

// SetModule sets the code module
func SetModule(module string) {
	All.module = module
}
