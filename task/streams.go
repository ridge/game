package task

import (
	"strings"
	"time"
)

type streamLineSink struct {
	task      *Task
	reporters []Reporter
	stream    Stream

	// current non-\n-terminated line of output
	tailTime time.Time
	tail     string
}

func (sls *streamLineSink) storeLine(t time.Time, line string) {
	logLine := LogLine{Stream: sls.stream, Line: line}
	sls.task.StoreLine(logLine)
	for _, r := range sls.reporters {
		r.OutputLine(sls.task, t, logLine)
	}
}

func (sls *streamLineSink) Add(t time.Time, input string) {
	for _, line := range strings.SplitAfter(input, "\n") {
		if strings.HasSuffix(line, "\n") {
			// full line
			if sls.tail == "" {
				sls.storeLine(t, line)
			} else {
				sls.storeLine(sls.tailTime, sls.tail+line)
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
		sls.storeLine(sls.tailTime, sls.tail+"\n")
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

func newStreamLineWriters(task *Task, reporters []Reporter) (flushWriter, flushWriter) {
	stdoutSink := &streamLineSink{task: task, reporters: reporters, stream: StdoutStream}
	stderrSink := &streamLineSink{task: task, reporters: reporters, stream: StderrStream}

	return streamLineWriter{stdoutSink, stderrSink}, streamLineWriter{stderrSink, stdoutSink}
}
