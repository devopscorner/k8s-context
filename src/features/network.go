package features

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ShowServiceByFilter(services *corev1.ServiceList) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{
		"NAME",
		"TYPE",
		"CLUSTER-IP",
		"EXTERNAL-IP(S)",
		"PORT(S)",
		"AGE",
	})

	table.SetAutoFormatHeaders(false)
	table.SetAutoWrapText(false)

	for _, service := range services.Items {
		var externalIPs string
		if service.Spec.Type == corev1.ServiceTypeLoadBalancer && len(service.Status.LoadBalancer.Ingress) > 0 {
			if service.Status.LoadBalancer.Ingress[0].IP != "" {
				externalIPs = service.Status.LoadBalancer.Ingress[0].IP
			} else if service.Status.LoadBalancer.Ingress[0].Hostname != "" {
				externalIPs = service.Status.LoadBalancer.Ingress[0].Hostname
			} else {
				externalIPs = "<pending>"
			}
		} else if len(service.Spec.ExternalIPs) > 0 {
			externalIPs = strings.Join(service.Spec.ExternalIPs, ", ")
		} else {
			externalIPs = "<none>"
		}
		age := HumanReadableDuration(time.Since(service.ObjectMeta.CreationTimestamp.Time))
		ports := make([]string, len(service.Spec.Ports))
		for i, port := range service.Spec.Ports {
			protocolName := string(port.Protocol)
			if port.Port != port.TargetPort.IntVal {
				// The port is not named, so try to find the corresponding named port
				for _, namedPort := range service.Spec.Ports {
					if namedPort.Name == port.Name {
						protocolName = string(namedPort.Protocol)
						break
					}
				}
			}
			if port.Port == 0 {
				ports[i] = fmt.Sprintf("%s", protocolName)
			} else {
				ports[i] = fmt.Sprintf("%d", port.Port)
			}
			if port.TargetPort.String() != "0" {
				ports[i] += ":" + port.TargetPort.String()
			}
			ports[i] += "/" + protocolName
		}

		table.Append([]string{
			service.Name,
			string(service.Spec.Type),
			service.Spec.ClusterIP,
			externalIPs,
			strings.Join(ports, ", "),
			age,
		})
	}
	table.Render()
}

func ShowEndpointByFilter(endpoints *corev1.EndpointsList) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{
		"NAME",
		"ENDPOINTS TARGET",
		"ENDPOINTS PORT(S)",
		// "ENDPOINTS NAME",
		"AGE",
	})

	table.SetAutoFormatHeaders(false)
	table.SetAutoWrapText(false)

	for _, ep := range endpoints.Items {
		serviceName := ep.ObjectMeta.Name
		age := HumanReadableDuration(time.Since(ep.ObjectMeta.CreationTimestamp.Time))

		for _, subset := range ep.Subsets {
			addresses := make([]string, len(subset.Addresses))
			ports := make([]string, len(subset.Ports))
			for i, addr := range subset.Addresses {
				target := addr.TargetRef.Name
				if addr.TargetRef.Kind == "Pod" {
					pod, err := GetPod(addr.TargetRef.Namespace, target)
					if err == nil {
						target = fmt.Sprintf("%s (%s)", target, pod.Status.PodIP)
					}
				}
				addresses[i] = target
			}
			for i, port := range subset.Ports {
				portNumber := strconv.Itoa(int(port.Port))
				if int(port.Port) == 0 {
					portNumber = port.Name
				}
				ports[i] = portNumber
			}
			table.Append([]string{
				serviceName,
				strings.Join(addresses, ", "),
				strings.Join(ports, ", "),
				age,
			})
		}
	}

	table.Render()
}

func GetPod(namespace string, name string) (*corev1.Pod, error) {
	clientset, err := GetClientSet(kubeconfig)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return pod, nil
}
