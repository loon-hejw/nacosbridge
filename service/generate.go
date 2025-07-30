package service

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"

	corev1 "k8s.io/api/core/v1"
)

const (
	// registry config
	REGISTRY_CONFIG = "nacosbridge.io/config"

	// registry service name
	REGISTRY_SERVICE_NAME = "nacosbridge.io/service"

	// registry service namespace
	REGISTRY_SERVICE_NAMESPACE = "nacosbridge.io/namespace"

	// registry service type
	REGISTRY_SERVICE_TYPE = "nacosbridge.io/service-type"

	// registry service cluster to external
	REGISTRY_SERVICE_EXTERNAL = "nacosbridge.io/external"

	// registry service openport
	REGISTRY_OPENPORT = "nacosbridge.io/openport"

	// registry service portname
	REGISTRY_PORTNAME = "nacosbridge.io/portname-"

	// registry service domin
	REGISTRY_SERVICE_DOMIN = "nacosbridge.io/service-domin"

	// registry service domin port
	REGISTRY_SERVICE_DOMIN_PORT = "nacosbridge.io/service-domain-"

	// registry service matedata
	SERVICE_MATEDATA = "nacosbridge.io/matedata"
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
	Name     string
	Port     int32
	IP       []string
	NacosNs  string
	Metadata map[string]string
}

func GenerateServiceInfos(svc *corev1.Service, nodeIps []string, selectNamespace map[string]bool) []Service {
	serviceInfos := make([]Service, 0)

	// 检查服务是否应该被处理
	if svc.Labels == nil {
		return serviceInfos
	}
	if _, ok := svc.Labels[REGISTRY_SERVICE_NAME]; !ok {
		return serviceInfos
	}
	if len(selectNamespace) > 0 && !selectNamespace[svc.Namespace] {
		return serviceInfos
	}

	switch svc.Labels[REGISTRY_SERVICE_TYPE] {
	case "cluster":
		serviceInfos = append(serviceInfos, generateServiceInfosForCluster(svc)...)
	case "external":
		switch svc.Spec.Type {
		case corev1.ServiceTypeLoadBalancer:
			if svc.Spec.LoadBalancerIP != "" {
				serviceInfos = append(serviceInfos, generateServiceInfosForExternal(svc, []string{svc.Spec.LoadBalancerIP}, nil)...)
			}
		case corev1.ServiceTypeExternalName:
			if svc.Spec.ExternalName != "" {
				serviceInfos = append(serviceInfos, generateServiceInfosForExternal(svc, []string{svc.Spec.ExternalName}, nil)...)
			}
		case corev1.ServiceTypeNodePort:
			if len(nodeIps) > 0 {
				serviceInfos = append(serviceInfos, generateServiceInfosForExternal(svc, nodeIps, func(port corev1.ServicePort) int32 {
					return port.NodePort
				})...)
			}
		}
	case "gateway":
		serviceInfos = append(serviceInfos, generateServiceInfosForGateway(svc)...)
	}
	return serviceInfos
}

func generateServiceInfosForCluster(svc *corev1.Service) []Service {
	serviceInfos := make([]Service, 0)

	// 获取基础服务名
	baseServiceName := svc.Name
	if customName, ok := svc.Labels[REGISTRY_SERVICE_NAME]; ok {
		baseServiceName = customName
	}

	namespace := ""
	if customNamespace := svc.Labels[REGISTRY_SERVICE_NAMESPACE]; customNamespace != "" {
		namespace = customNamespace
	}

	// 获取开放的端口名称
	openPortName := make(map[string]bool)
	if ns, ok := svc.Labels[REGISTRY_OPENPORT]; ok {
		for _, name := range strings.Split(ns, ",") {
			openPortName[name] = true
		}
	}
	metadata := GeneratePrefixConfig(SERVICE_MATEDATA, svc.Annotations)

	for _, port := range svc.Spec.Ports {

		if !openPortName[port.Name] {
			continue
		}
		serviceName := baseServiceName
		if svc.Labels != nil {
			if customPortName := svc.Labels[REGISTRY_PORTNAME+port.Name]; customPortName != "" {
				serviceName = customPortName
			}
		}
		domin := fmt.Sprintf("%s.%s.svc.cluster.local", svc.Name, svc.Namespace)
		portNumber := port.Port
		serviceInfos = append(serviceInfos, Service{
			Name:     serviceName,
			NacosNs:  namespace,
			IP:       []string{domin},
			Port:     portNumber,
			Metadata: metadata,
		})
	}
	return serviceInfos
}

// generateServiceInfosForPorts 为指定的IP和端口生成ServiceInfo
func generateServiceInfosForExternal(svc *corev1.Service, ips []string, portMapper func(corev1.ServicePort) int32) []Service {
	serviceInfos := make([]Service, 0)

	// 获取基础服务名
	baseServiceName := svc.Name
	if customName, ok := svc.Labels[REGISTRY_SERVICE_NAME]; ok {
		baseServiceName = customName
	}

	namespace := ""
	if customNamespace := svc.Labels[REGISTRY_SERVICE_NAMESPACE]; customNamespace != "" {
		namespace = customNamespace
	}

	// 获取开放的端口名称
	openPortName := make(map[string]bool)
	if ns, ok := svc.Labels[REGISTRY_OPENPORT]; ok {
		for _, name := range strings.Split(ns, ",") {
			openPortName[name] = true
		}
	}

	metadata := GeneratePrefixConfig(SERVICE_MATEDATA, svc.Annotations)

	for _, port := range svc.Spec.Ports {

		if !openPortName[port.Name] {
			continue
		}
		serviceName := baseServiceName
		if svc.Labels != nil {
			if customPortName := svc.Labels[REGISTRY_PORTNAME+port.Name]; customPortName != "" {
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
			Name:     serviceName,
			NacosNs:  namespace,
			IP:       ips,
			Port:     portNumber,
			Metadata: metadata,
		})
	}

	return serviceInfos
}

func generateServiceInfosForGateway(svc *corev1.Service) []Service {
	serviceInfos := make([]Service, 0)

	// 获取基础服务名
	baseServiceName := svc.Name
	if customName, ok := svc.Labels[REGISTRY_SERVICE_NAME]; ok {
		baseServiceName = customName
	}

	namespace := ""
	if customNamespace := svc.Labels[REGISTRY_SERVICE_NAMESPACE]; customNamespace != "" {
		namespace = customNamespace
	}

	domin, ok := svc.Labels[REGISTRY_SERVICE_DOMIN]
	if !ok {
		return serviceInfos
	}
	metadata := GeneratePrefixConfig(SERVICE_MATEDATA, svc.Annotations)

	for k, v := range svc.Labels {
		if strings.HasPrefix(k, REGISTRY_SERVICE_DOMIN_PORT) {
			port, err := strconv.Atoi(v)
			if err != nil {
				continue
			}
			serviceInfos = append(serviceInfos, Service{
				Name:     baseServiceName,
				NacosNs:  namespace,
				IP:       []string{domin},
				Port:     int32(port),
				Metadata: metadata,
			})
		}
	}
	return serviceInfos
}

func GeneratePrefixConfig(prefix string, config map[string]string) map[string]string {

	if prefix == "" || config == nil {
		return make(map[string]string)
	}
	newConfig := make(map[string]string)
	hasPrefix := fmt.Sprintf("%s.", prefix)
	for k, v := range config {
		if strings.HasPrefix(k, hasPrefix) {
			newKey := strings.TrimPrefix(k, prefix)
			newConfig[newKey] = v
		}
	}
	return newConfig
}
