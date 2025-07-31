# NacosBridge

[![Go Report Card](https://goreportcard.com/badge/github.com/your-org/nacosbridge)](https://goreportcard.com/report/github.com/your-org/nacosbridge)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

NacosBridge 是一个 Kubernetes Operator，用于将 Kubernetes 集群中的 Service 资源自动同步到 Nacos 服务注册中心。它提供了 Kubernetes 和 Nacos 之间的桥接功能，使得微服务架构中的服务发现更加统一和便捷。

## 功能特性

- 🔄 **自动服务同步**: 自动监听 Kubernetes Service 资源变化，实时同步到 Nacos
- 🏷️ **标签过滤**: 支持通过标签选择器过滤需要同步的 Service
- 📊 **监控指标**: 提供 Prometheus 格式的监控指标
- 🔐 **安全认证**: 支持 Nacos 的用户名密码认证
- 🏗️ **多命名空间**: 支持 Nacos 多命名空间管理
- 🐳 **容器化部署**: 提供完整的 Docker 镜像和 Kubernetes 部署配置
- 🔧 **配置灵活**: 支持通过 ConfigMap 进行灵活配置

## 架构设计

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Kubernetes    │    │   NacosBridge   │    │     Nacos       │
│   Service       │───▶│   Controller    │───▶│   Registry      │
│   Resources     │    │                 │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## 快速开始

### 前置要求

- Kubernetes 1.20+
- Go 1.24+
- Nacos 2.0+

### 本地开发

1. **克隆项目**
```bash
git clone https://github.com/your-org/nacosbridge.git
cd nacosbridge
```

2. **安装依赖**
```bash
go mod download
```

3. **生成代码**
```bash
make generate
```

4. **运行测试**
```bash
make test
```

5. **本地运行**
```bash
make run
```

### 构建部署

1. **构建二进制文件**
```bash
make build
```

2. **构建 Docker 镜像**
```bash
make docker
```

3. **推送镜像**
```bash
make docker-push
```

4. **部署到 Kubernetes**
```bash
make install
```

## 配置说明

### Nacos 配置

通过 ConfigMap 配置 Nacos 连接信息：

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: nacosbridge-config
  namespace: system
data:
  nacos.address: "nacos-server.example.com"
  nacos.port: "8848"
  nacos.username: "nacos"
  nacos.password: "nacos"
  nacos.namespace: "public"
```

### Service 标签配置

为需要同步到 Nacos 的 Service 添加标签：

#### 集群内服务 (Cluster Service)
```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
  namespace: default
  labels:
    nacosbridge.io/service: "my-custom-service"
    nacosbridge.io/namespace: "my-namespace"
    nacosbridge.io/service-type: "cluster"
    nacosbridge.io/openport: "http,https"
    nacosbridge.io/portname-http: "my-http-service"
    nacosbridge.io/portname-https: "my-https-service"
  annotations:
    nacosbridge.io/matedata.version: "v1.0.0"
    nacosbridge.io/matedata.environment: "production"
spec:
  selector:
    app: my-app
  ports:
  - name: http
    port: 80
    targetPort: 8080
  - name: https
    port: 443
    targetPort: 8443
```

#### 外部服务 (External Service)
```yaml
apiVersion: v1
kind: Service
metadata:
  name: external-service
  namespace: default
  labels:
    nacosbridge.io/service: "external-api"
    nacosbridge.io/namespace: "external"
    nacosbridge.io/service-type: "external"
    nacosbridge.io/openport: "api"
spec:
  type: LoadBalancer
  loadBalancerIP: "192.168.1.100"
  ports:
  - name: api
    port: 8080
    targetPort: 8080
```

#### 网关服务 (Gateway Service)
```yaml
apiVersion: v1
kind: Service
metadata:
  name: gateway-service
  namespace: default
  labels:
    nacosbridge.io/service: "api-gateway"
    nacosbridge.io/namespace: "gateway"
    nacosbridge.io/service-type: "gateway"
    nacosbridge.io/service-domin: "api.example.com"
    nacosbridge.io/service-domain-80: "80"
    nacosbridge.io/service-domain-443: "443"
spec:
  selector:
    app: gateway
  ports:
  - port: 80
    targetPort: 8080
  - port: 443
    targetPort: 8443
```

### 支持的标签

| 标签 | 说明 | 默认值 |
|------|------|--------|
| `nacosbridge.io/service` | 自定义服务名称 | Service 名称 |
| `nacosbridge.io/namespace` | Nacos 命名空间 | `public` |
| `nacosbridge.io/service-type` | 服务类型 (cluster/external/gateway) | - |
| `nacosbridge.io/external` | 外部服务标识 | - |
| `nacosbridge.io/openport` | 开放的端口名称 (逗号分隔) | - |
| `nacosbridge.io/portname-{portName}` | 为指定端口自定义服务名 | - |
| `nacosbridge.io/service-domin` | 网关服务域名 | - |
| `nacosbridge.io/service-domain-{port}` | 网关服务端口配置 | - |
| `nacosbridge.io/matedata` | 服务元数据前缀（从注解获取） | - |

### 注解配置

除了标签外，NacosBridge 还支持通过注解来配置服务元数据：

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
  namespace: default
  labels:
    nacosbridge.io/service: "my-service"
    nacosbridge.io/service-type: "cluster"
    nacosbridge.io/openport: "http"
  annotations:
    nacosbridge.io/matedata.version: "v1.0.0"
    nacosbridge.io/matedata.environment: "production"
    nacosbridge.io/matedata.team: "backend"
    nacosbridge.io/matedata.description: "User service API"
spec:
  selector:
    app: my-app
  ports:
  - name: http
    port: 80
    targetPort: 8080
```

上述注解会被转换为以下元数据：
- `version: v1.0.0`
- `environment: production`
- `team: backend`
- `description: User service API`

### 服务类型说明

NacosBridge 支持三种服务类型：

#### 1. Cluster Service (集群内服务)
- **用途**: 将 Kubernetes 集群内的服务注册到 Nacos
- **特点**: 使用集群内域名 (`service.namespace.svc.cluster.local`)
- **适用场景**: 微服务间的内部通信

#### 2. External Service (外部服务)
- **用途**: 将外部服务注册到 Nacos
- **支持类型**: 
  - `LoadBalancer`: 使用 LoadBalancerIP
  - `ExternalName`: 使用 ExternalName
  - `NodePort`: 使用节点 IP 和 NodePort
- **适用场景**: 外部 API 服务、第三方服务

#### 3. Gateway Service (网关服务)
- **用途**: 将网关服务注册到 Nacos
- **特点**: 使用自定义域名和端口配置
- **适用场景**: API 网关、入口服务

## 监控指标

NacosBridge 提供以下 Prometheus 指标：

- `nacosbridge_service_sync_total`: 服务同步总次数
- `nacosbridge_service_sync_success_total`: 成功同步次数
- `nacosbridge_service_sync_failure_total`: 同步失败次数
- `nacosbridge_nacos_connection_status`: Nacos 连接状态

访问 `http://localhost:9090/metrics` 查看完整指标。

## 开发指南

### 项目结构

```
nacosbridge/
├── cmd/                    # 主程序入口
│   └── main.go
├── controller/             # Kubernetes 控制器
│   ├── configmap.go       # ConfigMap 控制器
│   ├── node.go            # Node 控制器
│   └── service.go         # Service 控制器
├── service/               # 业务逻辑层
│   ├── nacos.go          # Nacos 客户端
│   ├── server.go         # HTTP 服务
│   ├── metric.go         # 监控指标
│   └── cache.go          # 缓存管理
├── config/               # Kubernetes 配置
│   ├── manager/          # 部署配置
│   ├── rbac/             # 权限配置
│   └── docker/           # Docker 配置
└── bin/                  # 构建产物
```

### 添加新的控制器

1. 在 `controller/` 目录下创建新的控制器文件
2. 实现 `Reconcile` 方法
3. 在 `main.go` 中注册控制器

### 添加新的服务提供者

1. 在 `service/` 目录下实现服务提供者接口
2. 实现 `Config` 和 `Build` 方法
3. 在服务工厂中注册新的提供者

## 故障排查

### 常见问题

1. **服务无法同步到 Nacos**
   - 检查 Nacos 连接配置
   - 确认 Service 标签配置正确
   - 查看控制器日志

2. **权限问题**
   - 确认 RBAC 配置正确
   - 检查 ServiceAccount 权限

3. **连接超时**
   - 检查网络连通性
   - 确认 Nacos 服务状态

### 日志查看

```bash
# 查看控制器日志
kubectl logs -f deployment/controller -n system

# 查看事件
kubectl get events -n system --sort-by='.lastTimestamp'
```

## 贡献指南

1. Fork 项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 创建 Pull Request

## 许可证

本项目采用 Apache 2.0 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。