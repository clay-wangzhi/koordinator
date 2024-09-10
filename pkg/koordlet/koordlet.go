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

package agent

import (
	"fmt"
	"os"
	"time"

	apiruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientset "k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/clay-wangzhi/koordinator/pkg/koordlet/config"
	"github.com/clay-wangzhi/koordinator/pkg/koordlet/metriccache"
	"github.com/clay-wangzhi/koordinator/pkg/koordlet/metricsadvisor"
	"github.com/clay-wangzhi/koordinator/pkg/koordlet/qosmanager"
	"github.com/clay-wangzhi/koordinator/pkg/koordlet/resourceexecutor"
	"github.com/clay-wangzhi/koordinator/pkg/koordlet/statesinformer"
	statesinformerimpl "github.com/clay-wangzhi/koordinator/pkg/koordlet/statesinformer/impl"
	"github.com/clay-wangzhi/koordinator/pkg/koordlet/util/system"
)

var (
	scheme = apiruntime.NewScheme()
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
}

type Daemon interface {
	Run(stopCh <-chan struct{})
}

type daemon struct {
	metricAdvisor  metricsadvisor.MetricAdvisor
	statesInformer statesinformer.StatesInformer
	metricCache    metriccache.MetricCache
	qosManager     qosmanager.QOSManager
	executor       resourceexecutor.ResourceUpdateExecutor
}

func NewDaemon(config *config.Configuration) (Daemon, error) {
	// get node name
	nodeName := os.Getenv("NODE_NAME")
	if len(nodeName) == 0 {
		return nil, fmt.Errorf("failed to new daemon: NODE_NAME env is empty")
	}
	klog.Infof("NODE_NAME is %v, start time %v", nodeName, float64(time.Now().Unix()))
	klog.Infof("sysconf: %+v, agentMode: %v", system.Conf, system.AgentMode)

	kubeClient := clientset.NewForConfigOrDie(config.KubeRestConf)
	metricCache, err := metriccache.NewMetricCache(config.MetricCacheConf)
	if err != nil {
		return nil, err
	}

	statesInformer := statesinformerimpl.NewStatesInformer(config.StatesInformerConf, kubeClient, nodeName)

	cgroupDriver := system.GetCgroupDriver()
	system.SetupCgroupPathFormatter(cgroupDriver)

	collectorService := metricsadvisor.NewMetricAdvisor(config.CollectorConf, statesInformer, metricCache)

	qosManager := qosmanager.NewQOSManager(config.QOSManagerConf, scheme, kubeClient, nodeName, statesInformer, metricCache, config.CollectorConf)

	if err != nil {
		return nil, err
	}

	d := &daemon{
		metricAdvisor:  collectorService,
		statesInformer: statesInformer,
		metricCache:    metricCache,
		qosManager:     qosManager,
		executor:       resourceexecutor.NewResourceUpdateExecutor(),
	}

	return d, nil
}

func (d *daemon) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	klog.Infof("Starting daemon")

	// start resource executor cache
	d.executor.Run(stopCh)

	go func() {
		if err := d.metricCache.Run(stopCh); err != nil {
			klog.Fatal("Unable to run the metric cache: ", err)
		}
	}()

	// start states informer
	go func() {
		if err := d.statesInformer.Run(stopCh); err != nil {
			klog.Fatal("Unable to run the states informer: ", err)
		}
	}()
	// wait for metric advisor sync
	if !cache.WaitForCacheSync(stopCh, d.statesInformer.HasSynced) {
		klog.Fatal("time out waiting for states informer to sync")
	}

	// start metric advisor
	go func() {
		if err := d.metricAdvisor.Run(stopCh); err != nil {
			klog.Fatal("Unable to run the metric advisor: ", err)
		}
	}()
	// wait for metric advisor sync
	if !cache.WaitForCacheSync(stopCh, d.metricAdvisor.HasSynced) {
		klog.Fatal("time out waiting for metric advisor to sync")
	}

	// start qos manager
	go func() {
		if err := d.qosManager.Run(stopCh); err != nil {
			klog.Fatal("Unable to run the qosManager: ", err)
		}
	}()

	klog.Info("Start daemon successfully")
	<-stopCh
	klog.Info("Shutting down daemon")
}
