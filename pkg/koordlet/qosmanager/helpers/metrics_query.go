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

package helpers

import (
	"fmt"
	"time"

	"github.com/clay-wangzhi/cfs-quota-burst/pkg/koordlet/metriccache"
)

func CollectNodeMetrics(metricCache metriccache.MetricCache, start, end time.Time, queryMeta metriccache.MetricMeta) (metriccache.AggregateResult, error) {
	querier, err := metricCache.Querier(start, end)
	if err != nil {
		return nil, err
	}

	aggregateResult := metriccache.DefaultAggregateResultFactory.New(queryMeta)
	if err := querier.QueryAndClose(queryMeta, nil, aggregateResult); err != nil {
		return nil, err
	}
	return aggregateResult, nil
}

func GenerateQueryParamsAvg(windowDuration time.Duration) *metriccache.QueryParam {
	end := time.Now()
	start := end.Add(-windowDuration)
	queryParam := &metriccache.QueryParam{
		Aggregate: metriccache.AggregationTypeAVG,
		Start:     &start,
		End:       &end,
	}
	return queryParam
}

func CollectContainerThrottledMetric(metricCache metriccache.MetricCache, containerID *string,
	metricCollectInterval time.Duration) (metriccache.AggregateResult, error) {
	if containerID == nil {
		return nil, fmt.Errorf("container is nil")
	}

	queryEndTime := time.Now()
	queryStartTime := queryEndTime.Add(-metricCollectInterval * 2)
	querier, err := metricCache.Querier(queryStartTime, queryEndTime)
	if err != nil {
		return nil, err
	}

	queryParam := metriccache.MetricPropertiesFunc.Container(*containerID)
	queryMeta, err := metriccache.ContainerCPUThrottledMetric.BuildQueryMeta(queryParam)
	if err != nil {
		return nil, err
	}

	aggregateResult := metriccache.DefaultAggregateResultFactory.New(queryMeta)
	if err := querier.QueryAndClose(queryMeta, nil, aggregateResult); err != nil {
		return nil, err
	}

	return aggregateResult, nil
}
