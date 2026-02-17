package internal

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/coffin"
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
		resultChan: make(chan DeploymentEvent),
	}, nil
}

type DeploymentWatcher struct {
	k8sService *K8sService
	resultChan chan DeploymentEvent
}

func (s *DeploymentWatcher) Watch(ctx context.Context, namespaces []string) error {
	if len(namespaces) == 0 {
		close(s.resultChan)

		return nil
	}

	cfn, cfnCtx := coffin.WithContext(ctx)

	for _, namespace := range namespaces {
		cfn.GoWithContext(cfnCtx, func(ctx context.Context) error {
			return s.watchNamespace(ctx, namespace)
		})
	}

	err := cfn.Wait()
	close(s.resultChan)

	return err
}

// watchNamespace watches a single namespace for FlinkDeployment events, automatically
// reconnecting when the underlying K8s watch channel closes.
func (s *DeploymentWatcher) watchNamespace(ctx context.Context, namespace string) error {
	for {
		watcher, err := s.k8sService.WatchDeployments(ctx, namespace)
		if err != nil {
			return fmt.Errorf("could not watch deployments: %w", err)
		}

		if err := s.processEvents(ctx, watcher); err != nil {
			return err
		}
	}
}

// processEvents reads events from a K8s watcher until the channel closes or the context is cancelled.
// Returns nil when the watch channel closes (signaling the caller to reconnect) or the context is done.
// Returns an error only on unrecoverable failures.
func (s *DeploymentWatcher) processEvents(ctx context.Context, watcher watch.Interface) error {
	defer watcher.Stop()

	var err error
	var fd *FlinkDeployment

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return nil
			}

			if fd, err = FromUnstructured(event.Object); err != nil {
				return fmt.Errorf("could not convert unstructured to FlinkDeployment: %w", err)
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
