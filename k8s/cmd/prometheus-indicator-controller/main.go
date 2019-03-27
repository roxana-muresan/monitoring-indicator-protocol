package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pivotal/monitoring-indicator-protocol/k8s/pkg/client/clientset/versioned"
	"github.com/pivotal/monitoring-indicator-protocol/k8s/pkg/prometheus"

	coreV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/klog"

	"code.cloudfoundry.org/go-envstruct"
	informers "github.com/pivotal/monitoring-indicator-protocol/k8s/pkg/client/informers/externalversions"
)

type config struct {
	Namespace string `env:"NAMESPACE,required,report"`
}

func init() {
	klog.SetOutput(os.Stderr)
}

func main() {
	flag.Parse()
	ctx := setupSignalHandler()

	var conf config
	err := envstruct.Load(&conf)
	if err != nil {
		log.Fatal(err.Error())
	}
	err = envstruct.WriteReport(&conf)
	if err != nil {
		log.Fatal(err.Error())
	}

	cfg, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal(err.Error())
	}

	client, err := versioned.NewForConfig(cfg)
	if err != nil {
		log.Fatal(err.Error())
	}

	coreV1Client, err := coreV1.NewForConfig(cfg)
	if err != nil {
		log.Fatal(err.Error())
	}

	promConfig := prometheus.NewConfig()

	controller := prometheus.NewController(
		coreV1Client.ConfigMaps(conf.Namespace),
		promConfig,
	)

	informerFactory := informers.NewSharedInformerFactory(client, time.Second*30)

	indicatorInformer := informerFactory.Apps().V1alpha1().IndicatorDocuments().Informer()
	indicatorInformer.AddEventHandler(controller)

	// blocks (main goroutine to dispatch events to our controller)
	indicatorInformer.Run(ctx.Done())
}

var onlyOneSignalHandler = make(chan struct{})

// setupSignalHandler registers SIGTERM and SIGINT. A context is returned
// which is canceled on one of these signals. If a second signal is caught,
// the program is terminated with exit code 1.
func setupSignalHandler() context.Context {
	close(onlyOneSignalHandler) // only call once, panic on calls > 1

	ctx, cancel := context.WithCancel(context.Background())

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cancel()
		<-c
		os.Exit(1) // second signal. Exit directly.
	}()

	return ctx
}
