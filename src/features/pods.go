package features

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	yaml "gopkg.in/yaml.v2"
	v1 "k8s.io/api/apps/v1"
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

func ShowPodsByFilter(pods *corev1.PodList) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{
		"POD NAME",
		"READY",
		"STATUS",
		"RESTARTS",
		"AGE",
		"IMAGE",
		"NODE",
	})

	table.SetAutoFormatHeaders(false)
	table.SetAutoWrapText(false)

	for _, pod := range pods.Items {
		var containerStatuses []string
		for _, cs := range pod.Status.ContainerStatuses {
			containerStatuses = append(containerStatuses, fmt.Sprintf("%s:%s", cs.Name, strconv.FormatBool(cs.Ready)))
		}
		ready, total := CalculateReadiness(&pod)
		age := HumanReadableDuration(time.Since(pod.ObjectMeta.CreationTimestamp.Time))
		image := strings.Join(GetContainerImages(&pod), ", ")
		node := pod.Spec.NodeName

		table.Append([]string{
			pod.Name,
			fmt.Sprintf("%d/%d", ready, total),
			string(pod.Status.Phase),
			strconv.Itoa(int(pod.Status.ContainerStatuses[0].RestartCount)),
			age,
			image,
			node,
		})
	}
	table.Render()
}

func ShowNamespaceByFilter(namespaces *corev1.NamespaceList) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{
		"NAME",
		"STATUS",
		"AGE",
	})
	table.SetAutoFormatHeaders(false)
	table.SetAutoWrapText(false)

	for _, ns := range namespaces.Items {
		name := ns.ObjectMeta.Name
		status := ns.Status.Phase
		age := HumanReadableDuration(time.Since(ns.ObjectMeta.CreationTimestamp.Time))

		table.Append([]string{
			name,
			string(status),
			age,
		})
	}
	table.Render()
}

func ShowServiceByFilter(services *corev1.ServiceList) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{
		"NAME",
		"TYPE",
		"CLUSTER-IP",
		"EXTERNAL-IP",
		"PORT(S)",
		"AGE",
	})

	table.SetAutoFormatHeaders(false)
	table.SetAutoWrapText(false)

	for _, service := range services.Items {
		var externalIPs string
		if len(service.Spec.ExternalIPs) > 0 {
			externalIPs = strings.Join(service.Spec.ExternalIPs, ", ")
		} else {
			externalIPs = "<none>"
		}
		age := HumanReadableDuration(time.Since(service.ObjectMeta.CreationTimestamp.Time))
		ports := make([]string, len(service.Spec.Ports))
		for i, port := range service.Spec.Ports {
			ports[i] = fmt.Sprintf("%d/%s", port.Port, string(port.Protocol))
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

func ShowDeploymentByFilter(deployments *v1.DeploymentList) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{
		"NAME",
		"READY",
		"UP-TO-DATE",
		"AVAILABLE",
		"AGE",
	})

	table.SetAutoFormatHeaders(false)
	table.SetAutoWrapText(false)

	for _, deploy := range deployments.Items {
		name := deploy.Name
		age := HumanReadableDuration(time.Since(deploy.ObjectMeta.CreationTimestamp.Time))

		table.Append([]string{
			name,
			fmt.Sprintf("%d/%d", deploy.Status.ReadyReplicas, deploy.Status.Replicas),
			fmt.Sprintf("%d", deploy.Status.UpdatedReplicas),
			fmt.Sprintf("%d", deploy.Status.AvailableReplicas),
			age,
		})
	}
	table.Render()
}

func DescribePods(pod *corev1.Pod) {
	// Print detailed information about the pod
	fmt.Printf("Name: \t\t%s\n", pod.ObjectMeta.Name)
	fmt.Printf("Namespace: \t%s\n", pod.ObjectMeta.Namespace)
	labelsJSON, err := json.MarshalIndent(pod.ObjectMeta.Labels, "", "\t")
	if err != nil {
		fmt.Println("Error marshaling labels to JSON:", err)
	} else {
		fmt.Println("Labels:\n", string(labelsJSON))
	}
	fmt.Printf("Status: \t%s\n", pod.Status.Phase)
	fmt.Printf("IP Address: \t%s\n", pod.Status.PodIP)
	fmt.Printf("Node Name: \t%s\n", pod.Spec.NodeName)
}

