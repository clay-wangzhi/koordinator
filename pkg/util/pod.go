package util

import (
	corev1 "k8s.io/api/core/v1"
)

func IsPodTerminated(pod *corev1.Pod) bool {
	return pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed
}