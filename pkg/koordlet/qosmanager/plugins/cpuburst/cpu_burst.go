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

package cpuburst

import (
	"encoding/json"
	"fmt"

	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"

	"github.com/clay-wangzhi/cfs-quota-burst/pkg/koordlet/metriccache"
	"github.com/clay-wangzhi/cfs-quota-burst/pkg/koordlet/qosmanager/framework"
	"github.com/clay-wangzhi/cfs-quota-burst/pkg/koordlet/qosmanager/helpers"
	"github.com/clay-wangzhi/cfs-quota-burst/pkg/koordlet/resourceexecutor"
	"github.com/clay-wangzhi/cfs-quota-burst/pkg/koordlet/statesinformer"
	koordletutil "github.com/clay-wangzhi/cfs-quota-burst/pkg/koordlet/util"
	"github.com/clay-wangzhi/cfs-quota-burst/pkg/koordlet/util/system"
	"github.com/clay-wangzhi/cfs-quota-burst/pkg/util"
)

const (
	CPUBurstName = "CPUBurst"

	cfsIncreaseStep = 1.2
	cfsDecreaseStep = 0.8

	sharePoolCoolingThresholdRatio = 0.9

	cpuThresholdPercentForLimiterConsumeTokens = 100
	cpuThresholdPercentForLimiterSavingTokens  = 60
)

// cfsOperation is used for CFSQuotaBurst strategy
type cfsOperation int64

const (
	cfsScaleUp cfsOperation = iota
	cfsScaleDown
	cfsRemain
	cfsReset
)

func (o cfsOperation) String() string {
	switch o {
	case cfsScaleUp:
		return "cfsScaleUp"
	case cfsScaleDown:
		return "cfsScaleDown"
	case cfsRemain:
		return "cfsRemain"
	case cfsReset:
		return "cfsReset"
	default:
		return fmt.Sprintf("unrecognized(%d)", o)
	}
}

// nodeStateForBurst depends on cpu-share-pool usage, used for CFSBurstStrategy
type nodeStateForBurst int64

const (
	// cpu-share-pool usage >= threshold
	nodeBurstOverload nodeStateForBurst = iota
	// threshold * 0.9 <= cpu-share-pool usage < threshold
	nodeBurstCooling nodeStateForBurst = iota
	// cpu-share-pool usage < threshold * 0.9
	nodeBurstIdle nodeStateForBurst = iota
	// cpu-share-pool is unknown
	nodeBurstUnknown nodeStateForBurst = iota
)

func (s nodeStateForBurst) String() string {
	switch s {
	case nodeBurstOverload:
		return "nodeBurstOverload"
	case nodeBurstCooling:
		return "nodeBurstCooling"
	case nodeBurstIdle:
		return "nodeBurstIdle"
	case nodeBurstUnknown:
		return "nodeBurstUnknown"
	default:
		return fmt.Sprintf("unrecognized(%d)", s)
	}
}

var _ framework.QOSStrategy = &cpuBurst{}

type CPUBurstPolicy string

const (
	// CPUBurstNone disables cpu burst policy
	CPUBurstNone CPUBurstPolicy = "none"
	// CPUBurstOnly enables cpu burst policy by setting cpu.cfs_burst_us
	CPUBurstOnly CPUBurstPolicy = "cpuBurstOnly"
	// CFSQuotaBurstOnly enables cfs quota burst policy by scale up cpu.cfs_quota_us if pod throttled
	CFSQuotaBurstOnly CPUBurstPolicy = "cfsQuotaBurstOnly"
	// CPUBurstAuto enables both
	CPUBurstAuto CPUBurstPolicy = "auto"
)

