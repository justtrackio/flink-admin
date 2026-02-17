package internal

import (
	"context"
	"fmt"
	"sync"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/uuid"
)

type deploymentWatcherModuleCtxKey struct{}

type DeploymentWatcherModule struct {
	kernel.EssentialModule
	kernel.ServiceStage

	lck         sync.Mutex
	logger      log.Logger
	watcher     *DeploymentWatcher
	deployments map[string]map[string]*FlinkDeployment
	channels    map[string]chan DeploymentEvent
	namespaces  []string
}

func ProvideDeploymentWatcherModule(ctx context.Context, config cfg.Config, logger log.Logger) (*DeploymentWatcherModule, error) {
	return appctx.Provide(ctx, deploymentWatcherModuleCtxKey{}, func() (*DeploymentWatcherModule, error) {
		var err error
		var watcher *DeploymentWatcher
		var namespaces []string

		if watcher, err = NewDeploymentWatcher(ctx, config, logger); err != nil {
			return nil, fmt.Errorf("failed to initialize k8s service: %w", err)
		}

		if namespaces, err = config.GetStringSlice("flink.namespaces"); err != nil {
			return nil, fmt.Errorf("failed to read namespaces from config: %w", err)
		}

		return &DeploymentWatcherModule{
			logger:      logger.WithChannel("k8s-watcher"),
			watcher:     watcher,
			deployments: make(map[string]map[string]*FlinkDeployment),
			channels:    map[string]chan DeploymentEvent{},
			namespaces:  namespaces,
		}, nil
	})
}

func (m *DeploymentWatcherModule) Watch(ctx context.Context) (deployments map[string]map[string]*FlinkDeployment, events <-chan DeploymentEvent, stop chan bool) {
	m.lck.Lock()
	defer m.lck.Unlock()

	id := uuid.New().NewV4()
	m.logger.Info(ctx, "adding new watcher with id %s", id)
	m.channels[id] = make(chan DeploymentEvent, 16)
	stop = make(chan bool)

	go func() {
		<-stop
		m.lck.Lock()
		m.logger.Info(context.Background(), "removing watcher with id %s", id)
		delete(m.channels, id)
		m.lck.Unlock()
	}()

	return cloneDeployments(m.deployments), m.channels[id], stop
}

// GetDeployment retrieves a single deployment from the in-memory cache by namespace and name
func (m *DeploymentWatcherModule) GetDeployment(namespace, name string) (*FlinkDeployment, bool) {
	m.lck.Lock()
	defer m.lck.Unlock()

	var ok bool
	var nsDeployments map[string]*FlinkDeployment

	if nsDeployments, ok = m.deployments[namespace]; !ok {
		return nil, false
	}

	deployment, exists := nsDeployments[name]

	return deployment, exists
}

func (m *DeploymentWatcherModule) Run(ctx context.Context) error {
	m.logger.Info(ctx, "starting deployment watcher")

	for _, namespace := range m.namespaces {
		m.deployments[namespace] = make(map[string]*FlinkDeployment)
	}

	cfn, cfnCtx := coffin.WithContext(ctx)

	cfn.GoWithContext(cfnCtx, func(cfnCtx context.Context) error {
		return m.watcher.Watch(cfnCtx, m.namespaces)
	})

	cfn.GoWithContext(cfnCtx, func(cfnCtx context.Context) error {
		return m.streamEvents(cfnCtx)
	})

	return cfn.Wait()
}

func (m *DeploymentWatcherModule) streamEvents(ctx context.Context) error {
	defer m.closeChannels()

	for {
		select {
		case <-ctx.Done():
			return nil
		case event, ok := <-m.watcher.ResultChan():
			if !ok {
				return nil
			}
			m.applyEvent(ctx, event)
		}
	}
}

func (m *DeploymentWatcherModule) closeChannels() {
	m.lck.Lock()
	defer m.lck.Unlock()

	for id, ch := range m.channels {
		close(ch)
		delete(m.channels, id)
	}
}

func (m *DeploymentWatcherModule) applyEvent(ctx context.Context, event DeploymentEvent) {
	fd := event.Deployment
	m.lck.Lock()
	defer m.lck.Unlock()

	if _, ok := m.deployments[fd.Namespace]; !ok {
		m.deployments[fd.Namespace] = make(map[string]*FlinkDeployment)
	}
	m.deployments[fd.Namespace][fd.Name] = fd

	m.logger.Info(ctx, "got event from k8s watcher for %s", fd.Name)

	for _, ch := range m.channels {
		select {
		case ch <- event:
		default:
			// subscriber is not keeping up or dead — skip
		}
	}
}

func cloneDeployments(deployments map[string]map[string]*FlinkDeployment) map[string]map[string]*FlinkDeployment {
	cloned := make(map[string]map[string]*FlinkDeployment, len(deployments))
	for namespace, nsDeployments := range deployments {
		clonedNamespace := make(map[string]*FlinkDeployment, len(nsDeployments))
		for name, deployment := range nsDeployments {
			clonedNamespace[name] = deployment
		}
		cloned[namespace] = clonedNamespace
	}

	return cloned
}