func DescribePodsDetail(pod *corev1.Pod) {
	var state string

	// Print detailed information about the pod
	fmt.Printf("Name:      \t%s\n", pod.ObjectMeta.Name)
	fmt.Printf("Namespace: \t%s\n", pod.ObjectMeta.Namespace)
	fmt.Printf("Priority:  \t%d\n", pod.Spec.Priority)
	fmt.Printf("Node:      \t%s\n", pod.Spec.NodeName)
	fmt.Printf("Start Time:\t%s\n", pod.Status.StartTime.Time)

	// Convert labels to YAML
	labelsYAML, err := yaml.Marshal(pod.ObjectMeta.Labels)
	if err != nil {
		fmt.Println("Error marshaling labels to YAML:", err)
	} else {
		fmt.Printf("Labels:")
		yamlLines := strings.Split(string(labelsYAML), "\n")
		for _, line := range yamlLines {
			fmt.Printf("\t\t%s\n", line)
		}
	}
	fmt.Printf("Annotations:")
	for key, value := range pod.ObjectMeta.Annotations {
		fmt.Printf("\t%s: %s\n", key, value)
	}
	fmt.Printf("Status:      \t%s\n", pod.Status.Phase)
	fmt.Printf("IP:          \t%s\n", pod.Status.PodIP)

	fmt.Printf("IPs:\n")
	for _, podIP := range pod.Status.PodIPs {
		fmt.Printf("  IP: \t\t%s\n", podIP.IP)
	}
	fmt.Printf("Controlled By: \t%s/%s\n", pod.ObjectMeta.OwnerReferences[0].Kind, pod.ObjectMeta.OwnerReferences[0].Name)
	fmt.Println("---------------------------------------------------------------------------")
	fmt.Println("Containers:")
	for _, container := range pod.Spec.Containers {

		containerStatus := GetContainerStatus(pod, container.Name)
		if containerStatus != nil {
			fmt.Printf("  %s:\n", container.Name)
			fmt.Printf("    Container ID: \t%s\n", containerStatus.ContainerID)
			fmt.Printf("    Image:        \t%s\n", container.Image)
			fmt.Printf("    Image ID:     \t%s\n", containerStatus.ImageID)

			if len(pod.Spec.Containers[0].Ports) > 0 {
				ports := ""
				for _, p := range pod.Spec.Containers[0].Ports {
					ports += fmt.Sprintf("%d/%s, ", p.ContainerPort, p.Protocol)
				}
				fmt.Printf("    Port(s):\t\t%s\n", ports[:len(ports)-2])
			} else {
				fmt.Printf("    Port(s):\t\t<none>\n")
			}

			if len(pod.Spec.Containers[0].Ports) > 0 {
				if pod.Spec.Containers[0].Ports[0].HostPort != 0 {
					fmt.Printf("    Host Port: \t\t%d\n", pod.Spec.Containers[0].Ports[0].HostPort)
				} else {
					fmt.Printf("    Host Port: \t\t<none>\n")
				}
			} else {
				fmt.Printf("    Host Port: \t\t<none>\n")
			}

			if pod.Status.ContainerStatuses[0].State.Running != nil {
				state = "Running"
			} else if pod.Status.ContainerStatuses[0].State.Terminated != nil {
				state = "Terminated"
			} else {
				state = "Waiting"
			}
			fmt.Printf("    State: \t\t%s\n", state)
			if pod.Status.ContainerStatuses[0].State.Running != nil {
				fmt.Printf("      Started: \t\t%s\n", pod.Status.ContainerStatuses[0].State.Running.StartedAt.Time)
			}

			if containerStatus.LastTerminationState.Terminated != nil {
				fmt.Printf("    Last State:\n")
				fmt.Printf("      Reason:     \t%s\n", containerStatus.LastTerminationState.Terminated.Reason)
				fmt.Printf("      Exit Code:  \t%d\n", containerStatus.LastTerminationState.Terminated.ExitCode)
				fmt.Printf("      Started:    \t%s\n", containerStatus.LastTerminationState.Terminated.StartedAt.Time)
				fmt.Printf("      Finished:   \t%s\n", containerStatus.LastTerminationState.Terminated.FinishedAt.Time)
			}

			fmt.Printf("    Ready:        \t%t\n", containerStatus.Ready)
			fmt.Printf("    Restart Count: \t%d\n", containerStatus.RestartCount)
			fmt.Printf("    Limits:\n")
			fmt.Printf("      cpu:        %s\n", container.Resources.Limits.Cpu().String())
			fmt.Printf("      memory:     %s\n", container.Resources.Limits.Memory().String())
			fmt.Printf("    Requests:\n")
			fmt.Printf("      cpu:        %s\n", container.Resources.Requests.Cpu().String())
			fmt.Printf("      memory:     %s\n", container.Resources.Requests.Memory().String())

			labelsYAML, err := yaml.Marshal(container.Env)
			if err != nil {
				fmt.Println("Error marshaling environment variables to YAML:", err)
			} else {
				fmt.Printf("    Environment:\n")
				env := make([]map[string]interface{}, 0)
				if err := yaml.Unmarshal(labelsYAML, &env); err != nil {
					fmt.Println("Error unmarshaling environment variables from YAML:", err)
				} else {
					for _, v := range env {
						name := v["name"].(string)
						value := v["value"].(string)
						fmt.Printf("      %s: %s\n", name, value)
					}
				}
			}

			fmt.Printf("    Mounts:\n")
			for _, mount := range container.VolumeMounts {
				fmt.Printf("      %s from %s (ro:%t)\n", mount.MountPath, mount.Name, mount.ReadOnly)
			}

			fmt.Println("Conditions:")
			fmt.Printf("  Type: \t\tStatus\n")
			for _, cond := range pod.Status.Conditions {
				if cond.Type == "Initialized" {
					fmt.Printf("  %s\t\t%s\n", cond.Type, cond.Status)
				}
				if cond.Type == "Ready" {
					fmt.Printf("  %s\t\t\t%s\n", cond.Type, cond.Status)
				}
				if cond.Type == "ContainersReady" {
					fmt.Printf("  %s\t%s\n", cond.Type, cond.Status)
				}
				if cond.Type == "PodScheduled" {
					fmt.Printf("  %s\t\t%s\n", cond.Type, cond.Status)
				}
			}

			fmt.Println("Volumes:")
			for _, volume := range pod.Spec.Volumes {
				switch {
				case volume.ConfigMap != nil:
					fmt.Printf("  %s:\n", volume.Name)
					fmt.Printf("    Type: ConfigMap\n")
					fmt.Printf("    Name: %s\n", volume.ConfigMap.Name)
					if volume.ConfigMap.Optional != nil {
						fmt.Printf("    Optional: %t\n", *volume.ConfigMap.Optional)
					} else {
						fmt.Printf("    Optional: false\n")
					}
				case volume.Secret != nil:
					fmt.Printf("  %s:\n", volume.Name)
					fmt.Printf("    Type: Secret\n")
					fmt.Printf("    Name: %s\n", volume.Secret.SecretName)
					if volume.Secret.Optional != nil {
						fmt.Printf("    Optional: %t\n", *volume.Secret.Optional)
					} else {
						fmt.Printf("    Optional: false\n")
					}
				default:
					fmt.Printf("  %s: Unknown volume type\n", volume.Name)
				}
			}

			// QoS Class
			fmt.Printf("QoS Class: \t\t%s\n", pod.Status.QOSClass)

			// Node Selectors
			nodeSelectors := "<none>"
			if len(pod.Spec.NodeSelector) > 0 {
				nodeSelectors = fmt.Sprintf("%v", pod.Spec.NodeSelector)
			}
			fmt.Printf("Node-Selectors: \t%s\n", nodeSelectors)
			fmt.Printf("\n")

			// Tolerations
			tolerations := pod.Spec.Tolerations
			tolerationStrings := make([]string, 0)

			for _, toleration := range tolerations {
				tolerationStrings = append(tolerationStrings, fmt.Sprintf("%s:%s op=%s for %ds", toleration.Key, toleration.Operator, toleration.Effect, toleration.TolerationSeconds))
			}
			tolerationsString := strings.Join(tolerationStrings, "\n\t\t")
			fmt.Printf("Tolerations:\t%s\n", strings.ReplaceAll(fmt.Sprintf("%v", tolerationsString), " ", "\t"))

			// Events
			events := "<none>"
			fmt.Println("Events:")
			if len(pod.Status.Conditions) > 0 {
				events = ""
				for _, condition := range pod.Status.Conditions {
					fmt.Printf("  %-16s %v\n", condition.Type, condition.Status)
					fmt.Printf("  Last Timestamp:  %v\n", condition.LastTransitionTime)
				}
				events = strings.TrimSuffix(events, ", ")
			}
			fmt.Printf("\t%s\n", events)
		}
	}
}

