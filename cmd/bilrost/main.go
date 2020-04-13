package main

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/slok/bilrost/internal/log"
)

// Run runs the main application.
func Run() error {
	logrusLog := logrus.New()
	logger := log.NewLogrus(logrus.NewEntry(logrusLog))

	logger.Infof("Hello world")

	return nil
}

func main() {
	err := Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error running application: %s", err)
		os.Exit(1)
	}

	os.Exit(0)
}
