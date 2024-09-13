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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	schedconfig "k8s.io/kubernetes/pkg/scheduler/apis/config"

	"github.com/clay-wangzhi/koordinator/apis/extension"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LoadAwareSchedulingArgs holds arguments used to configure the LoadAwareScheduling plugin.
type LoadAwareSchedulingArgs struct {
	metav1.TypeMeta

	// FilterExpiredNodeMetrics indicates whether to filter nodes where koordlet fails to update NodeMetric.
	// Deprecated: NodeMetric should always be checked for expiration.
	FilterExpiredNodeMetrics *bool
	// NodeMetricExpirationSeconds indicates the NodeMetric expiration in seconds.
	// When NodeMetrics expired, the node is considered abnormal.
	// Default is 180 seconds.
	NodeMetricExpirationSeconds *int64
	// EnableScheduleWhenNodeMetricsExpired Indicates whether nodes with expired nodeMetrics are allowed to schedule pods.
	EnableScheduleWhenNodeMetricsExpired *bool
	// ResourceWeights indicates the weights of resources.
	// The weights of CPU and Memory are both 1 by default.
	ResourceWeights map[corev1.ResourceName]int64
	// UsageThresholds indicates the resource utilization threshold of the whole machine.
	// The default for CPU is 65%, and the default for memory is 95%.
	UsageThresholds map[corev1.ResourceName]int64
	// ProdUsageThresholds indicates the resource utilization threshold of Prod Pods compared to the whole machine.
	// Not enabled by default
	ProdUsageThresholds map[corev1.ResourceName]int64
	// ScoreAccordingProdUsage controls whether to score according to the utilization of Prod Pod
	ScoreAccordingProdUsage bool
	// Estimator indicates the expected Estimator to use
	Estimator string
	// EstimatedScalingFactors indicates the factor when estimating resource usage.
	// The default value of CPU is 85%, and the default value of Memory is 70%.
	EstimatedScalingFactors map[corev1.ResourceName]int64
	// Aggregated supports resource utilization filtering and scoring based on percentile statistics
	Aggregated *LoadAwareSchedulingAggregatedArgs
}

type LoadAwareSchedulingAggregatedArgs struct {
	// UsageThresholds indicates the resource utilization threshold of the machine based on percentile statistics
	UsageThresholds map[corev1.ResourceName]int64
	// UsageAggregationType indicates the percentile type of the machine's utilization when filtering
	// If enabled, only one of the slov1alpha1.AggregationType definitions can be used.
	UsageAggregationType extension.AggregationType
	// UsageAggregatedDuration indicates the statistical period of the percentile of the machine's utilization when filtering
	// If no specific period is set, the maximum period recorded by NodeMetrics will be used by default.
	UsageAggregatedDuration metav1.Duration

	// ScoreAggregationType indicates the percentile type of the machine's utilization when scoring
	// If enabled, only one of the slov1alpha1.AggregationType definitions can be used.
	ScoreAggregationType extension.AggregationType
	// ScoreAggregatedDuration indicates the statistical period of the percentile of Prod Pod's utilization when scoring
	// If no specific period is set, the maximum period recorded by NodeMetrics will be used by default.
	ScoreAggregatedDuration metav1.Duration
}

// ScoringStrategyType is a "string" type.
type ScoringStrategyType string

const (
	// MostAllocated strategy favors node with the least amount of available resource
	MostAllocated ScoringStrategyType = "MostAllocated"
	// BalancedAllocation strategy favors nodes with balanced resource usage rate
	BalancedAllocation ScoringStrategyType = "BalancedAllocation"
	// LeastAllocated strategy favors node with the most amount of available resource
	LeastAllocated ScoringStrategyType = "LeastAllocated"
)

// ScoringStrategy define ScoringStrategyType for the plugin
type ScoringStrategy struct {
	// Type selects which strategy to run.
	Type ScoringStrategyType

	// Resources a list of pairs <resource, weight> to be considered while scoring
	// allowed weights start from 1.
	Resources []schedconfig.ResourceSpec
}
