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

package impl

import (
	"context"
	"reflect"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

const (
	nodeInformerName PluginName = "nodeInformer"
)

type nodeInformer struct {
	nodeInformer cache.SharedIndexInformer
	nodeRWMutex  sync.RWMutex
	node         *corev1.Node
}

func NewNodeInformer() *nodeInformer {
	return &nodeInformer{}
}

func (s *nodeInformer) GetNode() *corev1.Node {
	s.nodeRWMutex.RLock()
	defer s.nodeRWMutex.RUnlock()
	if s.node == nil {
		return nil
	}
	return s.node.DeepCopy()
}

func (s *nodeInformer) Setup(ctx *PluginOption, state *PluginState) {
	s.nodeInformer = newNodeInformer(ctx.KubeClient, ctx.NodeName)
	s.nodeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			node, ok := obj.(*corev1.Node)
			if ok {
				s.syncNode(node)
			} else {
				klog.Errorf("node informer add func parse Node failed, obj %T", obj)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldNode, oldOK := oldObj.(*corev1.Node)
			newNode, newOK := newObj.(*corev1.Node)
			if !oldOK || !newOK {
				klog.Errorf("unable to convert object to *corev1.Node, old %T, new %T", oldObj, newObj)
				return
			}
			if newNode.ResourceVersion == oldNode.ResourceVersion {
				klog.V(5).Infof("find node %s has not changed", newNode.Name)
				return
			}
			s.syncNode(newNode)
		},
	})
}

func (s *nodeInformer) Start(stopCh <-chan struct{}) {
	klog.V(2).Infof("starting node informer")
	go s.nodeInformer.Run(stopCh)
	klog.V(2).Infof("node informer started")
}

func (s *nodeInformer) HasSynced() bool {
	if s.nodeInformer == nil {
		return false
	}
	synced := s.nodeInformer.HasSynced()
	klog.V(5).Infof("node informer has synced %v", synced)
	return synced
}

func newNodeInformer(client clientset.Interface, nodeName string) cache.SharedIndexInformer {
	tweakListOptionsFunc := func(opt *metav1.ListOptions) {
		opt.FieldSelector = "metadata.name=" + nodeName
	}

	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (apiruntime.Object, error) {
				tweakListOptionsFunc(&options)
				return client.CoreV1().Nodes().List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				tweakListOptionsFunc(&options)
				return client.CoreV1().Nodes().Watch(context.TODO(), options)
			},
		},
		&corev1.Node{},
		time.Hour*12,
		cache.Indexers{},
	)
}

func (s *nodeInformer) syncNode(newNode *corev1.Node) {
	klog.V(5).Infof("node update detail %v", newNode)
	s.nodeRWMutex.Lock()
	defer s.nodeRWMutex.Unlock()

	if isNodeMetadataUpdated(s.node, newNode) {
		klog.V(5).Info("node metadata changed, send NodeMetadata update callback")
	}

	s.node = newNode.DeepCopy()
}

func isNodeMetadataUpdated(oldNode, newNode *corev1.Node) bool {
	if oldNode == nil && newNode == nil {
		return false
	}
	if oldNode == nil || newNode == nil {
		return true
	}

	if !reflect.DeepEqual(oldNode.ObjectMeta.Labels, newNode.ObjectMeta.Labels) {
		return true
	}
	if !reflect.DeepEqual(oldNode.ObjectMeta.Annotations, newNode.ObjectMeta.Annotations) {
		return true
	}
	return false
}
