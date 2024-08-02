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

package qosmanager

import (
	"time"

	apiruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	"github.com/clay-wangzhi/cfs-quota-burst/pkg/koordlet/metriccache"
	ma "github.com/clay-wangzhi/cfs-quota-burst/pkg/koordlet/metricsadvisor/framework"
	"github.com/clay-wangzhi/cfs-quota-burst/pkg/koordlet/qosmanager/framework"
	"github.com/clay-wangzhi/cfs-quota-burst/pkg/koordlet/qosmanager/plugins"
	"github.com/clay-wangzhi/cfs-quota-burst/pkg/koordlet/resourceexecutor"
	"github.com/clay-wangzhi/cfs-quota-burst/pkg/koordlet/statesinformer"
)

type QOSManager interface {
	Run(stopCh <-chan struct{}) error
}

type qosManager struct {
	options *framework.Options
	context *framework.Context
}

func NewQOSManager(cfg *framework.Config, schema *apiruntime.Scheme, kubeClient clientset.Interface, nodeName string,
	statesInformer statesinformer.StatesInformer, metricCache metriccache.MetricCache, metricAdvisorConfig *ma.Config) QOSManager {
	cgroupReader := resourceexecutor.NewCgroupReader()

	opt := &framework.Options{
		CgroupReader:        cgroupReader,
		StatesInformer:      statesInformer,
		MetricCache:         metricCache,
		KubeClient:          kubeClient,
		Config:              cfg,
		MetricAdvisorConfig: metricAdvisorConfig,
	}

	ctx := &framework.Context{
		Strategies: make(map[string]framework.QOSStrategy, len(plugins.StrategyPlugins)),
	}

	for name, strategyFn := range plugins.StrategyPlugins {
		ctx.Strategies[name] = strategyFn(opt)
	}

	r := &qosManager{
		options: opt,
		context: ctx,
	}
	return r
}

func (r *qosManager) setup() {
	for _, s := range r.context.Strategies {
		s.Setup(r.context)
	}
}

func (r *qosManager) Run(stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	// minimum interval is one second.
	if r.options.MetricAdvisorConfig.CollectResUsedInterval < time.Second {
		klog.Infof("collectResUsedIntervalSeconds is %v, qos manager is disabled",
			r.options.MetricAdvisorConfig.CollectResUsedInterval)
		return nil
	}

	klog.Info("Starting qos manager")
	r.setup()

	for name, strategy := range r.context.Strategies {
		klog.V(4).Infof("ready to start qos strategy %v", name)
		if !strategy.Enabled() {
			klog.V(4).Infof("qos strategy %v is not enabled, skip running", name)
			continue
		}
		go strategy.Run(stopCh)
		klog.V(4).Infof("qos strategy %v start", name)
	}

	klog.Info("Starting qosManager successfully")
	<-stopCh
	klog.Info("shutting down qosManager")
	return nil
}