type CPUBurstConfig struct {
	Policy CPUBurstPolicy `json:"policy,omitempty"`
	// cpu burst percentage for setting cpu.cfs_burst_us, legal range: [0, 10000], default as 1000 (1000%)
	// +kubebuilder:validation:Maximum=10000
	// +kubebuilder:validation:Minimum=0
	CPUBurstPercent *int64 `json:"cpuBurstPercent,omitempty" validate:"omitempty,min=1,max=10000"`
	// pod cfs quota scale up ceil percentage, default = 300 (300%)
	CFSQuotaBurstPercent *int64 `json:"cfsQuotaBurstPercent,omitempty" validate:"omitempty,min=100"`
	// specifies a period of time for pod can use at burst, default = -1 (unlimited)
	CFSQuotaBurstPeriodSeconds *int64 `json:"cfsQuotaBurstPeriodSeconds,omitempty" validate:"omitempty,min=-1"`
}

type CPUBurstStrategy struct {
	CPUBurstConfig `json:",inline"`
	// scale down cfs quota if node cpu overload, default = 50
	SharePoolThresholdPercent *int64 `json:"sharePoolThresholdPercent,omitempty" validate:"omitempty,min=0,max=100"`
}

type cpuBurst struct {
	reconcileInterval     time.Duration
	metricCollectInterval time.Duration
	statesInformer        statesinformer.StatesInformer
	metricCache           metriccache.MetricCache
	executor              resourceexecutor.ResourceUpdateExecutor
	cgroupReader          resourceexecutor.CgroupReader
	nodeCPUBurstStrategy  *CPUBurstStrategy
}

func New(opt *framework.Options) framework.QOSStrategy {
	return &cpuBurst{
		reconcileInterval:     time.Duration(opt.Config.ReconcileIntervalSeconds) * time.Second,
		metricCollectInterval: opt.MetricAdvisorConfig.CollectResUsedInterval,
		statesInformer:        opt.StatesInformer,
		metricCache:           opt.MetricCache,
		executor:              resourceexecutor.NewResourceUpdateExecutor(),
		cgroupReader:          opt.CgroupReader,
	}
}

func (b *cpuBurst) Enabled() bool {
	return true && b.reconcileInterval > 0
}

func (b *cpuBurst) Setup(ctx *framework.Context) {

}

func (b *cpuBurst) Run(stopCh <-chan struct{}) {
	b.init(stopCh)
	go wait.Until(b.start, b.reconcileInterval, stopCh)
}

func (b *cpuBurst) init(stopCh <-chan struct{}) {
	b.executor.Run(stopCh)
}

func (b *cpuBurst) start() {
	klog.V(5).Infof("start cpu burst strategy")
	var sharePoolThresholdPercent int64 = 50
	podsMeta := b.statesInformer.GetAllPods()
	cfsCm := b.statesInformer.GetCfsCM()

	// get node state by node share pool usage
	nodeState := b.getNodeStateForBurst(sharePoolThresholdPercent, podsMeta)

	for _, podMeta := range podsMeta {
		if podMeta == nil || podMeta.Pod == nil {
			klog.Warningf("podMeta is illegal, detail %v", podMeta)
			continue
		}
		if podMeta.Pod.Status.Phase != corev1.PodPending && podMeta.Pod.Status.Phase != corev1.PodRunning {
			// ignore pods that status.phase is not pending or running,
			// because the other pods(include succeed,failed and unknown) do not have any containers running
			// and therefore do not have a cgroup file,
			// so there is no need to deal with it
			continue
		}

		// merge burst config from pod and node
		cpuBurstCfg := genPodBurstConfig(podMeta.Pod, cfsCm)
		if cpuBurstCfg == nil {
			klog.Warningf("pod %v/%v burst config illegal, burst config %v",
				podMeta.Pod.Namespace, podMeta.Pod.Name, cpuBurstCfg)
			continue
		}
		klog.V(5).Infof("get pod %v/%v cpu burst config: %v", podMeta.Pod.Namespace, podMeta.Pod.Name, cpuBurstCfg)
		// scale cpu.cfs_quota_us for pod and containers
		b.applyCFSQuotaBurst(cpuBurstCfg, podMeta, nodeState)
	}
}

