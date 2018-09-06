package main

import (
	"github.com/ligato/networkservicemesh/plugins/sriovconfigmapcommand"
	"github.com/ligato/networkservicemesh/utils/command"
	"github.com/spf13/cobra"
)

func main() {
	cmd := &cobra.Command{Use: "nsm-generate-sriov-configmap"}
	command.SetRootCmd(cmd)
	app := sriovconfigmapcommand.NewPlugin()
	app.Run()
}
