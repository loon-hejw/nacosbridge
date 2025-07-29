package service

import (
	"encoding/json"
	"strings"
	"sync"

	corev1 "k8s.io/api/core/v1"
)

type Config struct {
	WatchNamespace map[string]string `json:"watch_namespace"`
	ServiceConfig  map[string]string `json:"service_config"`
	initialize     sync.Once
}

func (c *Config) init() {
	c.WatchNamespace = make(map[string]string)
	c.ServiceConfig = make(map[string]string)
}

func (c *Config) Load(content string) error {
	c.initialize.Do(c.init)
	return json.Unmarshal([]byte(content), &c)
}

type Service struct {
	Name    string
	Port    int32
	IP      []string
	NacosNs string
}

func GenerateServiceInfos(svc *corev1.Service, nodeIps []string, selectNamespace map[string]bool) []Service {
	serviceInfos := make([]Service, 0)

	// 检查服务是否应该被处理
	if svc.Labels == nil {
		return serviceInfos
	}
	if _, ok := svc.Labels["nacosbridge.io/service"]; !ok {
		return serviceInfos
	}
	if len(selectNamespace) > 0 && !selectNamespace[svc.Namespace] {
		return serviceInfos
	}

	// 处理ExternalIPs
	if len(svc.Spec.ExternalIPs) > 0 {
		serviceInfos = append(serviceInfos, generateServiceInfosForPorts(svc, svc.Spec.ExternalIPs, nil)...)
	}
	// 根据服务类型处理特殊逻辑
	switch svc.Spec.Type {
	case corev1.ServiceTypeLoadBalancer:
		if svc.Spec.LoadBalancerIP != "" {
			serviceInfos = append(serviceInfos, generateServiceInfosForPorts(svc, []string{svc.Spec.LoadBalancerIP}, nil)...)
		}
	case corev1.ServiceTypeExternalName:
		if svc.Spec.ExternalName != "" {
			serviceInfos = append(serviceInfos, generateServiceInfosForPorts(svc, []string{svc.Spec.ExternalName}, nil)...)
		}
	case corev1.ServiceTypeNodePort:
		if len(nodeIps) > 0 {
			serviceInfos = append(serviceInfos, generateServiceInfosForPorts(svc, nodeIps, func(port corev1.ServicePort) int32 {
				return port.NodePort
			})...)
		}
	}

	return serviceInfos
}

// generateServiceInfosForPorts 为指定的IP和端口生成ServiceInfo
func generateServiceInfosForPorts(svc *corev1.Service, ips []string, portMapper func(corev1.ServicePort) int32) []Service {
	serviceInfos := make([]Service, 0)

	// 获取基础服务名
	baseServiceName := svc.Name
	if customName, ok := svc.Labels["nacosbridge.io/service"]; ok {
		baseServiceName = customName
	}

	namespace := ""
	if customNamespace := svc.Labels["nacosbridge.io/namespace"]; customNamespace != "" {
		namespace = customNamespace
	}

	// 获取开放的端口名称
	openPortName := make(map[string]bool)
	if ns, ok := svc.Labels["nacosbridge.io/openport"]; ok {
		for _, name := range strings.Split(ns, ",") {
			openPortName[name] = true
		}
	}

	for _, port := range svc.Spec.Ports {

		if !openPortName[port.Name] {
			continue
		}
		serviceName := baseServiceName
		if svc.Labels != nil {
			if customPortName := svc.Labels["nacosbridge.io/portname-"+port.Name]; customPortName != "" {
				serviceName = customPortName
			}
		}
		// 确定端口号
		var portNumber int32
		if portMapper != nil {
			portNumber = portMapper(port)
		} else {
			portNumber = port.Port
		}

		serviceInfos = append(serviceInfos, Service{
			Name:    serviceName,
			NacosNs: namespace,
			IP:      ips,
			Port:    portNumber,
		})
	}

	return serviceInfos
}

func GenerateServiceConfig(svc string, config map[string]string) map[string]string {

	newConfig := make(map[string]string)
	prefix := svc + "."
	for k, v := range config {
		if strings.HasPrefix(k, prefix) {
			newKey := strings.TrimPrefix(k, prefix)
			newConfig[newKey] = v
		}
	}
	return newConfig
}
