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

package frameworkext

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	schedconfig "k8s.io/kubernetes/pkg/scheduler/apis/config"
	"k8s.io/kubernetes/pkg/scheduler/framework"

	koordinatorclientset "github.com/clay-wangzhi/koordinator/pkg/client/clientset/versioned"
	koordinatorinformers "github.com/clay-wangzhi/koordinator/pkg/client/informers/externalversions"
)

// ExtendedHandle extends the k8s scheduling framework Handle interface
// to facilitate plugins to access Koordinator's resources and states.
type ExtendedHandle interface {
	framework.Handle
	// Scheduler return the scheduler adapter to support operating with cache and schedulingQueue.
	// NOTE: Plugins do not acquire a dispatcher instance during plugin initialization,
	// nor are they allowed to hold the object within the plugin object.
	Scheduler() Scheduler
	KoordinatorClientSet() koordinatorclientset.Interface
	KoordinatorSharedInformerFactory() koordinatorinformers.SharedInformerFactory
	// RegisterErrorHandlerFilters supports registering custom PreErrorHandlerFilter and PostErrorHandlerFilter to intercept scheduling errors.
	// If PreErrorHandlerFilter returns true, the k8s scheduler's default error handler and other handlers will not be called.
	// After handling scheduling errors, will execute PostErrorHandlerFilter, and if return true, other custom handlers will not be called.
	RegisterErrorHandlerFilters(preFilter PreErrorHandlerFilter, afterFilter PostErrorHandlerFilter)
	RegisterForgetPodHandler(handler ForgetPodHandler)
	ForgetPod(logger klog.Logger, pod *corev1.Pod) error
}

// FrameworkExtender extends the K8s Scheduling Framework interface to provide more extension methods to support Koordinator.
type FrameworkExtender interface {
	framework.Framework
	ExtendedHandle

	SetConfiguredPlugins(plugins *schedconfig.Plugins)

	RunResizePod(ctx context.Context, cycleState *framework.CycleState, pod *corev1.Pod, nodeName string) *framework.Status
}

// SchedulingTransformer is the parent type for all the custom transformer plugins.
type SchedulingTransformer interface {
	Name() string
}

// PreFilterTransformer is executed before and after PreFilter.
type PreFilterTransformer interface {
	SchedulingTransformer
	// BeforePreFilter If there is a change to the incoming Pod, it needs to be modified after DeepCopy and returned.
	BeforePreFilter(ctx context.Context, cycleState *framework.CycleState, pod *corev1.Pod) (*corev1.Pod, bool, *framework.Status)
	// AfterPreFilter is executed after PreFilter.
	// There is a chance to trigger the correction of the State data of each plugin after the PreFilter.
	AfterPreFilter(ctx context.Context, cycleState *framework.CycleState, pod *corev1.Pod) *framework.Status
}

// FilterTransformer is executed before Filter.
type FilterTransformer interface {
	SchedulingTransformer
	BeforeFilter(ctx context.Context, cycleState *framework.CycleState, pod *corev1.Pod, nodeInfo *framework.NodeInfo) (*corev1.Pod, *framework.NodeInfo, bool, *framework.Status)
}

// ScoreTransformer is executed before Score.
type ScoreTransformer interface {
	SchedulingTransformer
	BeforeScore(ctx context.Context, cycleState *framework.CycleState, pod *corev1.Pod, nodes []*corev1.Node) (*corev1.Pod, []*corev1.Node, bool, *framework.Status)
}

const (
	// MaxReservationScore is the maximum score a ReservationScorePlugin plugin is expected to return.
	MaxReservationScore int64 = 100

	// MinReservationScore is the minimum score a ReservationScorePlugin plugin is expected to return.
	MinReservationScore int64 = 0
)

// ReservationScore is a struct with reservation name and score.
type ReservationScore struct {
	Name      string
	Namespace string
	UID       types.UID
	Score     int64
}

// ResizePodPlugin is an interface that resize the pod resource spec after reserve.
// If you want to use the feature, must enable the feature gate ResizePod=true
type ResizePodPlugin interface {
	ResizePod(ctx context.Context, cycleState *framework.CycleState, pod *corev1.Pod, nodeName string) *framework.Status
}

// PreBindExtensions is an extension to PreBind, which supports converting multiple modifications to the same object into a Patch operation.
// It supports configuring multiple plugin instances. A certain instance can be skipped if it does not need to be processed.
// Once a plugin instance returns success or failure, the process ends.
type PreBindExtensions interface {
	framework.Plugin
	ApplyPatch(ctx context.Context, cycleState *framework.CycleState, originalObj, modifiedObj metav1.Object) *framework.Status
}

type ForgetPodHandler func(pod *corev1.Pod)
