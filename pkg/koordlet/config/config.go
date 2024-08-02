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

package config

import (
	"flag"

	"github.com/clay-wangzhi/cfs-quota-burst/pkg/koordlet/metriccache"
	maframework "github.com/clay-wangzhi/cfs-quota-burst/pkg/koordlet/metricsadvisor/framework"
	qmframework "github.com/clay-wangzhi/cfs-quota-burst/pkg/koordlet/qosmanager/framework"
	"github.com/clay-wangzhi/cfs-quota-burst/pkg/koordlet/resourceexecutor"
	statesinformerimpl "github.com/clay-wangzhi/cfs-quota-burst/pkg/koordlet/statesinformer/impl"
	"github.com/clay-wangzhi/cfs-quota-burst/pkg/koordlet/util/system"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

const (
	DefaultKoordletConfigMapNamespace = "koordinator-system"
	DefaultKoordletConfigMapName      = "koordlet-config"
)

type Configuration struct {
	ConfigMapName      string
	ConfigMapNamesapce string
	KubeRestConf       *rest.Config
	StatesInformerConf *statesinformerimpl.Config
	CollectorConf      *maframework.Config
	MetricCacheConf    *metriccache.Config
	QOSManagerConf     *qmframework.Config
}

func NewConfiguration() *Configuration {
	return &Configuration{
		ConfigMapName:      DefaultKoordletConfigMapName,
		ConfigMapNamesapce: DefaultKoordletConfigMapNamespace,
		StatesInformerConf: statesinformerimpl.NewDefaultConfig(),
		CollectorConf:      maframework.NewDefaultConfig(),
		MetricCacheConf:    metriccache.NewDefaultConfig(),
		QOSManagerConf:     qmframework.NewDefaultConfig(),
	}
}

func (c *Configuration) InitFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.ConfigMapName, "configmap-name", DefaultKoordletConfigMapName, "determines the name the koordlet configmap uses.")
	fs.StringVar(&c.ConfigMapNamesapce, "configmap-namespace", DefaultKoordletConfigMapNamespace, "determines the namespace of configmap uses.")
	system.Conf.InitFlags(fs)
	c.StatesInformerConf.InitFlags(fs)
	c.CollectorConf.InitFlags(fs)
	c.MetricCacheConf.InitFlags(fs)
	c.QOSManagerConf.InitFlags(fs)
	resourceexecutor.Conf.InitFlags(fs)
}

func (c *Configuration) InitKubeConfigForKoordlet(kubeAPIQPS float64, kubeAPIBurst int) error {
	cfg, err := config.GetConfig()
	if err != nil {
		return err
	}
	cfg.UserAgent = "koordlet"
	cfg.QPS = float32(kubeAPIQPS)
	cfg.Burst = kubeAPIBurst
	c.KubeRestConf = cfg
	return nil
}
