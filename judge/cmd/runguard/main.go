package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// Metrics contains precise resource usage measurements from getrusage.
type Metrics struct {
	// WallTime in seconds (float64 for sub-second precision)
	WallTime float64 `json:"wall_time"`
	// CPUTime is user + system CPU time in seconds (from getrusage)
	CPUTime float64 `json:"cpu_time"`
	// UserTime in seconds (from getrusage)
	UserTime float64 `json:"user_time"`
	// SystemTime in seconds (from getrusage)
	SystemTime float64 `json:"sys_time"`
	// MaxRSS in kilobytes (from getrusage ru_maxrss, converted from bytes on Linux)
	MaxRSS int64 `json:"max_rss_kb"`
	// ExitCode of the child process
	ExitCode int `json:"exit_code"`
	// Signal that killed the process (0 if exited normally)
	Signal int `json:"signal"`
	// Verdict: "ok", "time-limit", "memory-limit", "runtime-error"
	Verdict string `json:"verdict"`
}

func main() {
	var (
		timeLimit   float64
		memoryLimit int64
		outputFile  string
	)

	flag.Float64Var(&timeLimit, "t", 0, "CPU time limit in seconds (0 = no limit)")
	flag.Int64Var(&memoryLimit, "m", 0, "Memory limit in KB (0 = no limit)")
	flag.StringVar(&outputFile, "o", "/workspace/.metrics.json", "Output file for metrics JSON")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "runguard: no command specified\n")
		os.Exit(1)
	}

	// Set RLIMIT_CPU if time limit specified (ceil to integer seconds + 1s grace)
	if timeLimit > 0 {
		cpuSeconds := uint64(timeLimit) + 1
		hardLimit := cpuSeconds + 1 // hard limit 1s beyond soft for SIGKILL
		err := syscall.Setrlimit(syscall.RLIMIT_CPU, &syscall.Rlimit{
			Cur: cpuSeconds,
			Max: hardLimit,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "runguard: failed to set RLIMIT_CPU: %v\n", err)
		}
	}

	// Set RLIMIT_AS if memory limit specified (convert KB to bytes, with some overhead)
	if memoryLimit > 0 {
		memBytes := uint64(memoryLimit) * 1024
		// Add 16MB overhead for runtime/stack
		memBytes += 16 * 1024 * 1024
		err := syscall.Setrlimit(syscall.RLIMIT_AS, &syscall.Rlimit{
			Cur: memBytes,
			Max: memBytes,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "runguard: failed to set RLIMIT_AS: %v\n", err)
		}
	}

	// Build command
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Wall-clock timer with generous margin (2x CPU limit + 10s)
	wallTimeout := time.Duration(0)
	if timeLimit > 0 {
		wallTimeout = time.Duration(timeLimit*2+10) * time.Second
	}

	wallStart := time.Now()

	// Start the process
	if err := cmd.Start(); err != nil {
		metrics := Metrics{
			WallTime: time.Since(wallStart).Seconds(),
			ExitCode: 1,
			Verdict:  "runtime-error",
		}
		writeMetrics(outputFile, &metrics)
		fmt.Fprintf(os.Stderr, "runguard: failed to start: %v\n", err)
		os.Exit(1)
	}

	// Wall-clock timeout killer
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	var waitErr error
	if wallTimeout > 0 {
		select {
		case waitErr = <-done:
			// Process finished
		case <-time.After(wallTimeout):
			// Wall-clock timeout - kill
			_ = cmd.Process.Kill()
			waitErr = <-done
		}
	} else {
		waitErr = <-done
	}

	wallTime := time.Since(wallStart)

	// Get resource usage from the child process
	var rusage syscall.Rusage
	exitCode := 0
	signalNum := 0
	verdict := "ok"

	if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			ws := exitErr.Sys().(syscall.WaitStatus)
			if ws.Signaled() {
				signalNum = int(ws.Signal())
				switch ws.Signal() {
				case syscall.SIGXCPU, syscall.SIGKILL:
					verdict = "time-limit"
				case syscall.SIGSEGV, syscall.SIGBUS:
					verdict = "runtime-error"
				default:
					verdict = "runtime-error"
				}
				exitCode = 128 + signalNum
			} else {
				exitCode = ws.ExitStatus()
				verdict = "runtime-error"
			}
			rusage = *exitErr.SysUsage().(*syscall.Rusage)
		} else {
			exitCode = 1
			verdict = "runtime-error"
		}
	} else {
		rusage = *cmd.ProcessState.SysUsage().(*syscall.Rusage)
	}

	userTime := timevalToSeconds(rusage.Utime)
	sysTime := timevalToSeconds(rusage.Stime)
	cpuTime := userTime + sysTime

	// On Linux, ru_maxrss is in KB already
	maxRSS := rusage.Maxrss

	// Check CPU time limit
	if timeLimit > 0 && cpuTime > timeLimit {
		verdict = "time-limit"
	}

	// Check memory limit (compare in KB)
	if memoryLimit > 0 && maxRSS > memoryLimit {
		verdict = "memory-limit"
	}

	metrics := Metrics{
		WallTime:   wallTime.Seconds(),
		CPUTime:    cpuTime,
		UserTime:   userTime,
		SystemTime: sysTime,
		MaxRSS:     maxRSS,
		ExitCode:   exitCode,
		Signal:     signalNum,
		Verdict:    verdict,
	}

	writeMetrics(outputFile, &metrics)

	os.Exit(exitCode)
}

func timevalToSeconds(tv syscall.Timeval) float64 {
	return float64(tv.Sec) + float64(tv.Usec)/1e6
}

func writeMetrics(path string, m *Metrics) {
	data, err := json.Marshal(m)
	if err != nil {
		fmt.Fprintf(os.Stderr, "runguard: failed to marshal metrics: %v\n", err)
		return
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "runguard: failed to write metrics: %v\n", err)
	}
}
