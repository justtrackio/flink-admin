package main

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

func NewServiceFlink(ctx context.Context, config cfg.Config, logger log.Logger) (*ServiceFlink, error) {
	var err error
	var client *FlinkClient
	var k8sService *K8sService

	if client, err = ProvideFlinkClient(ctx, config, logger); err != nil {
		return nil, fmt.Errorf("could not create flink client: %w", err)
	}

	if k8sService, err = ProvideK8sService(ctx, config, logger); err != nil {
		return nil, fmt.Errorf("could not create k8s service: %w", err)
	}

	return &ServiceFlink{
		logger:     logger.WithChannel("flink"),
		client:     client,
		k8sService: k8sService,
	}, nil
}

type ServiceFlink struct {
	logger     log.Logger
	client     *FlinkClient
	k8sService *K8sService
}

