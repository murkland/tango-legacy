package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/apenwarr/fixconsole"
)

var (
	logsDir = flag.String("logs_dir", "logs", "file to log to")
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

	if *logsDir != "" {
		os.MkdirAll(*logsDir, 0o700)

		logF, err := os.Create(filepath.Join(*logsDir, fmt.Sprintf("tango_%s.log", time.Now().Format("20060102030405"))))
		if err != nil {
			log.Fatalf("failed to open log file: %s", err)
		}
		log.Printf("logging to %s", logF.Name())
		logWriter = logF
	}

	cmd := exec.Command(execPath, append([]string{"-child"}, os.Args[1:]...)...)
	cmd.Stderr = logWriter
	if err := cmd.Run(); err != nil {
		log.Fatalf("child exited with error: %s", err)
	}
}
