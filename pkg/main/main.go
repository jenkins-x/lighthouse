package main

import (
	"fmt"
	"os"

	"github.com/jenkins-x/lighthouse/pkg/cmd"
	"github.com/spf13/cobra"
)

// Entrypoint for the command
func main() {

	cmds := &cobra.Command{
		Use:   "lighthouse",
		Short: "a command line for lighthouse",
		// PersistentPreRun: setLoggingLevel,
		// Run:              runHelp,
	}

	cmds.AddCommand(cmd.NewCmdWebhook())

	err := cmds.Execute()
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
		os.Exit(1)
	}
	os.Exit(0)

}
