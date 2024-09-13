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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	schedconfig "k8s.io/kubernetes/pkg/scheduler/apis/config"
	"k8s.io/kubernetes/pkg/scheduler/framework"

	koordinatorclientset "github.com/clay-wangzhi/koordinator/pkg/client/clientset/versioned"
	koordinatorinformers "github.com/clay-wangzhi/koordinator/pkg/client/informers/externalversions"
)

var _ FrameworkExtender = &frameworkExtenderImpl{}

type frameworkExtenderImpl struct {
	framework.Framework
	*errorHandlerDispatcher
	forgetPodHandlers []ForgetPodHandler

	schedulerFn       func() Scheduler
	configuredPlugins *schedconfig.Plugins
	monitor           *SchedulerMonitor

	koordinatorClientSet             koordinatorclientset.Interface
	koordinatorSharedInformerFactory koordinatorinformers.SharedInformerFactory

	preFilterTransformers map[string]PreFilterTransformer
	filterTransformers    map[string]FilterTransformer
	scoreTransformers     map[string]ScoreTransformer

	resizePodPlugins         []ResizePodPlugin
	preBindExtensionsPlugins map[string]PreBindExtensions
}

func NewFrameworkExtender(f *FrameworkExtenderFactory, fw framework.Framework) FrameworkExtender {
	schedulerFn := func() Scheduler {
		return f.Scheduler()
	}

	frameworkExtender := &frameworkExtenderImpl{
		Framework:                        fw,
		errorHandlerDispatcher:           f.errorHandlerDispatcher,
		schedulerFn:                      schedulerFn,
		monitor:                          f.monitor,
		koordinatorClientSet:             f.KoordinatorClientSet(),
		koordinatorSharedInformerFactory: f.koordinatorSharedInformerFactory,
		preFilterTransformers:            map[string]PreFilterTransformer{},
		filterTransformers:               map[string]FilterTransformer{},
		scoreTransformers:                map[string]ScoreTransformer{},
		preBindExtensionsPlugins:         map[string]PreBindExtensions{},
	}
	return frameworkExtender
}

func (ext *frameworkExtenderImpl) updateTransformer(transformers ...SchedulingTransformer) {
	for _, transformer := range transformers {
		preFilterTransformer, ok := transformer.(PreFilterTransformer)
		if ok {
			ext.preFilterTransformers[transformer.Name()] = preFilterTransformer
			klog.V(4).InfoS("framework extender got scheduling transformer registered", "preFilter", preFilterTransformer.Name())
		}
		filterTransformer, ok := transformer.(FilterTransformer)
		if ok {
			ext.filterTransformers[transformer.Name()] = filterTransformer
			klog.V(4).InfoS("framework extender got scheduling transformer registered", "filter", filterTransformer.Name())
		}
		scoreTransformer, ok := transformer.(ScoreTransformer)
		if ok {
			ext.scoreTransformers[transformer.Name()] = scoreTransformer
			klog.V(4).InfoS("framework extender got scheduling transformer registered", "score", scoreTransformer.Name())
		}
	}
}

func (ext *frameworkExtenderImpl) updatePlugins(pl framework.Plugin) {
	if transformer, ok := pl.(SchedulingTransformer); ok {
		ext.updateTransformer(transformer)
	}
	// TODO(joseph): In the future, use only the default ReservationNominator
	if r, ok := pl.(ResizePodPlugin); ok {
		ext.resizePodPlugins = append(ext.resizePodPlugins, r)
	}
	if p, ok := pl.(PreBindExtensions); ok {
		ext.preBindExtensionsPlugins[p.Name()] = p
	}
}

func (ext *frameworkExtenderImpl) SetConfiguredPlugins(plugins *schedconfig.Plugins) {
	ext.configuredPlugins = plugins
}

func (ext *frameworkExtenderImpl) KoordinatorClientSet() koordinatorclientset.Interface {
	return ext.koordinatorClientSet
}

