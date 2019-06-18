// Dummy entry point for making dev-container
package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
)

var version string

func checkError(err error) {
	if err != nil {
		println("Error: %s", err)
	}
}
func main() {
	println("Starting NSC DevEnv dummy application")
	fmt.Printf("Version: %v", version)
	cmd := exec.Command("bash", "/go/src/github.com/networkservicemesh/networkservicemesh/scripts/debug_env.sh")

	// Create stdout, stderr streams of type io.Reader
	stdout, err := cmd.StdoutPipe()
	checkError(err)
	stderr, err := cmd.StderrPipe()
	checkError(err)

	// Start command
	err = cmd.Start()
	checkError(err)

	// Non-blockingly echo command output to terminal
	go io.Copy(os.Stdout, stdout)
	go io.Copy(os.Stderr, stderr)

	// Don't let main() exit before our command has finished running
	cmd.Wait() // Doesn't block

	println("Initialisation done... \nPlease use docker run debug.sh app to attach and start debug for particular application\n#You could do Ctrl+C to detach from this log.")
	var wg sync.WaitGroup
	wg.Add(1)
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		wg.Done()
	}()
	wg.Wait()
}
