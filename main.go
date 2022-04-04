package main

import (
	"flag"
	"io"
	"log"
	"os"
	"os/exec"

	"github.com/apenwarr/fixconsole"
)

var (
	logFile = flag.String("log_file", "tango.log", "file to log to")
	child   = flag.Bool("child", false, "is this the child process?")
)

func main() {
	flag.Parse()
	if err := fixconsole.FixConsoleIfNeeded(); err != nil {
		log.Fatalf("failed to fix console: %s", err)
	}

	if *child {
		childMain()
		return
	}

	execPath, err := os.Executable()
	if err != nil {
		log.Fatalf("failed to locate executable: %s", err)
	}

	var logWriter io.Writer = os.Stderr

	if *logFile != "" {
		logF, err := os.Create(*logFile)
		if err != nil {
			log.Fatalf("failed to open log file: %s", err)
		}
		log.Printf("logging to %s", *logFile)
		logWriter = logF
	}

	cmd := exec.Command(execPath, append([]string{"-child"}, os.Args[1:]...)...)
	cmd.Stderr = logWriter
	if err := cmd.Run(); err != nil {
		log.Fatalf("child exited with error: %s", err)
	}
}
