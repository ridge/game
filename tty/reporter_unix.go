//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package tty

import (
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ridge/game/task"
	"golang.org/x/sys/unix"
)

// From https://en.wikipedia.org/wiki/ANSI_escape_code#Description
const (
	clearToEndOfLine = "\x1b[K"
	red              = "\x1b[31m"
	gray             = "\x1b[37m"
	blue             = "\x1b[34m"
	defColor         = "\x1b[0m"
)

type Reporter struct {
	mu         sync.Mutex
	termCols   int
	unfinished map[int]string
	deps       depSet
}

// Tasks line format:
//
// <gray>... #1234 #4565</gray> <blue>#6666 #7777 #8888 #9999 doing.Stuff</blue>
//       ^                          ^                     ^
//       |                          |                     |
// currently blocked tasks      running tasks     as many task names as fits into the line
//
// If the terminal is too narrow to display all unfinished tasks, then
// first blocked tasks are clipped, then running tasks' names are
// removed and finally running tasks are clipped.

func (r *Reporter) drawTasksLine() {
	// Calculate blocked tasks
	blockedSet := r.deps.blocked()

	var blocked []int
	for id := range blockedSet {
		blocked = append(blocked, id)
	}
	sort.Ints(blocked)

	// Calculate running tasks
	var running []int
	for id := range r.unfinished {
		if blockedSet[id] {
			continue
		}
		running = append(running, id)
	}
	sort.Ints(running)

	// Keep the last column of the terminal free, or the cursor
	// will jump to the next line and won't be clearable until
	// this code learns to use cursor navigation commands.
	maxSize := r.termCols - 1

	runningStr := formatRunningTasks(maxSize, running, r.unfinished)
	blockedStr := formatBlockedTasks(maxSize-len(runningStr), blocked)

	// Clear the existing tasks line, draw it, move cursor back
	fmt.Printf("%s%s%s%s%s%s\r", clearToEndOfLine, gray, blockedStr, blue, runningStr, defColor)
}

func formatRunningTasks(maxSize int, running []int, allTasks map[int]string) string {
	// Deal with awkward conditions first to avoid doing extra checks below
	if len(running) == 0 {
		return ""
	}
	if maxSize < 3 {
		return "..."[:maxSize]
	}

	// Format all tasks
	// IDs and texts separate, as we're going to shorten them separately)

	var runningIDs []string
	var runningTexts []string
	totalSize := 0
	for _, id := range running {
		s := allTasks[id]

		runningID := fmt.Sprintf(" #%04d", id)
		if len(runningIDs) == 0 {
			runningID = runningID[1:] // no space before first task
		}
		runningIDs = append(runningIDs, runningID)
		runningTexts = append(runningTexts, " "+s)
		totalSize += 6 + len(s)
	}

	// If the total length is larger than the terminal width, remove task names one by one
	for i := 0; i < len(running) && totalSize > maxSize; i++ {
		totalSize -= len(runningTexts[i])
		runningTexts[i] = ""
	}

	line := ""
	for i := 0; i < len(runningIDs); i++ {
		line += runningIDs[i]
		line += runningTexts[i]
	}

	// We still can't put all running tasks into the line, then just cut from the beginning
	if len(line) > maxSize {
		return "..." + line[len(line)-maxSize+3:]
	}

	return line
}

func formatBlockedTasks(maxSize int, blocked []int) string {
	// Deal with awkward conditions first to avoid doing extra checks below
	if len(blocked) == 0 {
		return ""
	}
	if maxSize < 3 {
		return "..."[:maxSize]
	}

	line := ""
	for _, id := range blocked {
		line += fmt.Sprintf("#%04d ", id)
	}

	// If we can't fit the line, then just cut from the beginning
	if len(line) > maxSize {
		return "..." + line[len(line)-maxSize+3:]
	}

	return line
}

func (r *Reporter) Started(t *task.Task) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.unfinished[t.ID] = t.ShortName()

	r.drawTasksLine()
}

func (r *Reporter) Dependencies(dependent *task.Task, dependees []*task.Task, sequential bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, dependee := range dependees {
		r.deps.add(dependent.ID, dependee.ID)
	}

	r.drawTasksLine()
}

func (r *Reporter) Finished(t *task.Task) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.deps.unblock(t.ID)
	delete(r.unfinished, t.ID)

	r.drawTasksLine()
}

func formatLine(line task.LogLine) string {
	if line.Stream == task.StderrStream {
		return red + strings.TrimRight(line.Line, "\n") + defColor + "\n"
	}
	return line.Line
}

func (r *Reporter) OutputLine(t *task.Task, time time.Time, line task.LogLine) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Clear the tasks line before drawing the output
	fmt.Print(clearToEndOfLine)
	fmt.Printf("%s %s", t.StringID(), formatLine(line))

	r.drawTasksLine()
}

func (r *Reporter) handleTermWidthChange() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if cols, err := termWidth(); err == nil {
		r.termCols = cols
	}
}

func termWidth() (int, error) {
	ws, err := unix.IoctlGetWinsize(unix.Stdout, unix.TIOCGWINSZ)
	if err != nil {
		return -1, err
	}
	return int(ws.Col), nil
}

func NewReporter() (*Reporter, error) {
	winszCh := make(chan os.Signal, 1)
	// signal.Notify before TIOCGWINSZ to avoid missing a resize on startup
	signal.Notify(winszCh, syscall.SIGWINCH)

	cols, err := termWidth()
	if err != nil {
		return nil, err
	}
	r := &Reporter{
		termCols:   cols,
		unfinished: map[int]string{},
	}
	go func() {
		for range winszCh {
			r.handleTermWidthChange()
		}
	}()
	return r, nil
}