// getNodeStateForBurst checks whether node share pool cpu usage beyonds the threshold
// return isOverload, share pool usage ratio and message detail
func (b *cpuBurst) getNodeStateForBurst(sharePoolThresholdPercent int64,
	podsMeta []*statesinformer.PodMeta) nodeStateForBurst {
	overloadMetricDuration := time.Duration(util.MinInt64(int64(b.reconcileInterval*5), int64(10*time.Second)))
	queryParam := helpers.GenerateQueryParamsAvg(overloadMetricDuration)

	queryMeta, err := metriccache.NodeCPUUsageMetric.BuildQueryMeta(nil)
	if err != nil {
		klog.Warningf("get node metric queryMeta failed, error: %v", err)
		return nodeBurstUnknown
	}
	queryResult, err := helpers.CollectNodeMetrics(b.metricCache, *queryParam.Start, *queryParam.End, queryMeta)
	if err != nil {
		klog.Warningf("get node cpu metric failed, error: %v", err)
		return nodeBurstUnknown
	}
	if queryResult.Count() == 0 {
		klog.Warning("node metric is empty during handle cfs burst scale down")
		return nodeBurstUnknown
	}
	nodeCPUUsed, err := queryResult.Value(queryParam.Aggregate)
	if err != nil {
		klog.Warningf("get node cpu metric failed, error: %v", err)
	}

	nodeCPUInfoRaw, exist := b.metricCache.Get(metriccache.NodeCPUInfoKey)
	if !exist {
		klog.Warning("get node cpu info failed : not exist")
		return nodeBurstUnknown
	}
	nodeCPUInfo := nodeCPUInfoRaw.(*metriccache.NodeCPUInfo)
	if nodeCPUInfo == nil {
		klog.Warning("get node cpu info failed : value is nil")
		return nodeBurstUnknown
	}

	nodeCPUCoresTotal := len(nodeCPUInfo.ProcessorInfos)
	nodeCPUCoresUsage := nodeCPUUsed
	// calculate cpu share pool info; for conservative reason, include system usage in share pool
	sharePoolCPUCoresTotal := float64(nodeCPUCoresTotal)
	sharePoolCPUCoresUsage := nodeCPUCoresUsage

	// calculate cpu share pool usage ratio
	sharePoolThresholdRatio := float64(sharePoolThresholdPercent) / 100
	sharePoolCoolingRatio := sharePoolThresholdRatio * sharePoolCoolingThresholdRatio
	sharePoolUsageRatio := 1.0
	if sharePoolCPUCoresTotal > 0 {
		sharePoolUsageRatio = sharePoolCPUCoresUsage / sharePoolCPUCoresTotal
	}
	klog.V(5).Infof("share pool usage / share pool total = [%v/%v] = [%v],  threshold = [%v]",
		sharePoolCPUCoresUsage, sharePoolCPUCoresTotal, sharePoolUsageRatio, sharePoolThresholdRatio)

	// generate node burst state by cpu share pool usage
	var nodeBurstState nodeStateForBurst
	if sharePoolUsageRatio >= sharePoolThresholdRatio {
		nodeBurstState = nodeBurstOverload
	} else if sharePoolCoolingRatio <= sharePoolUsageRatio && sharePoolUsageRatio < sharePoolThresholdRatio {
		nodeBurstState = nodeBurstCooling
	} else { // sharePoolUsageRatio < sharePoolCoolingRatio
		nodeBurstState = nodeBurstIdle
	}
	return nodeBurstState
}

