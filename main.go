package main

import (
	"context"
	"fmt"
	"time"

	"github.com/cenkalti/backoff"
	config "github.com/ndau/dao-voting-setup/configuration"
	"github.com/ndau/dao-voting-setup/dal"
	"github.com/ndau/dao-voting-setup/eventing"
	configure "github.com/ndau/go-config"
	logger "github.com/ndau/go-logger"
)

// main this is the main knative function.
// if we panic here upon upgrade knative will not upgrade the pod and will use that last successful version of the container
func main() {
	// Load logger and configurator
	log, err := logger.New("main", "main")
	if err != nil {
		fmt.Println("failed to create logger ", err)
		return
	}

	log.Infof("Initializing config...")
	cfg, er := configure.New()
	if err != nil {
		log.Error(er)
		return
	}

	log.Infof("Loading config...")
	ctx := context.Background()

	cf, e := config.LoadConfig(ctx, cfg, log)
	if e != nil {
		log.Error(e)
		return
		//panic(e)
	}

	var repo dal.Repo
	err = backoff.Retry(func() error {
		repo, err = dal.NewDb(cf, log)
		if err != nil {
			log.Errorf("Failed to initialize db client: %v", err)
		}

		return err
	}, backoff.NewConstantBackOff(4*time.Second))

	if err != nil {
		return
	}
	defer repo.Close()

	kn, err := eventing.NewKnClient(cf, log)
	if err != nil {
		log.Error("Failed to initialize knative client: %v", err)
		return
		//panic(erKn)
	}

	kn.Run(ctx, repo, cf)

	log.Info("waiting on context Done channel ...")
	<-ctx.Done()
	log.Info("cancelled context...")
}
