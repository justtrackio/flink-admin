package internal

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	eventsv1 "k8s.io/api/events/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	ClientModeInCluster  = "in-cluster"
	ClientModeKubeConfig = "kube-config"
)

type KubeSettings struct {
	ClientMode string `cfg:"client_mode" default:"in-cluster"`
	Context    string `cfg:"context"`
}

type k8sServiceCtxKey struct{}

func ProvideK8sService(ctx context.Context, config cfg.Config, logger log.Logger) (*K8sService, error) {
	return appctx.Provide(ctx, k8sServiceCtxKey{}, func() (*K8sService, error) {
		settings := &KubeSettings{}
		if err := config.UnmarshalKey("kube", settings); err != nil {
			return nil, fmt.Errorf("could not unmarshal kube settings: %w", err)
		}

		if settings.ClientMode == ClientModeInCluster {
			clientConfig, err := rest.InClusterConfig()
			if err != nil {
				return nil, fmt.Errorf("could not load in cluster config: %w", err)
			}

			return newK8sServiceFromConfig(clientConfig, logger)
		}

		rules := clientcmd.NewDefaultClientConfigLoadingRules()
		loader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, &clientcmd.ConfigOverrides{
			CurrentContext: settings.Context,
		})

		clientConfig, err := loader.ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("could not load config: %w", err)
		}

		return newK8sServiceFromConfig(clientConfig, logger)
	})
}

func newK8sServiceFromConfig(clientConfig *rest.Config, logger log.Logger) (*K8sService, error) {
	client, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("could not create k8s client: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("could not create dynamic client: %w", err)
	}

	return &K8sService{
		logger:        logger.WithChannel("k8s"),
		dynamicClient: dynamicClient,
		client:        client,
	}, nil
}

type K8sService struct {
	logger        log.Logger
	dynamicClient dynamic.Interface
	client        *kubernetes.Clientset
}

func (s *K8sService) WatchDeployments(ctx context.Context) (watch.Interface, error) {
	gvr := schema.GroupVersionResource{Group: "flink.apache.org", Version: "v1beta1", Resource: "flinkdeployments"}
	deployments := s.dynamicClient.Resource(gvr)

	return deployments.Watch(ctx, metav1.ListOptions{})
}

func (s *K8sService) GetEvents(ctx context.Context, namespace string, name string) (*eventsv1.EventList, error) {
	fieldSelector := fmt.Sprintf("regarding.name=%s", name)
	events, err := s.client.EventsV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fieldSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("could not list events for %s/%s: %w", namespace, name, err)
	}

	return events, nil
}
