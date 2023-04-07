package features

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	corev1 "k8s.io/api/core/v1"
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
