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

package nodemetric

import (
	"fmt"

	"github.com/clay-wangzhi/koordinator/apis/configuration"
	slov1alpha1 "github.com/clay-wangzhi/koordinator/apis/slo/v1alpha1"
	"github.com/clay-wangzhi/koordinator/pkg/util/sloconfig"
)

func getNodeMetricCollectPolicy(strategy *configuration.ColocationStrategy) (*slov1alpha1.NodeMetricCollectPolicy, error) {
	if strategy == nil {
		return nil, fmt.Errorf("failed to find satisfied strategy")
	}

	if !sloconfig.IsColocationStrategyValid(strategy) {
		return nil, fmt.Errorf("invalid colocationConfig")
	}

	if strategy.Enable == nil || !*strategy.Enable {
		return nil, fmt.Errorf("colocationConfig disabled")
	}

	collectPolicy := &slov1alpha1.NodeMetricCollectPolicy{
		AggregateDurationSeconds: strategy.MetricAggregateDurationSeconds,
		ReportIntervalSeconds:    strategy.MetricReportIntervalSeconds,
		NodeAggregatePolicy:      strategy.MetricAggregatePolicy,
		NodeMemoryCollectPolicy:  strategy.MetricMemoryCollectPolicy,
	}
	return collectPolicy, nil
}
