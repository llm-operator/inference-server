package k8s

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

// EventHandlerWithContext is an interface similar to cache.EventHandler but with a context object.
type EventHandlerWithContext interface {
	ProcessAdd(ctx context.Context, obj interface{})
	ProcessUpdate(ctx context.Context, oldObj, newObj interface{})
	ProcessDelete(ctx context.Context, obj interface{})
}

// Informers keeps multiple informers.
type Informers struct {
	stopCh      chan struct{}
	podInformer cache.SharedIndexInformer
}

// NewInformers creates a new Informer for Pod.
func NewInformers(
	k8sclient *Client,
	stopCh chan struct{},
	namespace string,
	labels map[string]string,
) (*Informers, error) {
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: labels,
	})
	if err != nil {
		return nil, err
	}
	factory := informers.NewSharedInformerFactoryWithOptions(
		k8sclient.CoreClientset,
		0, /*defaultResync*/
		informers.WithNamespace(namespace),
		informers.WithTweakListOptions(func(opts *metav1.ListOptions) {
			opts.LabelSelector = selector.String()
		}),
	)

	return &Informers{
		stopCh:      stopCh,
		podInformer: factory.Core().V1().Pods().Informer(),
	}, nil
}

// Run runs the informer.
func (is *Informers) Run() {
	is.podInformer.Run(is.stopCh)
}

// SetPodEventHandlers sets the specified handlers to the Pod informer.
func (is *Informers) SetPodEventHandlers(
	ctx context.Context,
	handlers []EventHandlerWithContext,
) error {
	_, err := is.podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			for _, handler := range handlers {
				handler.ProcessAdd(ctx, obj)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			for _, handler := range handlers {
				handler.ProcessUpdate(ctx, oldObj, newObj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			for _, handler := range handlers {
				handler.ProcessDelete(ctx, obj)
			}
		},
	})
	return err
}
