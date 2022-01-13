package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	koopercontroller "github.com/spotahome/kooper/v2/controller"
	kooperlog "github.com/spotahome/kooper/v2/log/logrus"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	authbackendfactory "github.com/slok/bilrost/internal/authbackend/factory"
	"github.com/slok/bilrost/internal/backup"
	"github.com/slok/bilrost/internal/controller"
	"github.com/slok/bilrost/internal/kubernetes"
	kubernetesclient "github.com/slok/bilrost/internal/kubernetes/client"
	"github.com/slok/bilrost/internal/log"
	bilrostprometheus "github.com/slok/bilrost/internal/metrics/prometheus"
	"github.com/slok/bilrost/internal/proxy"
	"github.com/slok/bilrost/internal/proxy/oauth2proxy"
	"github.com/slok/bilrost/internal/security"
)

// Run runs the main application.
func Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

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
	kubeBilrostCli, err := kubernetesclient.BaseFactory.NewBilrostClient(context.TODO(), kcfg)
	if err != nil {
		return fmt.Errorf("could not create K8S Bilrost client: %w", err)
	}
	kubeCoreCli, err := kubernetesclient.BaseFactory.NewCoreClient(context.TODO(), kcfg)
	if err != nil {
		return fmt.Errorf("could not create K8S core client: %w", err)
	}

	// Create main dependencies.
	metricsRecorder := bilrostprometheus.NewRecorder(prometheus.DefaultRegisterer)
	kubeSvc := kubernetes.NewMeasuredService(metricsRecorder, kubernetes.NewService(kubeCoreCli, kubeBilrostCli, logger))
	proxyProvisioner := proxy.NewMeasuredOIDCProvisioner(
		"oauth2proxy",
		metricsRecorder,
		oauth2proxy.NewOIDCProvisioner(kubeSvc, logger))
	backupSvc := backup.NewMeasuredbackupper("ingress", metricsRecorder, backup.NewIngressBackupper(kubeSvc, logger))
	authBackFactory := authbackendfactory.NewFactory(cmdCfg.NamespaceRunning, metricsRecorder, kubeSvc, logger)
	secSvc, err := security.NewService(security.ServiceConfig{
		Backupper:             backupSvc,
		ServiceTranslator:     kubeSvc,
		OIDCProxyProvisioner:  proxyProvisioner,
		AuthBackendRegFactory: authBackFactory,
		AuthBackendRepo:       kubeSvc,
		Logger:                logger,
	})
	if err != nil {
		return fmt.Errorf("could not create security service: %w", err)
	}

	// Prepare our run entrypoints.
	var g run.Group

	// Serving HTTP server.
	{
		mux := http.NewServeMux()

		// Metrics.
		mux.Handle(cmdCfg.MetricsPath, promhttp.Handler())

		// Pprof.
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

		server := &http.Server{
			Addr:    cmdCfg.ListenAddr,
			Handler: mux,
		}

		g.Add(
			func() error {
				logger.WithKV(log.KV{"addr": cmdCfg.ListenAddr}).Infof("http server listening for requests")
				return server.ListenAndServe()
			},
			func(_ error) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				err := server.Shutdown(ctx)
				if err != nil {
					logger.Errorf("error shutting down metrics server: %w", err)
				}
			},
		)
	}

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

	// Controllers.
	// We create and run 2 controllers that have the same handler.
	//
	// The primary controller is based on Ingresses and the secondary controller is based on
	// IngressAuth CR.
	//
	// Both controllers end executing the same reconciliation loop but if anything changes in any of
	// the resources we will reconcile again.
	{
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		const retries = 2
		handler, err := controller.NewHandler(controller.HandlerConfig{
			KubernetesRepo: kubeSvc,
			SecuritySvc:    secSvc,
			Logger:         logger,
		})
		if err != nil {
			return fmt.Errorf("could not create controller handler: %w", err)
		}

		ctrlIng, err := koopercontroller.New(&koopercontroller.Config{
			Handler:              handler,
			Retriever:            controller.NewIngressRetriever(cmdCfg.NamespaceFilter, kubeSvc),
			MetricsRecorder:      metricsRecorder,
			Logger:               kooperLogger,
			Name:                 "bilrost-controller-ingress",
			ConcurrentWorkers:    cmdCfg.Workers,
			ProcessingJobRetries: retries,
			ResyncInterval:       cmdCfg.ResyncInterval,
		})
		if err != nil {
			return fmt.Errorf("could not create backend auth kubernetes controller: %w", err)
		}

		g.Add(
			func() error {
				return ctrlIng.Run(ctx)
			},
			func(_ error) {
				cancel()
			},
		)

		ctrlIngAuth, err := koopercontroller.New(&koopercontroller.Config{
			Handler:              handler,
			Retriever:            controller.NewIngressAuthRetriever(cmdCfg.NamespaceFilter, kubeSvc),
			MetricsRecorder:      metricsRecorder,
			Logger:               kooperLogger,
			Name:                 "bilrost-controller-ingressauth",
			ConcurrentWorkers:    cmdCfg.Workers,
			ProcessingJobRetries: retries,
			// This controller is for secondary resources, we resync with the primary resource,
			// we only want updates on this one so we can be faster on secondary resource updates.
			DisableResync: true,
		})
		if err != nil {
			return fmt.Errorf("could not create backend auth kubernetes controller: %w", err)
		}

		g.Add(
			func() error {
				return ctrlIngAuth.Run(ctx)
			},
			func(_ error) {
				cancel()
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
			return nil, fmt.Errorf("error loading kubernetes configuration ºide cluster, check app is running outside kubernetes cluster or run in development mode: %w", err)
		}
		cfg = config
	}

	// Set better cli rate limiter.
	cfg.QPS = 100
	cfg.Burst = 100

	return cfg, nil
}

func main() {
	ctx := context.Background()
	err := Run(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error running application: %s", err)
		os.Exit(1)
	}

	os.Exit(0)
}