// scale cpu.cfs_quota_us for pod/containers by container throttled state and node state
func (b *cpuBurst) applyCFSQuotaBurst(burstCfg *CPUBurstConfig, podMeta *statesinformer.PodMeta,
	nodeState nodeStateForBurst) {
	pod := podMeta.Pod
	containerMap := make(map[string]*corev1.Container)
	for i := range pod.Spec.Containers {
		container := &pod.Spec.Containers[i]
		containerMap[container.Name] = container
	}

	for i := range pod.Status.ContainerStatuses {
		containerStat := &pod.Status.ContainerStatuses[i]
		container, exist := containerMap[containerStat.Name]
		if !exist || container == nil {
			klog.Warningf("container %s/%s/%s not found in pod spec", pod.Namespace, pod.Name, containerStat.Name)
			continue
		}

		if containerStat.State.Running == nil {
			klog.V(6).Infof("skip container %s/%s/%s, because it is not running", pod.Namespace, pod.Name, containerStat.Name)
			continue
		}

		containerBaseCFS := koordletutil.GetContainerBaseCFSQuota(container)
		if containerBaseCFS <= 0 {
			continue
		}
		containerPath, err := koordletutil.GetContainerCgroupParentDir(podMeta.CgroupDir, containerStat)
		if err != nil {
			klog.Infof("get container %s/%s/%s cgroup path failed, err %v",
				pod.Namespace, pod.Name, containerStat.Name, err)
			continue
		}
		containerCurCFS, err := b.cgroupReader.ReadCPUQuota(containerPath)
		if err != nil && resourceexecutor.IsCgroupDirErr(err) {
			klog.V(5).Infof("get container %s/%s/%s current cfs quota failed, maybe not exist, skip this round, reason %v",
				pod.Namespace, pod.Name, containerStat.Name, err)
			continue
		} else if err != nil {
			klog.Infof("get container %s/%s/%s current cfs quota failed, maybe not exist, skip this round, reason %v",
				pod.Namespace, pod.Name, containerStat.Name, err)
			continue
		}
		containerCeilCFS := containerBaseCFS
		if burstCfg.CFSQuotaBurstPercent != nil && *burstCfg.CFSQuotaBurstPercent > 100 {
			containerCeilCFS = int64(float64(containerBaseCFS) * float64(*burstCfg.CFSQuotaBurstPercent) / 100)
		}

		originOperation := b.genOperationByContainer(burstCfg, pod, container, containerStat)
		klog.V(6).Infof("cfs burst operation for container %v/%v/%v is %v",
			pod.Namespace, pod.Name, containerStat.Name, originOperation)

		changed, finalOperation := changeOperationByNode(nodeState, originOperation)
		if changed {
			klog.Infof("node is in %v state, switch origin scale operation %v to %v",
				nodeState, originOperation.String(), finalOperation.String())
		} else {
			klog.V(5).Infof("node is in %v state, operation %v is same as before %v",
				nodeState, finalOperation.String(), originOperation.String())
		}

		containerTargetCFS := containerCurCFS
		if finalOperation == cfsScaleUp {
			containerTargetCFS = int64(float64(containerCurCFS) * cfsIncreaseStep)
		} else if finalOperation == cfsScaleDown {
			containerTargetCFS = int64(float64(containerCurCFS) * cfsDecreaseStep)
		} else if finalOperation == cfsReset {
			containerTargetCFS = containerBaseCFS
		}
		containerTargetCFS = util.MaxInt64(containerBaseCFS, util.MinInt64(containerTargetCFS, containerCeilCFS))

		if containerTargetCFS == containerCurCFS {
			klog.V(5).Infof("no need to scale for container %v/%v/%v, operation %v, target cfs quota %v",
				pod.Namespace, pod.Name, containerStat.Name, finalOperation, containerTargetCFS)
			continue
		}
		deltaContainerCFS := containerTargetCFS - containerCurCFS
		err = b.applyContainerCFSQuota(podMeta, containerStat, containerCurCFS, deltaContainerCFS)
		if err != nil {
			klog.Infof("scale container %v/%v/%v cfs quota failed, operation %v, delta cfs quota %v, reason %v",
				pod.Namespace, pod.Name, containerStat.Name, finalOperation, deltaContainerCFS, err)
			continue
		}
		// metrics.RecordContainerScaledCFSQuotaUS(pod.Namespace, pod.Name, containerStat.ContainerID, containerStat.Name, float64(containerTargetCFS))
		klog.Infof("scale container %v/%v/%v cfs quota success, operation %v, current cfs %v, target cfs %v",
			pod.Namespace, pod.Name, containerStat.Name, finalOperation, containerCurCFS, containerTargetCFS)
	} // end for containers
}

