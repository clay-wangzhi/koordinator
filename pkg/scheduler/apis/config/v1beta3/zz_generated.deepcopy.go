//go:build !ignore_autogenerated
// +build !ignore_autogenerated

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

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1beta3

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	configv1beta3 "k8s.io/kube-scheduler/config/v1beta3"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LoadAwareSchedulingAggregatedArgs) DeepCopyInto(out *LoadAwareSchedulingAggregatedArgs) {
	*out = *in
	if in.UsageThresholds != nil {
		in, out := &in.UsageThresholds, &out.UsageThresholds
		*out = make(map[v1.ResourceName]int64, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.UsageAggregatedDuration != nil {
		in, out := &in.UsageAggregatedDuration, &out.UsageAggregatedDuration
		*out = new(metav1.Duration)
		**out = **in
	}
	if in.ScoreAggregatedDuration != nil {
		in, out := &in.ScoreAggregatedDuration, &out.ScoreAggregatedDuration
		*out = new(metav1.Duration)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LoadAwareSchedulingAggregatedArgs.
func (in *LoadAwareSchedulingAggregatedArgs) DeepCopy() *LoadAwareSchedulingAggregatedArgs {
	if in == nil {
		return nil
	}
	out := new(LoadAwareSchedulingAggregatedArgs)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LoadAwareSchedulingArgs) DeepCopyInto(out *LoadAwareSchedulingArgs) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	if in.FilterExpiredNodeMetrics != nil {
		in, out := &in.FilterExpiredNodeMetrics, &out.FilterExpiredNodeMetrics
		*out = new(bool)
		**out = **in
	}
	if in.NodeMetricExpirationSeconds != nil {
		in, out := &in.NodeMetricExpirationSeconds, &out.NodeMetricExpirationSeconds
		*out = new(int64)
		**out = **in
	}
	if in.EnableScheduleWhenNodeMetricsExpired != nil {
		in, out := &in.EnableScheduleWhenNodeMetricsExpired, &out.EnableScheduleWhenNodeMetricsExpired
		*out = new(bool)
		**out = **in
	}
	if in.ResourceWeights != nil {
		in, out := &in.ResourceWeights, &out.ResourceWeights
		*out = make(map[v1.ResourceName]int64, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.UsageThresholds != nil {
		in, out := &in.UsageThresholds, &out.UsageThresholds
		*out = make(map[v1.ResourceName]int64, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.ProdUsageThresholds != nil {
		in, out := &in.ProdUsageThresholds, &out.ProdUsageThresholds
		*out = make(map[v1.ResourceName]int64, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.ScoreAccordingProdUsage != nil {
		in, out := &in.ScoreAccordingProdUsage, &out.ScoreAccordingProdUsage
		*out = new(bool)
		**out = **in
	}
	if in.EstimatedScalingFactors != nil {
		in, out := &in.EstimatedScalingFactors, &out.EstimatedScalingFactors
		*out = make(map[v1.ResourceName]int64, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Aggregated != nil {
		in, out := &in.Aggregated, &out.Aggregated
		*out = new(LoadAwareSchedulingAggregatedArgs)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LoadAwareSchedulingArgs.
func (in *LoadAwareSchedulingArgs) DeepCopy() *LoadAwareSchedulingArgs {
	if in == nil {
		return nil
	}
	out := new(LoadAwareSchedulingArgs)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *LoadAwareSchedulingArgs) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ScoringStrategy) DeepCopyInto(out *ScoringStrategy) {
	*out = *in
	if in.Resources != nil {
		in, out := &in.Resources, &out.Resources
		*out = make([]configv1beta3.ResourceSpec, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ScoringStrategy.
func (in *ScoringStrategy) DeepCopy() *ScoringStrategy {
	if in == nil {
		return nil
	}
	out := new(ScoringStrategy)
	in.DeepCopyInto(out)
	return out
}