package features

import (
	"fmt"
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
)

func GetContainerImages(pod *corev1.Pod) []string {
	var images []string
	for _, container := range pod.Spec.Containers {
		images = append(images, container.Image)
	}
	return images
}

func GetOwnerKindAndName(pod *corev1.Pod) (string, string) {
	var ownerKind, ownerName string
	for _, ownerReference := range pod.OwnerReferences {
		ownerKind = string(ownerReference.Kind)
		ownerName = ownerReference.Name
		break
	}
	return ownerKind, ownerName
}

func GetLabels(pod *corev1.Pod) []string {
	var labels []string
	for k, v := range pod.Labels {
		labels = append(labels, fmt.Sprintf("%s=%s", k, v))
	}
	sort.Strings(labels)
	return labels
}

func HumanReadableDuration(duration time.Duration) string {
	if duration.Seconds() < 60 {
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	} else if duration.Minutes() < 60 {
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	} else if duration.Hours() < 24 {
		return fmt.Sprintf("%dh", int(duration.Hours()))
	}
	return fmt.Sprintf("%dd", int(duration.Hours()/24))
}

func CalculateReadiness(pod *corev1.Pod) (int, int) {
	var ready, total int
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.Ready {
			ready++
		}
		total++
	}
	return ready, total
}