func (b *cpuBurst) genOperationByContainer(burstCfg *CPUBurstConfig, pod *corev1.Pod,
	container *corev1.Container, containerStat *corev1.ContainerStatus) cfsOperation {

	// allowedByLimiterCfg := b.cfsBurstAllowedByLimiter(burstCfg, container, &containerStat.ContainerID)
	if !cfsQuotaBurstEnabled(burstCfg.Policy) {
		return cfsReset
	}

	containerThrottled, err := helpers.CollectContainerThrottledMetric(b.metricCache, &containerStat.ContainerID, b.metricCollectInterval)
	if err != nil {
		klog.V(4).Infof("failed to get container %s/%s/%s throttled metric, maybe not exist, skip this round, reason %v",
			pod.Namespace, pod.Name, containerStat.Name, err)
		return cfsRemain
	}

	if containerThrottled.Count() == 0 {
		klog.V(4).Infof("container %s/%s/%s throttled metric is empty, skip this round",
			pod.Namespace, pod.Name, containerStat.Name)
		return cfsRemain
	}

	containerThrottledLastValue, err := containerThrottled.Value(metriccache.AggregationTypeLast)
	if err != nil {
		klog.V(4).Infof("failed to get container %s/%s/%s last throttled metric, skip this round, reason %v",
			pod.Namespace, pod.Name, containerStat.Name, err)
		return cfsRemain
	}

	if containerThrottledLastValue > 0 {
		return cfsScaleUp
	}
	klog.V(5).Infof("container %s/%s/%s is not throttled, no need to scale up cfs quota",
		pod.Namespace, pod.Name, containerStat.Name)
	return cfsRemain
}

func (b *cpuBurst) applyContainerCFSQuota(podMeta *statesinformer.PodMeta, containerStat *corev1.ContainerStatus,
	curContaienrCFS, deltaContainerCFS int64) error {
	podDir := podMeta.CgroupDir
	curPodCFS, podPathErr := b.cgroupReader.ReadCPUQuota(podDir)
	if podPathErr != nil {
		return fmt.Errorf("get pod %v/%v current cfs quota failed, error: %v",
			podMeta.Pod.Namespace, podMeta.Pod.Name, podPathErr)
	}
	containerDir, containerPathErr := koordletutil.GetContainerCgroupParentDir(podMeta.CgroupDir, containerStat)
	if containerPathErr != nil {
		return fmt.Errorf("get container %v/%v/%v cgroup path failed, error: %v",
			podMeta.Pod.Namespace, podMeta.Pod.Name, containerStat.Name, containerPathErr)
	}

	updatePodCFSQuota := func() error {
		// no need to adjust pod cpu.cfs_quota if it is already -1
		if curPodCFS <= 0 {
			return nil
		}

		targetPodCFS := curPodCFS + deltaContainerCFS
		podCFSValStr := strconv.FormatInt(targetPodCFS, 10)
		// eventHelper := audit.V(3).Pod(podMeta.Pod.Namespace, podMeta.Pod.Name).Reason("CFSQuotaBurst").Message("update pod CFSQuota: %v", podCFSValStr)
		// updater, _ := resourceexecutor.DefaultCgroupUpdaterFactory.New(system.CPUCFSQuotaName, podDir, podCFSValStr, eventHelper)
		updater, _ := resourceexecutor.DefaultCgroupUpdaterFactory.New(system.CPUCFSQuotaName, podDir, podCFSValStr)
		if _, err := b.executor.Update(true, updater); err != nil {
			return fmt.Errorf("update pod cgroup %v failed, error %v", podMeta.CgroupDir, err)
		}

		return nil
	}

	updateContainerCFSQuota := func() error {
		targetContainerCFS := curContaienrCFS + deltaContainerCFS
		containerCFSValStr := strconv.FormatInt(targetContainerCFS, 10)
		// eventHelper := audit.V(3).Container(containerStat.Name).Reason("CFSQuotaBurst").Message("update container CFSQuota: %v", containerCFSValStr)
		// updater, _ := resourceexecutor.DefaultCgroupUpdaterFactory.New(system.CPUCFSQuotaName, containerDir, containerCFSValStr, eventHelper)
		updater, _ := resourceexecutor.DefaultCgroupUpdaterFactory.New(system.CPUCFSQuotaName, containerDir, containerCFSValStr)
		if _, err := b.executor.Update(true, updater); err != nil {
			return fmt.Errorf("update container cgroup %v failed, reason %v", containerDir, err)
		}

		return nil
	}

	// cfs scale down, order: container->pod
	sortOfUpdateQuota := []func() error{updateContainerCFSQuota, updatePodCFSQuota}
	if deltaContainerCFS > 0 {
		// cfs scale up, order: pod->container
		sortOfUpdateQuota = []func() error{updatePodCFSQuota, updateContainerCFSQuota}
	}

	for _, update := range sortOfUpdateQuota {
		if err := update(); err != nil {
			return err
		}
	}

	return nil
}

