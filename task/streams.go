package task

import (
	"strings"
	"time"
)

type streamLineSink struct {
	task     *Task
	reporter Reporter
	stream   Stream

	// current non-\n-terminated line of output
	tailTime time.Time
	tail     string
}

func (sls *streamLineSink) Add(t time.Time, input string) {
	for _, line := range strings.SplitAfter(input, "\n") {
		if strings.HasSuffix(line, "\n") {
			// full line
			if sls.tail == "" {
				sls.reporter.OutputLine(sls.task, t, sls.stream, line)
			} else {
				sls.reporter.OutputLine(sls.task, sls.tailTime, sls.stream, sls.tail+line)
				sls.tail = ""
			}
		} else {
			// partial line, may only happen once at the end of the loop
			if sls.tail == "" {
				sls.tail = line
				sls.tailTime = t
			} else {
				sls.tail += line
			}
		}
	}
}

func (sls *streamLineSink) Flush(t time.Time) {
	if sls.tail != "" {
		sls.reporter.OutputLine(sls.task, sls.tailTime, sls.stream, sls.tail+"\n")
		sls.tail = ""
	}
}

// streamLineWriter is a Writer for an output stream that flushes any pending
// data from other stream
type streamLineWriter struct {
	sink  *streamLineSink
	other *streamLineSink
}

func (slw streamLineWriter) Write(p []byte) (int, error) {
	t := time.Now()
	slw.other.Flush(t)
	slw.sink.Add(t, string(p))
	return len(p), nil
}

func (slw streamLineWriter) Flush() {
	slw.sink.Flush(time.Now())
}

func newStreamLineWriters(task *Task, reporter Reporter) (flushWriter, flushWriter) {
	stdoutSink := &streamLineSink{task: task, reporter: reporter, stream: StdoutStream}
	stderrSink := &streamLineSink{task: task, reporter: reporter, stream: StderrStream}

	return streamLineWriter{stdoutSink, stderrSink}, streamLineWriter{stderrSink, stdoutSink}
}