func GetContainerStatus(pod *corev1.Pod, containerName string) *corev1.ContainerStatus {
	for _, status := range pod.Status.ContainerStatuses {
		if status.Name == containerName {
			return &status
		}
	}
	return nil
}

func GetFreePort() (int, error) {
	// listen on a random port
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()

	// extract the port number
	addr := l.Addr().(*net.TCPAddr)
	port := addr.Port

	return port, nil
}

func BoolToString(value corev1.ConditionStatus) string {
	if value == corev1.ConditionTrue {
		return "True"
	}
	return "False"
}

func ByteCountSI(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "kMGTPE"[exp])
}

func DescribeNode(node *corev1.Node) {
	// Print detailed information about the node
	fmt.Println("Name: \t", node.ObjectMeta.Name)

	labelsJSON, err := json.MarshalIndent(node.ObjectMeta.Labels, "", "\t")
	if err != nil {
		fmt.Println("Error marshaling labels to JSON:", err)
	} else {
		fmt.Println("Labels:\n", string(labelsJSON))
	}

	addrs := node.Status.Addresses
	fmt.Println("Addresses:")
	for _, addr := range addrs {
		fmt.Printf("  - %s: %s\n", addr.Type, addr.Address)
	}

	fmt.Println("Allocatable Resources:")
	for resourceName, quantity := range node.Status.Allocatable {
		fmt.Printf("  - %s: %s\n", resourceName, quantity.String())
	}

	fmt.Println("Capacity:")
	for resourceName, quantity := range node.Status.Capacity {
		fmt.Printf("  - %s: %s\n", resourceName, quantity.String())
	}

	fmt.Println("Conditions:")
	for _, condition := range node.Status.Conditions {
		fmt.Printf("  - %s: %s\n", condition.Type, BoolToString(condition.Status))
	}

	fmt.Println("Daemon Endpoint:")
	// endpoint := n.Status.DaemonEndpoints.KubeletEndpoint
	// fmt.Printf("  - Kubelet Endpoint: %s\n", endpoint.String())
	fmt.Printf("  - Kubelet Endpoint Port: %d\n", node.Status.DaemonEndpoints.KubeletEndpoint.Port)

	fmt.Println("Images:")
	for _, image := range node.Status.Images {
		fmt.Printf("  - %s: %d\n", image.Names[0], image.SizeBytes)
	}

	fmt.Println("Node Info:")
	fmt.Printf("  - Machine ID: \t\t%s\n", node.Status.NodeInfo.MachineID)
	fmt.Printf("  - System UUID: \t\t%s\n", node.Status.NodeInfo.SystemUUID)
	fmt.Printf("  - Boot ID: \t\t\t%s\n", node.Status.NodeInfo.BootID)
	fmt.Printf("  - OS Image: \t\t\t%s\n", node.Status.NodeInfo.OSImage)
	fmt.Printf("  - Kernel Version: \t\t%s\n", node.Status.NodeInfo.KernelVersion)
	fmt.Printf("  - Container Runtime Version: \t%s\n", node.Status.NodeInfo.ContainerRuntimeVersion)
	fmt.Printf("  - Kubelet Version: \t\t%s\n", node.Status.NodeInfo.KubeletVersion)
	fmt.Printf("  - Kube-Proxy Version: \t%s\n", node.Status.NodeInfo.KubeProxyVersion)
	fmt.Printf("  - Operating System: \t\t%s\n", node.Status.NodeInfo.OperatingSystem)
	fmt.Printf("  - Architecture: \t\t%s\n", node.Status.NodeInfo.Architecture)
}

