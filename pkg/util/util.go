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

package util

import (
	"context"
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	clientset "k8s.io/client-go/kubernetes"

	apimachinerytypes "k8s.io/apimachinery/pkg/types"

	"k8s.io/klog/v2"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/retry"
)

func MinInt64(i, j int64) int64 {
	if i < j {
		return i
	}
	return j
}

func MaxInt64(i, j int64) int64 {
	if i > j {
		return i
	}
	return j
}

func GetPodKey(pod *corev1.Pod) string {
	return fmt.Sprintf("%v/%v", pod.GetNamespace(), pod.GetName())
}

func RetryOnConflictOrTooManyRequests(fn func() error) error {
	return retry.OnError(retry.DefaultBackoff, func(err error) bool {
		return errors.IsConflict(err) || errors.IsTooManyRequests(err)
	}, fn)
}

func PatchPod(ctx context.Context, clientset clientset.Interface, oldPod, newPod *corev1.Pod) (*corev1.Pod, error) {
	// generate patch bytes for the update
	patchBytes, err := GeneratePodPatch(oldPod, newPod)
	if err != nil {
		klog.V(5).InfoS("failed to generate pod patch", "pod", klog.KObj(oldPod), "err", err)
		return nil, err
	}
	if string(patchBytes) == "{}" { // nothing to patch
		return oldPod, nil
	}

	// patch with pod client
	patched, err := clientset.CoreV1().Pods(oldPod.Namespace).
		Patch(ctx, oldPod.Name, apimachinerytypes.StrategicMergePatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		klog.V(5).InfoS("failed to patch pod", "pod", klog.KObj(oldPod), "patch", string(patchBytes), "err", err)
		return nil, err
	}
	klog.V(6).InfoS("successfully patch pod", "pod", klog.KObj(oldPod), "patch", string(patchBytes))
	return patched, nil
}

func GeneratePodPatch(oldPod, newPod *corev1.Pod) ([]byte, error) {
	oldData, err := json.Marshal(oldPod)
	if err != nil {
		return nil, err
	}

	newData, err := json.Marshal(newPod)
	if err != nil {
		return nil, err
	}
	return strategicpatch.CreateTwoWayMergePatch(oldData, newData, &corev1.Pod{})
}
