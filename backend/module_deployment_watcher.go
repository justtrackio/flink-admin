package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/uuid"
)

type deploymentWatcherModuleCtxKey struct{}

type DeploymentWatcherModule struct {
	kernel.EssentialModule
	kernel.ServiceStage

	logger      log.Logger
	watcher     *DeploymentWatcher
	deployments map[string]map[string]*FlinkDeployment
	lck         sync.Mutex
	channels    map[string]chan DeploymentEvent
}

func ProvideDeploymentWatcherModule(ctx context.Context, config cfg.Config, logger log.Logger) (*DeploymentWatcherModule, error) {
	return appctx.Provide(ctx, deploymentWatcherModuleCtxKey{}, func() (*DeploymentWatcherModule, error) {
		var err error
		var watcher *DeploymentWatcher

		if watcher, err = NewDeploymentWatcher(ctx, config, logger); err != nil {
			return nil, fmt.Errorf("failed to initialize k8s service: %w", err)
		}

		return &DeploymentWatcherModule{
			logger:      logger.WithChannel("k8s-watcher"),
			watcher:     watcher,
			deployments: make(map[string]map[string]*FlinkDeployment),
			channels:    map[string]chan DeploymentEvent{},
		}, nil
	})
}

func (m *DeploymentWatcherModule) Watch() (map[string]map[string]*FlinkDeployment, <-chan DeploymentEvent, chan bool) {
	m.lck.Lock()
	defer m.lck.Unlock()

	id := uuid.New().NewV4()
	ch := make(chan DeploymentEvent)
	m.logger.Info(context.Background(), "adding new watcher with id %s", id)
	m.channels[id] = ch
	stop := make(chan bool)

	go func() {
		<-stop
		m.lck.Lock()
		m.logger.Info(context.Background(), "removing watcher with id %s", id)
		delete(m.channels, id)
		m.lck.Unlock()
	}()

	return m.deployments, ch, stop
}

func (m *DeploymentWatcherModule) Run(ctx context.Context) error {
	m.logger.Info(ctx, "starting deployment watcher")

	cfn, cfnCtx := coffin.WithContext(context.Background())

	cfn.GoWithContext(ctx, func(ctx context.Context) error {
		<-ctx.Done()
		m.watcher.Stop()

		return nil
	})

	cfn.GoWithContext(ctx, func(ctx context.Context) error {
		timer := clock.NewRealTicker(time.Second * 5)

		for {
			select {
			case <-ctx.Done():
				return nil
			case <-timer.Chan():
				m.logger.Info(ctx, "serving %d watchers", len(m.channels))
			}
		}
	})

	for _, namespace := range []string{"annotators"} {
		m.deployments[namespace] = make(map[string]*FlinkDeployment)

		cfn.GoWithContext(cfnCtx, func(cfnCtx context.Context) error {
			return m.watcher.Watch(cfnCtx, namespace)
		})
	}

	cfn.GoWithContext(cfnCtx, func(cfnCtx context.Context) error {
		var ok bool
		var event DeploymentEvent

		for {
			select {
			case <-cfnCtx.Done():
				return nil

			case event, ok = <-m.watcher.ResultChan():
				if !ok {
					return nil
				}

				fd := event.Deployment
				m.deployments[fd.Namespace][fd.Name] = fd

				m.logger.Info(context.Background(), "got event from k8s watcher for %s", fd.Name)

				m.lck.Lock()
				for _, ch := range m.channels {
					ch <- event
				}
				m.lck.Unlock()
			}
		}
	})

	return cfn.Wait()
}
