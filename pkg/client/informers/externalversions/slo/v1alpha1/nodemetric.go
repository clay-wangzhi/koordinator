/*
Copyright 2022 The Koordinator Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	time "time"

	slov1alpha1 "github.com/clay-wangzhi/koordinator/apis/slo/v1alpha1"
	versioned "github.com/clay-wangzhi/koordinator/pkg/client/clientset/versioned"
	internalinterfaces "github.com/clay-wangzhi/koordinator/pkg/client/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/clay-wangzhi/koordinator/pkg/client/listers/slo/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// NodeMetricInformer provides access to a shared informer and lister for
// NodeMetrics.
type NodeMetricInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.NodeMetricLister
}

type nodeMetricInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewNodeMetricInformer constructs a new informer for NodeMetric type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewNodeMetricInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredNodeMetricInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredNodeMetricInformer constructs a new informer for NodeMetric type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredNodeMetricInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.SloV1alpha1().NodeMetrics().List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.SloV1alpha1().NodeMetrics().Watch(context.TODO(), options)
			},
		},
		&slov1alpha1.NodeMetric{},
		resyncPeriod,
		indexers,
	)
}

func (f *nodeMetricInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredNodeMetricInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *nodeMetricInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&slov1alpha1.NodeMetric{}, f.defaultInformer)
}

func (f *nodeMetricInformer) Lister() v1alpha1.NodeMetricLister {
	return v1alpha1.NewNodeMetricLister(f.Informer().GetIndexer())
}
