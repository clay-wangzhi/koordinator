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

package v1beta3

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	schedconfigv1beta3 "k8s.io/kube-scheduler/config/v1beta3"

	"github.com/clay-wangzhi/koordinator/apis/extension"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LoadAwareSchedulingArgs holds arguments used to configure the LoadAwareScheduling plugin.
type LoadAwareSchedulingArgs struct {
	metav1.TypeMeta `json:",inline"`

	// FilterExpiredNodeMetrics indicates whether to filter nodes where koordlet fails to update NodeMetric.
	FilterExpiredNodeMetrics *bool `json:"filterExpiredNodeMetrics,omitempty"`
	// NodeMetricExpirationSeconds indicates the NodeMetric expiration in seconds.
	// When NodeMetrics expired, the node is considered abnormal.
	// Default is 180 seconds.
	NodeMetricExpirationSeconds *int64 `json:"nodeMetricExpirationSeconds,omitempty"`
	// EnableScheduleWhenNodeMetricsExpired Indicates whether nodes with expired nodeMetrics are allowed to schedule pods.
	EnableScheduleWhenNodeMetricsExpired *bool `json:"enableScheduleWhenNodeMetricsExpired,omitempty"`
	// ResourceWeights indicates the weights of resources.
	// The weights of CPU and Memory are both 1 by default.
	ResourceWeights map[corev1.ResourceName]int64 `json:"resourceWeights,omitempty"`
	// UsageThresholds indicates the resource utilization threshold of the whole machine.
	// The default for CPU is 65%, and the default for memory is 95%.
	UsageThresholds map[corev1.ResourceName]int64 `json:"usageThresholds,omitempty"`
	// ProdUsageThresholds indicates the resource utilization threshold of Prod Pods compared to the whole machine.
	// Not enabled by default
	ProdUsageThresholds map[corev1.ResourceName]int64 `json:"prodUsageThresholds,omitempty"`
	// ScoreAccordingProdUsage controls whether to score according to the utilization of Prod Pod
	ScoreAccordingProdUsage *bool `json:"scoreAccordingProdUsage,omitempty"`
	// Estimator indicates the expected Estimator to use
	Estimator string `json:"estimator,omitempty"`
	// EstimatedScalingFactors indicates the factor when estimating resource usage.
	// The default value of CPU is 85%, and the default value of Memory is 70%.
	EstimatedScalingFactors map[corev1.ResourceName]int64 `json:"estimatedScalingFactors,omitempty"`
	// Aggregated supports resource utilization filtering and scoring based on percentile statistics
	Aggregated *LoadAwareSchedulingAggregatedArgs `json:"aggregated,omitempty"`
}

type LoadAwareSchedulingAggregatedArgs struct {
	// UsageThresholds indicates the resource utilization threshold of the machine based on percentile statistics
	UsageThresholds map[corev1.ResourceName]int64 `json:"usageThresholds,omitempty"`
	// UsageAggregationType indicates the percentile type of the machine's utilization when filtering
	UsageAggregationType extension.AggregationType `json:"usageAggregationType,omitempty"`
	// UsageAggregatedDuration indicates the statistical period of the percentile of the machine's utilization when filtering
	UsageAggregatedDuration *metav1.Duration `json:"usageAggregatedDuration,omitempty"`

	// ScoreAggregationType indicates the percentile type of the machine's utilization when scoring
	ScoreAggregationType extension.AggregationType `json:"scoreAggregationType,omitempty"`
	// ScoreAggregatedDuration indicates the statistical period of the percentile of Prod Pod's utilization when scoring
	ScoreAggregatedDuration *metav1.Duration `json:"scoreAggregatedDuration,omitempty"`
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
	Type ScoringStrategyType `json:"type,omitempty"`

	// Resources a list of pairs <resource, weight> to be considered while scoring
	// allowed weights start from 1.
	Resources []schedconfigv1beta3.ResourceSpec `json:"resources,omitempty"`
}