func (ext *frameworkExtenderImpl) KoordinatorSharedInformerFactory() koordinatorinformers.SharedInformerFactory {
	return ext.koordinatorSharedInformerFactory
}

// Scheduler return the scheduler adapter to support operating with cache and schedulingQueue.
// NOTE: Plugins do not acquire a dispatcher instance during plugin initialization,
// nor are they allowed to hold the object within the plugin object.
func (ext *frameworkExtenderImpl) Scheduler() Scheduler {
	return ext.schedulerFn()
}

// RunPreFilterPlugins transforms the PreFilter phase of framework with pre-filter transformers.
func (ext *frameworkExtenderImpl) RunPreFilterPlugins(ctx context.Context, cycleState *framework.CycleState, pod *corev1.Pod) (*framework.PreFilterResult, *framework.Status) {
	for _, pl := range ext.configuredPlugins.PreFilter.Enabled {
		transformer := ext.preFilterTransformers[pl.Name]
		if transformer == nil {
			continue
		}
		newPod, transformed, status := transformer.BeforePreFilter(ctx, cycleState, pod)
		if !status.IsSuccess() {
			klog.ErrorS(status.AsError(), "Failed to run BeforePreFilter", "pod", klog.KObj(pod), "plugin", transformer.Name())
			return nil, status
		}
		if transformed {
			klog.V(5).InfoS("BeforePreFilter transformed", "transformer", transformer.Name(), "pod", klog.KObj(pod))
			pod = newPod
		}
	}

	result, status := ext.Framework.RunPreFilterPlugins(ctx, cycleState, pod)
	if !status.IsSuccess() {
		return result, status
	}

	for _, pl := range ext.configuredPlugins.PreFilter.Enabled {
		transformer := ext.preFilterTransformers[pl.Name]
		if transformer == nil {
			continue
		}
		if status := transformer.AfterPreFilter(ctx, cycleState, pod); !status.IsSuccess() {
			klog.ErrorS(status.AsError(), "Failed to run AfterPreFilter", "pod", klog.KObj(pod), "plugin", transformer.Name())
			return nil, status
		}
	}
	return result, nil
}

// RunFilterPluginsWithNominatedPods transforms the Filter phase of framework with filter transformers.
// We don't transform RunFilterPlugins since framework's RunFilterPluginsWithNominatedPods just calls its RunFilterPlugins.
func (ext *frameworkExtenderImpl) RunFilterPluginsWithNominatedPods(ctx context.Context, cycleState *framework.CycleState, pod *corev1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	for _, pl := range ext.configuredPlugins.Filter.Enabled {
		transformer := ext.filterTransformers[pl.Name]
		if transformer == nil {
			continue
		}
		newPod, newNodeInfo, transformed, status := transformer.BeforeFilter(ctx, cycleState, pod, nodeInfo)
		if !status.IsSuccess() {
			klog.ErrorS(status.AsError(), "Failed to run BeforeFilter", "pod", klog.KObj(pod), "plugin", transformer.Name())
			return status
		}
		if transformed {
			klog.V(5).InfoS("BeforeFilter transformed", "transformer", transformer.Name(), "pod", klog.KObj(pod))
			pod = newPod
			nodeInfo = newNodeInfo
		}
	}
	status := ext.Framework.RunFilterPluginsWithNominatedPods(ctx, cycleState, pod, nodeInfo)
	if !status.IsSuccess() && debugFilterFailure {
		klog.Infof("Failed to filter for Pod %q on Node %q, failedPlugin: %s, reason: %s", klog.KObj(pod), klog.KObj(nodeInfo.Node()), status.FailedPlugin(), status.Message())
	}
	return status
}

func (ext *frameworkExtenderImpl) RunPostFilterPlugins(ctx context.Context, state *framework.CycleState, pod *corev1.Pod, filteredNodeStatusMap framework.NodeToStatusMap) (*framework.PostFilterResult, *framework.Status) {
	result, status := ext.Framework.RunPostFilterPlugins(ctx, state, pod, filteredNodeStatusMap)
	return result, status
}

