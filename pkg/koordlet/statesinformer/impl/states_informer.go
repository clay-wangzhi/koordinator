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
	"fmt"

	koordclientset "github.com/clay-wangzhi/koordinator/pkg/client/clientset/versioned"
	"go.uber.org/atomic"
	corev1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientset "k8s.io/client-go/kubernetes"

	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/clay-wangzhi/koordinator/pkg/koordlet/metriccache"
	"github.com/clay-wangzhi/koordinator/pkg/koordlet/prediction"
	"github.com/clay-wangzhi/koordinator/pkg/koordlet/statesinformer"
)

const (
	HTTPScheme  = "http"
	HTTPSScheme = "https"
)

type StatesInformer interface {
	Run(stopCh <-chan struct{}) error
	HasSynced() bool
	GetNode() *corev1.Node
	GetCfsCM() *corev1.ConfigMap
	GetAllPods() []*statesinformer.PodMeta
}

type PluginName string

type PluginOption struct {
	config      *Config
	KubeClient  clientset.Interface
	KoordClient koordclientset.Interface
	NodeName    string
}

type PluginState struct {
	metricCache      metriccache.MetricCache
	informerPlugins  map[PluginName]informerPlugin
	predictorFactory prediction.PredictorFactory
}

type statesInformer struct {
	config       *Config
	metricsCache metriccache.MetricCache
	option       *PluginOption
	states       *PluginState
	started      *atomic.Bool
}

type informerPlugin interface {
	Setup(ctx *PluginOption, state *PluginState)
	Start(stopCh <-chan struct{})
	HasSynced() bool
}

// TODO merge all clients into one struct
func NewStatesInformer(config *Config, kubeClient clientset.Interface, crdClient koordclientset.Interface, metricsCache metriccache.MetricCache,
	nodeName string, predictorFactory prediction.PredictorFactory) StatesInformer {
	opt := &PluginOption{
		config:      config,
		KubeClient:  kubeClient,
		KoordClient: crdClient,
		NodeName:    nodeName,
	}
	stat := &PluginState{
		metricCache:      metricsCache,
		informerPlugins:  map[PluginName]informerPlugin{},
		predictorFactory: predictorFactory,
	}
	s := &statesInformer{
		config:  config,
		option:  opt,
		states:  stat,
		started: atomic.NewBool(false),
	}
	s.initInformerPlugins()
	return s
}

func (s *statesInformer) initInformerPlugins() {
	s.states.informerPlugins = DefaultPluginRegistry
}

func (s *statesInformer) setupPlugins() {
	for name, plugin := range s.states.informerPlugins {
		plugin.Setup(s.option, s.states)
		klog.V(2).Infof("plugin %v has been setup", name)
	}
}

func (s *statesInformer) Run(stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	klog.V(2).Infof("setup statesInformer")

	klog.V(2).Infof("starting informer plugins")
	s.setupPlugins()
	s.startPlugins(stopCh)

	// waiting for node synced.
	klog.V(2).Infof("waiting for informer syncing")
	waitInformersSynced := s.waitForSyncFunc()
	if !cache.WaitForCacheSync(stopCh, waitInformersSynced...) {
		return fmt.Errorf("timed out waiting for states informer caches to sync")
	}

	klog.Infof("start states informer successfully")
	s.started.Store(true)
	<-stopCh
	klog.Infof("shutting down states informer daemon")
	return nil
}

func (s *statesInformer) waitForSyncFunc() []cache.InformerSynced {
	waitInformersSynced := make([]cache.InformerSynced, 0, len(s.states.informerPlugins))
	for _, p := range s.states.informerPlugins {
		waitInformersSynced = append(waitInformersSynced, p.HasSynced)
	}
	return waitInformersSynced
}

func (s *statesInformer) startPlugins(stopCh <-chan struct{}) {
	for name, p := range s.states.informerPlugins {
		klog.V(4).Infof("starting informer plugin %v", name)
		go p.Start(stopCh)
	}
}

func (s *statesInformer) HasSynced() bool {
	for _, p := range s.states.informerPlugins {
		if !p.HasSynced() {
			return false
		}
	}
	return true
}

func (s *statesInformer) GetNode() *corev1.Node {
	nodeInformerIf := s.states.informerPlugins[nodeInformerName]
	nodeInformer, ok := nodeInformerIf.(*nodeInformer)
	if !ok {
		klog.Errorf("node informer format error")
		return nil
	}
	return nodeInformer.GetNode()
}

func (s *statesInformer) GetCfsCM() *corev1.ConfigMap {
	cmInformerIf := s.states.informerPlugins[cmInformerName]
	cmInformer, ok := cmInformerIf.(*cmInformer)
	if !ok {
		klog.Errorf("cm informer format error")
		return nil
	}
	return cmInformer.GetCm()
}

func (s *statesInformer) GetAllPods() []*statesinformer.PodMeta {
	podsInformerIf := s.states.informerPlugins[podsInformerName]
	podsInformer, ok := podsInformerIf.(*podsInformer)
	if !ok {
		klog.Errorf("pods informer format error")
		return nil
	}
	return podsInformer.GetAllPods()
}
