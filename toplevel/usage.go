package toplevel

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"time"
)

type UsageConfig struct {
	StateFile string
	Process   func(UsageState) error
	Interval  time.Duration
}

type UsageState struct {
	Created     time.Time
	User        string
	Hostname    string
	OS          string
	Frequencies map[string]int
}

func newState() UsageState {
	user := os.Getenv("USER")
	if user == "ubuntu" || user == "tectonic" {
		user = ""
	}
	hostname, _ := os.Hostname()
	return UsageState{
		Created:     time.Now(),
		User:        user,
		Hostname:    hostname,
		OS:          runtime.GOOS,
		Frequencies: map[string]int{},
	}
}

func loadState(stateFile string) UsageState {
	stateBytes, err := ioutil.ReadFile(stateFile)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "unable to load usage state: %v\n", err)
		}
		return newState()
	}

	var out UsageState
	if json.Unmarshal(stateBytes, &out) != nil {
		fmt.Fprintf(os.Stderr, "unable to load usage state: %v\n", err)
		return newState()
	}
	return out
}

func saveState(stateFile string, st UsageState) {
	stateBytes, err := json.Marshal(st)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to save usage state: %v\n", err)
		return
	}
	if err := ioutil.WriteFile(stateFile, stateBytes, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "unable to save usage state: %v\n", err)
	}
}

func processUsage(c UsageConfig, targets []string) {
	st := loadState(c.StateFile)
	for _, target := range targets {
		st.Frequencies[target]++
	}
	saveState(c.StateFile, st)

	if c.Interval == 0 {
		c.Interval = 24 * time.Hour
	}

	if time.Since(st.Created) < c.Interval {
		return
	}

	if err := c.Process(st); err != nil {
		fmt.Fprintf(os.Stderr, "unable to send usage report: %v\n", err)
		return
	}
	os.Remove(c.StateFile)
}
