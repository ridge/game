package toplevel

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"time"
)

type state struct {
	Created     time.Time
	User        string
	Hostname    string
	OS          string
	Frequencies map[string]int
}

func newState() state {
	user := os.Getenv("USER")
	if user == "ubuntu" || user == "tectonic" {
		user = ""
	}
	hostname, _ := os.Hostname()
	return state{
		Created:     time.Now(),
		User:        user,
		Hostname:    hostname,
		OS:          runtime.GOOS,
		Frequencies: map[string]int{},
	}
}

func loadState(stateFile string) state {
	stateBytes, err := ioutil.ReadFile(stateFile)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "unable to load usage state: %v\n", err)
		}
		return newState()
	}

	var out state
	if json.Unmarshal(stateBytes, &out) != nil {
		fmt.Fprintf(os.Stderr, "unable to load usage state: %v\n", err)
		return newState()
	}
	return out
}

func saveState(stateFile string, st state) {
	stateBytes, err := json.Marshal(st)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to save usage state: %v\n", err)
		return
	}
	if err := ioutil.WriteFile(stateFile, stateBytes, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "unable to save usage state: %v\n", err)
	}
}

func updateUsage(stateFile string, targets []string, cmd string, frequency time.Duration) {
	st := loadState(stateFile)
	for _, target := range targets {
		st.Frequencies[target]++
	}
	saveState(stateFile, st)

	if time.Since(st.Created) < frequency {
		return
	}

	if cmd == "" {
		fmt.Fprintf(os.Stderr, "unable to send usage report: no report command\n")
		return
	}
	sendCmd := exec.Command(cmd, stateFile)
	sendCmd.Stdout = os.Stderr
	sendCmd.Stderr = os.Stderr
	if err := sendCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "unable to send usage report: %v\n", err)
		return
	}
	os.Remove(stateFile)
}
