package util

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
)

// Stopper returns a channel that remains open until an interrupt is received.
func Stopper() chan struct{} {
	stop := make(chan struct{})
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		logrus.Warn("Interrupt received, attempting clean shutdown...")
		close(stop)
		<-c
		logrus.Error("Second interrupt received, force exiting...")
		os.Exit(1)
	}()
	return stop
}
