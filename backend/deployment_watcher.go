package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"k8s.io/apimachinery/pkg/watch"
)

type DeploymentEvent struct {
	Type       watch.EventType  `json:"type"`
	Deployment *FlinkDeployment `json:"deployment"`
}

func NewDeploymentWatcher(ctx context.Context, config cfg.Config, logger log.Logger) (*DeploymentWatcher, error) {
	var err error
	var k8sService *K8sService

	if k8sService, err = ProvideK8sService(ctx, config, logger); err != nil {
		return nil, fmt.Errorf("could not create k8s service: %w", err)
	}

	return &DeploymentWatcher{
		k8sService: k8sService,
		stop:       make(chan struct{}),
		resultChan: make(chan DeploymentEvent),
	}, nil
}

type DeploymentWatcher struct {
	k8sService *K8sService
	once       sync.Once
	stop       chan struct{}
	resultChan chan DeploymentEvent
}

func (s *DeploymentWatcher) Watch(ctx context.Context, namespace string) error {
	var ok bool
	var err error

	defer s.once.Do(func() {
		close(s.resultChan)
	})

	for {
		if ok, err = s.doWatch(ctx, namespace); err != nil {
			return err
		}

		if ok {
			break
		}
	}

	return nil
}

func (s *DeploymentWatcher) doWatch(ctx context.Context, namespace string) (bool, error) {
	var err error
	var watcher watch.Interface
	var fd *FlinkDeployment

	if watcher, err = s.k8sService.WatchDeployments(ctx, namespace); err != nil {
		return true, fmt.Errorf("could not watch deployments: %w", err)
	}
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			return true, nil
		case <-s.stop:
			return true, nil
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return false, nil
			}

			if fd, err = FromUnstructured(event.Object); err != nil {
				return false, fmt.Errorf("could not convert unstructured to FlinkDeployment: %w", err)
			}

			s.resultChan <- DeploymentEvent{
				Type:       event.Type,
				Deployment: fd,
			}
		}
	}
}

func (w *DeploymentWatcher) ResultChan() <-chan DeploymentEvent {
	return w.resultChan
}

func (w *DeploymentWatcher) Stop() {
	close(w.stop)
}