func (ext *frameworkExtenderImpl) RunScorePlugins(ctx context.Context, state *framework.CycleState, pod *corev1.Pod, nodes []*corev1.Node) ([]framework.NodePluginScores, *framework.Status) {
	for _, pl := range ext.configuredPlugins.Score.Enabled {
		transformer := ext.scoreTransformers[pl.Name]
		if transformer == nil {
			continue
		}
		newPod, newNodes, transformed, status := transformer.BeforeScore(ctx, state, pod, nodes)
		if !status.IsSuccess() {
			klog.ErrorS(status.AsError(), "Failed to run BeforeScore", "pod", klog.KObj(pod), "plugin", transformer.Name())
			return nil, status
		}
		if transformed {
			klog.V(5).InfoS("BeforeScore transformed", "transformer", transformer.Name(), "pod", klog.KObj(pod))
			pod = newPod
			nodes = newNodes
		}
	}
	pluginToNodeScores, status := ext.Framework.RunScorePlugins(ctx, state, pod, nodes)
	if status.IsSuccess() && debugTopNScores > 0 {
		debugScores(debugTopNScores, pod, pluginToNodeScores, nodes)
	}
	return pluginToNodeScores, status
}

func (ext *frameworkExtenderImpl) runPreBindExtensionPlugins(ctx context.Context, cycleState *framework.CycleState, originalObj, modifiedObj metav1.Object) *framework.Status {
	plugins := ext.configuredPlugins
	for _, plugin := range plugins.PreBind.Enabled {
		pl := ext.preBindExtensionsPlugins[plugin.Name]
		if pl == nil {
			continue
		}
		status := pl.ApplyPatch(ctx, cycleState, originalObj, modifiedObj)
		if status != nil && status.Code() == framework.Skip {
			continue
		}
		if !status.IsSuccess() {
			err := status.AsError()
			klog.ErrorS(err, "Failed running PreBindExtension plugin", "plugin", pl.Name(), "pod", klog.KObj(originalObj))
			return framework.AsStatus(fmt.Errorf("running PreBindExtension plugin %q: %w", pl.Name(), err))
		}
		return status
	}
	return nil
}

func (ext *frameworkExtenderImpl) RunPostBindPlugins(ctx context.Context, state *framework.CycleState, pod *corev1.Pod, nodeName string) {
	if ext.monitor != nil {
		defer ext.monitor.Complete(pod)
	}
	ext.Framework.RunPostBindPlugins(ctx, state, pod, nodeName)
}

func (ext *frameworkExtenderImpl) RegisterForgetPodHandler(handler ForgetPodHandler) {
	ext.forgetPodHandlers = append(ext.forgetPodHandlers, handler)
}

func (ext *frameworkExtenderImpl) ForgetPod(logger klog.Logger, pod *corev1.Pod) error {
	if err := ext.Scheduler().GetCache().ForgetPod(logger, pod); err != nil {
		return err
	}
	for _, handler := range ext.forgetPodHandlers {
		handler(pod)
	}
	return nil
}

func (ext *frameworkExtenderImpl) RunResizePod(ctx context.Context, cycleState *framework.CycleState, pod *corev1.Pod, nodeName string) *framework.Status {
	for _, pl := range ext.resizePodPlugins {
		status := pl.ResizePod(ctx, cycleState, pod, nodeName)
		if !status.IsSuccess() {
			return status
		}
	}
	return nil
}

const podAssumedStateKey = "koordinator.sh/assumed"

type assumedState struct{}

func (s *assumedState) Clone() framework.StateData {
	return s
}

func markPodAssumed(cycleState *framework.CycleState) {
	cycleState.Write(podAssumedStateKey, &assumedState{})
}

func isPodAssumed(cycleState *framework.CycleState) bool {
	assumed, _ := cycleState.Read(podAssumedStateKey)
	return assumed != nil
}
