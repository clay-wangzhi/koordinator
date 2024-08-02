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

package metricsadvisor

import (
	"github.com/clay-wangzhi/cfs-quota-burst/pkg/koordlet/metricsadvisor/collectors/nodeinfo"
	"github.com/clay-wangzhi/cfs-quota-burst/pkg/koordlet/metricsadvisor/collectors/noderesource"
	"github.com/clay-wangzhi/cfs-quota-burst/pkg/koordlet/metricsadvisor/collectors/podthrottled"
	"github.com/clay-wangzhi/cfs-quota-burst/pkg/koordlet/metricsadvisor/framework"
)

// NOTE: map variables in this file can be overwritten for extension

var (
	collectorPlugins = map[string]framework.CollectorFactory{
		noderesource.CollectorName: noderesource.New,
		nodeinfo.CollectorName:     nodeinfo.New,
		podthrottled.CollectorName: podthrottled.New,
	}
)