func DescribeNodeTable(node *corev1.Node) {
	fmt.Printf("Name:\t%s\n", node.Name)

	labels, _ := json.MarshalIndent(node.Labels, "", "\t")
	fmt.Printf("Labels:\n%s\n", string(labels))

	addresses := node.Status.Addresses
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Type", "Address"})
	for _, addr := range addresses {
		table.Append([]string{string(addr.Type), addr.Address})
	}
	table.Render()

	table = tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Resource", "Allocatable", "Capacity"})
	allocatable := node.Status.Allocatable
	capacity := node.Status.Capacity
	for resourceName, quantity := range allocatable {
		capacityQuantity := capacity[resourceName]
		table.Append([]string{string(resourceName), quantity.String(), capacityQuantity.String()})
	}
	table.Render()

	table = tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Condition", "Status"})
	for _, condition := range node.Status.Conditions {
		table.Append([]string{string(condition.Type), BoolToString(condition.Status)})
	}
	table.Render()

	fmt.Printf("Daemon Endpoint Port:\t%d\n", node.Status.DaemonEndpoints.KubeletEndpoint.Port)

	table = tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Size"})
	for _, image := range node.Status.Images {
		table.Append([]string{image.Names[0], ByteCountSI(image.SizeBytes)})
	}
	table.Render()

	fmt.Printf("Machine ID:\t\t%s\n", node.Status.NodeInfo.MachineID)
	fmt.Printf("System UUID:\t\t%s\n", node.Status.NodeInfo.SystemUUID)
	fmt.Printf("Boot ID:\t\t%s\n", node.Status.NodeInfo.BootID)
	fmt.Printf("OS Image:\t\t%s\n", node.Status.NodeInfo.OSImage)
	fmt.Printf("Kernel Version:\t\t%s\n", node.Status.NodeInfo.KernelVersion)
	fmt.Printf("Container Runtime Version:\t%s\n", node.Status.NodeInfo.ContainerRuntimeVersion)
	fmt.Printf("Kubelet Version:\t%s\n", node.Status.NodeInfo.KubeletVersion)
	fmt.Printf("Kube-Proxy Version:\t%s\n", node.Status.NodeInfo.KubeProxyVersion)
	fmt.Printf("Operating System:\t%s\n", node.Status.NodeInfo.OperatingSystem)
	fmt.Printf("Architecture:\t\t%s\n", node.Status.NodeInfo.Architecture)
}
