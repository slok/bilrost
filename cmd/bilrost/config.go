package main

import (
	"os"
	"path/filepath"

	"gopkg.in/alecthomas/kingpin.v2"
	"k8s.io/client-go/util/homedir"
)

// CmdConfig represents the configuration of the command.
type CmdConfig struct {
	Development bool
	Debug       bool

	Workers    int
	KubeConfig string
	Namespace  string
}

// NewCmdConfig returns a new command configuration.
func NewCmdConfig() (*CmdConfig, error) {
	kubeHome := filepath.Join(homedir.HomeDir(), ".kube", "config")

	c := &CmdConfig{}
	app := kingpin.New("bilrost", "A Kubernetes controller to secure services behind an ingress.")

	app.Flag("debug", "Enable debug mode.").BoolVar(&c.Debug)
	app.Flag("development", "Enable development mode.").BoolVar(&c.Development)
	app.Flag("kube-config", "kubernetes configuration path, only used when development mode enabled.").Default(kubeHome).Short('c').StringVar(&c.KubeConfig)
	app.Flag("namespace", "kubernetes namespace where the controller will listen to events.").Short('n').StringVar(&c.Namespace)
	app.Flag("workers", "concurrent processing workers for each kubernetes controller.").Default("3").Short('w').IntVar(&c.Workers)

	_, err := app.Parse(os.Args[1:])
	if err != nil {
		return nil, err
	}

	return c, nil
}
