package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/oklog/run"
	"github.com/sirupsen/logrus"
	koopercontroller "github.com/spotahome/kooper/controller"
	kooperlog "github.com/spotahome/kooper/log/logrus"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	authbackendfactory "github.com/slok/bilrost/internal/authbackend/factory"
	"github.com/slok/bilrost/internal/backup"
	"github.com/slok/bilrost/internal/controller"
	"github.com/slok/bilrost/internal/kubernetes"
	kubernetesclient "github.com/slok/bilrost/internal/kubernetes/client"
	"github.com/slok/bilrost/internal/log"
	"github.com/slok/bilrost/internal/proxy/oauth2proxy"
	"github.com/slok/bilrost/internal/security"
)

// Run runs the main application.
func Run() error {
	// Load command flags and arguments.
	cmdCfg, err := NewCmdConfig()
	if err != nil {
		return fmt.Errorf("could not load command configuration: %w", err)
	}

	// Set up logger.
	logrusLog := logrus.New()
	logrusLogEntry := logrus.NewEntry(logrusLog).WithField("app", "bilrost")
	kooperLogger := kooperlog.New(logrusLogEntry.WithField("lib", "kooper"))
	logger := log.NewLogrus(logrusLogEntry)
	if cmdCfg.Debug {
		logrusLog.SetLevel(logrus.DebugLevel)
	}

	// Load Kubernetes clients.
	logger.Infof("loading Kubernetes configuration...")
	kcfg, err := loadKubernetesConfig(*cmdCfg)
	if err != nil {
		return fmt.Errorf("could not load K8S configuration: %w", err)
	}
	kBilrostCli, err := kubernetesclient.BaseFactory.NewBilrostClient(context.TODO(), kcfg)
	if err != nil {
		return fmt.Errorf("could not create K8S Bilrost client: %w", err)
	}
	kCoreCli, err := kubernetesclient.BaseFactory.NewCoreClient(context.TODO(), kcfg)
	if err != nil {
		return fmt.Errorf("could not create K8S core client: %w", err)
	}

	// Create services.
	kubeSvc := kubernetes.NewService(kCoreCli, kBilrostCli, logger)
	authBackFactory := authbackendfactory.NewFactory(logger)
	proxyProvisioner := oauth2proxy.NewOIDCProvisioner(kubeSvc, logger)
	backupSvc := backup.NewIngressBackupper(kubeSvc, logger)
	secSvc, err := security.NewService(security.ServiceConfig{
		Backupper:              backupSvc,
		ServiceTranslator:      kubeSvc,
		OIDCProxyProvisioner:   proxyProvisioner,
		AuthBackendRepoFactory: authBackFactory,
		AuthBackendRepo:        kubeSvc,
		Logger:                 logger,
	})
	if err != nil {
		return fmt.Errorf("could not create security service: %w", err)
	}

	// Prepare our run entrypoints.
	var g run.Group

	// OS signals.
	{
		sigC := make(chan os.Signal, 1)
		exitC := make(chan struct{})
		signal.Notify(sigC, syscall.SIGTERM, syscall.SIGINT)

		g.Add(
			func() error {
				select {
				case s := <-sigC:
					logger.Infof("signal %s received", s)
					return nil
				case <-exitC:
					return nil
				}
			},
			func(_ error) {
				close(exitC)
			},
		)
	}

	// Controller.
	{
		handler, err := controller.NewHandler(controller.HandlerConfig{
			KubernetesRepo: kubeSvc,
			SecuritySvc:    secSvc,
			Logger:         logger,
		})
		if err != nil {
			return fmt.Errorf("could not create ingress handler: %w", err)
		}

		ctrl, err := koopercontroller.New(&koopercontroller.Config{
			Handler:              handler,
			Retriever:            controller.NewRetriever(cmdCfg.Namespace, kubeSvc),
			Logger:               kooperLogger,
			Name:                 "security-controller",
			ConcurrentWorkers:    cmdCfg.Workers,
			ProcessingJobRetries: 2,
		})
		if err != nil {
			return fmt.Errorf("could not create backend auth kubernetes controller: %w", err)
		}

		stopC := make(chan struct{})
		g.Add(
			func() error {
				return ctrl.Run(stopC)
			},
			func(_ error) {
				close(stopC)
			},
		)
	}

	err = g.Run()
	if err != nil {
		return err
	}

	return nil
}

// loadKubernetesConfig loads kubernetes configuration based on flags.
func loadKubernetesConfig(cmdCfg CmdConfig) (*rest.Config, error) {
	var cfg *rest.Config

	// If devel mode then use configuration flag path.
	if cmdCfg.Development {
		config, err := clientcmd.BuildConfigFromFlags("", cmdCfg.KubeConfig)
		if err != nil {
			return nil, fmt.Errorf("could not load configuration: %w", err)
		}
		cfg = config
	} else {
		config, err := rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("error loading kubernetes configuration inside cluster, check app is running outside kubernetes cluster or run in development mode: %w", err)
		}
		cfg = config
	}

	// Set better cli rate limiter.
	cfg.QPS = 100
	cfg.Burst = 100

	return cfg, nil
}

func main() {
	err := Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error running application: %s", err)
		os.Exit(1)
	}

	os.Exit(0)
}
