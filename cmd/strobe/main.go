package main

import (
	"github.com/jenkins-x/lighthouse/cmd/strobe/start"
	"github.com/jenkins-x/lighthouse/cmd/strobe/sync"
	"github.com/jenkins-x/lighthouse/pkg/logrusutil"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func main() {
	logrusutil.ComponentInit("lighthouse-strobe")

	rootCmd := &cobra.Command{Use: "strobe"}
	rootCmd.AddCommand(sync.GetCommand(), start.GetCommand())

	if err := rootCmd.Execute(); err != nil {
		logrus.WithError(err).Fatal("Failed to execute command")
	}
}
