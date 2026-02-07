package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type KubeSettings struct {
	Context    string   `cfg:"context"`
	Namespaces []string `cfg:"namespaces"`
}

type k8sServiceCtxKey struct{}

func ProvideK8sService(ctx context.Context, config cfg.Config, logger log.Logger) (*K8sService, error) {
	return appctx.Provide(ctx, k8sServiceCtxKey{}, func() (*K8sService, error) {
		var err error
		var clientConfig *rest.Config
		var client *kubernetes.Clientset
		var dynamicClient dynamic.Interface

		settings := &KubeSettings{}
		if err = config.UnmarshalKey("kube", settings); err != nil {
			return nil, fmt.Errorf("could not unmarshal kube settings: %w", err)
		}

		if len(settings.Namespaces) == 0 {
			return nil, fmt.Errorf("kube.namespaces must contain at least one namespace")
		}

		rules := clientcmd.NewDefaultClientConfigLoadingRules()
		loader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, &clientcmd.ConfigOverrides{
			CurrentContext: settings.Context,
		})

		if clientConfig, err = loader.ClientConfig(); err != nil {
			return nil, fmt.Errorf("could not load config: %w", err)
		}

		if client, err = kubernetes.NewForConfig(clientConfig); err != nil {
			return nil, fmt.Errorf("could not create k8s client: %w", err)
		}

		if dynamicClient, err = dynamic.NewForConfig(clientConfig); err != nil {
			return nil, fmt.Errorf("could not create dynamic client: %w", err)
		}

		gvr := schema.GroupVersionResource{Group: "flink.apache.org", Version: "v1beta1", Resource: "flinkdeployments"}

		// Create resource interfaces for each namespace
		deploymentsByNamespace := make(map[string]dynamic.ResourceInterface)
		for _, ns := range settings.Namespaces {
			deploymentsByNamespace[ns] = dynamicClient.Resource(gvr).Namespace(ns)
		}

		return &K8sService{
			logger:                 logger.WithChannel("k8s"),
			dynamicClient:          dynamicClient,
			client:                 client,
			gvr:                    gvr,
			namespaces:             settings.Namespaces,
			deploymentsByNamespace: deploymentsByNamespace,
			cache:                  make(map[string]Cluster),
		}, nil
	})
}

type K8sService struct {
	logger                 log.Logger
	dynamicClient          dynamic.Interface
	client                 *kubernetes.Clientset
	gvr                    schema.GroupVersionResource
	namespaces             []string
	deploymentsByNamespace map[string]dynamic.ResourceInterface

	cacheMu sync.RWMutex
	cache   map[string]Cluster // keyed by namespace/name
}

func (s *K8sService) ListNodes(ctx context.Context) ([]v1.Node, error) {
	var err error
	var list *v1.NodeList

	selector := labels.SelectorFromSet(map[string]string{
		"scheduling.cast.ai/node-template": "lake-shovel",
	})
	opts := metav1.ListOptions{LabelSelector: selector.String()}

	if list, err = s.client.CoreV1().Nodes().List(ctx, opts); err != nil {
		return nil, fmt.Errorf("could not list nodes: %w", err)
	}

	return list.Items, nil
}

func (s *K8sService) WatchNodes(ctx context.Context) (watch.Interface, error) {
	selector := labels.SelectorFromSet(map[string]string{
		"scheduling.cast.ai/node-template": "lake-shovel",
	})
	opts := metav1.ListOptions{LabelSelector: selector.String()}

	return s.client.CoreV1().Nodes().Watch(ctx, opts)
}

func (s *K8sService) ListPods(ctx context.Context, namespace string, labelSelector string) ([]v1.Pod, error) {
	opts := metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	list, err := s.client.CoreV1().Pods(namespace).List(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("could not list pods: %w", err)
	}

	return list.Items, nil
}

func (s *K8sService) WatchDeployments(ctx context.Context, namespace string) (watch.Interface, error) {
	gvr := schema.GroupVersionResource{Group: "flink.apache.org", Version: "v1beta1", Resource: "flinkdeployments"}
	deployments := s.dynamicClient.Resource(gvr).Namespace(namespace)

	return deployments.Watch(ctx, metav1.ListOptions{})
}
