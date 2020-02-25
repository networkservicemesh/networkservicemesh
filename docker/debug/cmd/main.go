// Dummy entry point for making dev-container
package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
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
	go func() {
		_, copyErr := io.Copy(os.Stdout, stdout)
		checkError(copyErr)
	}()
	go func() {
		_, copyErr := io.Copy(os.Stderr, stderr)
		checkError(copyErr)
	}()

	// Don't let main() exit before our command has finished running
	err = cmd.Wait() // Doesn't block
	checkError(err)

	println("Initialisation done... \nPlease use docker run debug.sh app to attach and start debug for particular application\n#You could do Ctrl+C to detach from this log.")
	c := tools.NewOSSignalChannel()
	<-c
}