const (
	AnnotationPodCPUBurst = "koordinator.sh/" + "cpuBurst"
)

func GetPodCPUBurstConfig(pod *corev1.Pod) (*CPUBurstConfig, error) {
	if pod == nil || pod.Annotations == nil {
		return nil, nil
	}
	annotation, exist := pod.Annotations[AnnotationPodCPUBurst]
	if !exist {
		return nil, nil
	}
	cpuBurst := CPUBurstConfig{}

	err := json.Unmarshal([]byte(annotation), &cpuBurst)
	if err != nil {
		return nil, err
	}
	return &cpuBurst, nil
}

// use node config by default, overlap if pod specify config
func genPodBurstConfig(pod *corev1.Pod, cfsCm *corev1.ConfigMap) *CPUBurstConfig {
	podCPUBurstCfg, err := GetPodCPUBurstConfig(pod)
	if err != nil {
		klog.Infof("parse pod %s/%s cpu burst config failed, reason %v", pod.Namespace, pod.Name, err)
		return nil
	}
	if podCPUBurstCfg != nil {
		return podCPUBurstCfg
	}

	cmData := cfsCm.Data
	if cmData == nil {
		return nil
	}
	labelCPUBurstCfg, exist := cmData["cpu-burst-config"]
	if !exist {
		return nil
	}
	cpuBurst := CPUBurstConfig{}
	err = json.Unmarshal([]byte(labelCPUBurstCfg), &cpuBurst)
	if err != nil {
		return nil
	}
	klog.V(5).Infof("cm config is %v pod label is %v", cmData, pod.Labels)
	for cmKey, cmValue := range cmData {
		for podKey, podValue := range pod.Labels {
			if podKey == cmKey && podValue == cmValue {
				return &cpuBurst
			}
		}
	}

	return nil
}

func cfsQuotaBurstEnabled(burstPolicy CPUBurstPolicy) bool {
	return burstPolicy == CPUBurstAuto || burstPolicy == CFSQuotaBurstOnly
}

func changeOperationByNode(nodeState nodeStateForBurst, originOperation cfsOperation) (bool, cfsOperation) {
	changedOperation := originOperation
	if nodeState == nodeBurstOverload && (originOperation == cfsScaleUp || originOperation == cfsRemain) {
		changedOperation = cfsScaleDown
	} else if (nodeState == nodeBurstCooling || nodeState == nodeBurstUnknown) && originOperation == cfsScaleUp {
		changedOperation = cfsRemain
	}
	return changedOperation != originOperation, changedOperation
}
