package features

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
)

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
