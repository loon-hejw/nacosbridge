package service

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/go-logr/logr"
	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Nacos struct {
	only sync.Once

	address  string
	port     int
	username string
	password string

	namespaceClients map[string]naming_client.INamingClient

	newService map[string]Service
	oldService map[string]Service

	log logr.Logger
}

func (n *Nacos) init() {
	n.only.Do(func() {
		n.newService = make(map[string]Service)
		n.oldService = make(map[string]Service)
		n.namespaceClients = make(map[string]naming_client.INamingClient)
		n.log = log.Log.WithName("nacos")
	})
}

func (n *Nacos) Name() string {
	return "nacos"
}

func (n *Nacos) Config(config map[string]string) error {

	n.init()
	// 解析配置参数
	if address, ok := config["address"]; ok {
		n.address = address
	} else {
		return fmt.Errorf("nacos address is required")
	}

	if portStr, ok := config["port"]; ok {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return fmt.Errorf("invalid nacos port: %v", err)
		}
		n.port = port
	} else {
		n.port = 8848 // 默认端口
	}

	if username, ok := config["username"]; ok {
		n.username = username
	}

	if password, ok := config["password"]; ok {
		n.password = password
	}
	return nil
}

func (n *Nacos) Build(services []Service) error {

	for _, svc := range services {
		for _, ip := range svc.IP {
			svcName := fmt.Sprintf("%s.%s.%s", svc.NacosNs, svc.Name, ip)
			if _, ok := n.newService[svcName]; !ok {
				n.newService[svcName] = svc
			}
		}
	}

	addSvc := make([]Service, 0)
	delSvc := make([]Service, 0)
	for k, svc := range n.newService {
		if _, ok := n.oldService[k]; !ok {
			addSvc = append(addSvc, svc)
		}
	}

	for k, svc := range n.oldService {
		if _, ok := n.newService[k]; !ok {
			delSvc = append(delSvc, svc)
		}
	}

	for _, svc := range addSvc {
		if err := n.registerService(svc); err != nil {
			return fmt.Errorf("failed to register service %s: %v", svc.Name, err)
		}
	}

	for _, svc := range delSvc {
		if err := n.deregisterService(svc); err != nil {
			return fmt.Errorf("failed to deregister service %s: %v", svc.Name, err)
		}
	}

	n.oldService = n.newService
	n.newService = make(map[string]Service)
	return nil
}

// registerService 注册单个服务到nacos
func (n *Nacos) registerService(service Service) error {

	namespace := service.NacosNs
	if namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	nacosClient, err := n.generateNamespaceClient(namespace)
	if err != nil {
		return fmt.Errorf("failed to generate namespace client for %s: %v", namespace, err)
	}

	// 添加调试日志
	n.log.Info("DEBUG: Attempting to register service", "service", service.Name, "namespace", namespace, "ips", service.IP, "port", service.Port)

	for _, ip := range service.IP {
		param := vo.RegisterInstanceParam{
			Ip:          ip,
			Port:        uint64(service.Port),
			ServiceName: service.Name,
			Weight:      10, // 默认权重
			Enable:      true,
			Healthy:     true,
			Ephemeral:   false,
			Metadata: map[string]string{
				"created_by": "nacosbridge.io",
				"cluster":    "k8s.service",
			},
		}

		// 添加参数调试日志
		n.log.Info("DEBUG: Registering instance with params", "ip", param.Ip, "port", param.Port, "serviceName", param.ServiceName, "ephemeral", param.Ephemeral)

		success, err := nacosClient.RegisterInstance(param)
		if err != nil {
			n.log.Error(err, "ERROR: Register instance failed", "ip", ip, "port", service.Port, "serviceName", service.Name)
			return fmt.Errorf("failed to register service %s: %v", service.Name, err)
		}
		if !success {
			n.log.Error(fmt.Errorf("register instance returned false"), "WARNING: Register instance returned false", "ip", ip, "port", service.Port, "serviceName", service.Name)
			return fmt.Errorf("failed to register service %s: service not registered", service.Name)
		}
		n.log.Info("SUCCESS: Registered instance", "ip", ip, "port", service.Port, "serviceName", service.Name)
	}
	return nil
}

func (n *Nacos) deregisterService(service Service) error {

	namespace := service.NacosNs
	if namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	nacosClient, err := n.generateNamespaceClient(namespace)
	if err != nil {
		return fmt.Errorf("failed to generate namespace client for %s: %v", namespace, err)
	}

	// 添加调试日志
	n.log.Info("DEBUG: Attempting to deregister service", "service", service.Name, "namespace", namespace, "ips", service.IP, "port", service.Port)

	for _, ip := range service.IP {
		param := vo.DeregisterInstanceParam{
			Ip:          ip,
			Port:        uint64(service.Port),
			ServiceName: service.Name,
			Ephemeral:   false,
		}

		// 添加参数调试日志
		n.log.Info("DEBUG: Deregistering instance with params", "ip", param.Ip, "port", param.Port, "serviceName", param.ServiceName, "ephemeral", param.Ephemeral)

		success, err := nacosClient.DeregisterInstance(param)
		if err != nil {
			n.log.Error(err, "ERROR: Deregister instance failed", "ip", ip, "port", service.Port, "serviceName", service.Name)
			return fmt.Errorf("deregister instance failed: %v", err)
		}
		if !success {
			n.log.Error(fmt.Errorf("deregister instance returned false"), "WARNING: Deregister instance returned false", "ip", ip, "port", service.Port, "serviceName", service.Name)
			return fmt.Errorf("deregister instance failed: service not deregistered")
		}
		n.log.Info("SUCCESS: Deregistered instance", "ip", ip, "port", service.Port, "serviceName", service.Name)
	}
	return nil
}

func (n *Nacos) generateNamespaceClient(namespace string) (naming_client.INamingClient, error) {

	if client, ok := n.namespaceClients[namespace]; ok {
		return client, nil
	}

	clientConfig := constant.ClientConfig{
		NamespaceId:         namespace,
		TimeoutMs:           5000,
		NotLoadCacheAtStart: true,
		LogDir:              "/tmp/nacos/log",
		CacheDir:            "/tmp/nacos/cache",
		LogLevel:            "debug",
	}

	if n.username != "" && n.password != "" {
		clientConfig.Username = n.username
		clientConfig.Password = n.password
	}

	serverConfigs := []constant.ServerConfig{
		{
			IpAddr: n.address,
			Port:   uint64(n.port),
		},
	}

	client, err := clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  &clientConfig,
			ServerConfigs: serverConfigs,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create namespace client for %s: %v", namespace, err)
	}

	return client, nil
}
