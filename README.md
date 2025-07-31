# NacosBridge

[![Go Report Card](https://goreportcard.com/badge/github.com/your-org/nacosbridge)](https://goreportcard.com/report/github.com/your-org/nacosbridge)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

NacosBridge is a Kubernetes Operator that automatically syncs Kubernetes Service resources to Nacos service registry. It provides a bridge between Kubernetes and Nacos for unified service discovery in microservices architecture.

## Features

- 🔄 **Auto Service Sync**: Automatically monitors Kubernetes Service changes and syncs to Nacos
- 🏷️ **Label Filtering**: Filter services using label selectors
- 📊 **Metrics**: Prometheus-compatible monitoring metrics
- 🔐 **Authentication**: Support for Nacos username/password auth
- 🏗️ **Multi-namespace**: Support for Nacos multi-namespace management
- 🐳 **Containerized**: Complete Docker image and Kubernetes deployment configs
- 🔧 **Flexible Config**: Configurable via ConfigMap

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Kubernetes    │    │   NacosBridge   │    │     Nacos       │
│   Service       │───▶│   Controller    │───▶│   Registry      │
│   Resources     │    │                 │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Quick Start

### Prerequisites

- Kubernetes 1.20+
- Go 1.24+
- Nacos 2.0+

### Local Development

1. **Clone Project**
```bash
git clone https://github.com/your-org/nacosbridge.git
cd nacosbridge
```

2. **Install Dependencies**
```bash
go mod download
```

3. **Generate Code**
```bash
make generate
```

4. **Run Tests**
```bash
make test
```

5. **Local Run**
```bash
make run
```

### Build Deployment

1. **Build Binary**
```bash
make build
```

2. **Build Docker Image**
```bash
make docker
```

3. **Push Image**
```bash
make docker-push
```

4. **Deploy to Kubernetes**
```bash
make install
```

## Configuration

### Nacos Configuration

Configure Nacos connection via ConfigMap:

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

### Service Label Configuration

Add labels to services that need to be synced to Nacos:

#### Cluster Services (Cluster Services)
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

#### External Services (External Services)
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

#### Gateway Services (Gateway Services)
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

### Supported Labels

| Label | Description | Default |
|-------|-------------|---------|
| `nacosbridge.io/service` | Custom service name | Service name |
| `nacosbridge.io/namespace` | Nacos namespace | `public` |
| `nacosbridge.io/service-type` | Service type (cluster/external/gateway) | - |
| `nacosbridge.io/external` | External service identifier | - |
| `nacosbridge.io/openport` | Open port names (comma-separated) | - |
| `nacosbridge.io/portname-{portName}` | Custom service name for a specific port | - |
| `nacosbridge.io/service-domin` | Gateway service domain name | - |
| `nacosbridge.io/service-domain-{port}` | Gateway service port configuration | - |
| `nacosbridge.io/matedata` | Service metadata prefix (from annotations) | - |

### Annotation Configuration

In addition to labels, NacosBridge also supports configuring service metadata via annotations:

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

These annotations are converted to the following metadata:
- `version: v1.0.0`
- `environment: production`
- `team: backend`
- `description: User service API`

### Service Type Description

NacosBridge supports three service types:

#### 1. Cluster Service (Cluster Services)
- **Purpose**: Registers Kubernetes cluster services to Nacos
- **Features**: Uses cluster internal domain (`service.namespace.svc.cluster.local`)
- **Applicable Scenarios**: Internal communication between microservices

#### 2. External Service (External Services)
- **Purpose**: Registers external services to Nacos
- **Supported Types**: 
  - `LoadBalancer`: Uses LoadBalancerIP
  - `ExternalName`: Uses ExternalName
  - `NodePort`: Uses node IP and NodePort
- **Applicable Scenarios**: External API services, third-party services

#### 3. Gateway Service (Gateway Services)
- **Purpose**: Registers gateway services to Nacos
- **Features**: Uses custom domain and port configuration
- **Applicable Scenarios**: API gateways, entry services

## Monitoring Metrics

NacosBridge provides the following Prometheus metrics:

- `nacosbridge_service_sync_total`: Total service syncs
- `nacosbridge_service_sync_success_total`: Successful syncs
- `nacosbridge_service_sync_failure_total`: Failed syncs
- `nacosbridge_nacos_connection_status`: Nacos connection status

Access `http://localhost:9090/metrics` to view full metrics.

## Development Guide

### Project Structure

```
nacosbridge/
├── cmd/                    # Main program entry
│   └── main.go
├── controller/             # Kubernetes controller
│   ├── configmap.go       # ConfigMap controller
│   ├── node.go            # Node controller
│   └── service.go         # Service controller
├── service/               # Business logic layer
│   ├── nacos.go          # Nacos client
│   ├── server.go         # HTTP service
│   ├── metric.go         # Monitoring metrics
│   └── cache.go          # Cache management
├── config/               # Kubernetes configuration
│   ├── manager/          # Deployment configuration
│   ├── rbac/             # RBAC configuration
│   └── docker/           # Docker configuration
└── bin/                  # Build artifacts
```

### Adding New Controllers

1. Create a new controller file in the `controller/` directory
2. Implement the `Reconcile` method
3. Register the controller in `main.go`

### Adding New Service Providers

1. Implement the service provider interface in the `service/` directory
2. Implement `Config` and `Build` methods
3. Register the new provider in the service factory

## Troubleshooting

### Common Issues

1. **Services Cannot Sync to Nacos**
   - Check Nacos connection configuration
   - Confirm correct Service label configuration
   - View controller logs

2. **Permission Issues**
   - Confirm correct RBAC configuration
   - Check ServiceAccount permissions

3. **Connection Timeout**
   - Check network connectivity
   - Confirm Nacos service status

### Log Viewing

```bash
# View controller logs
kubectl logs -f deployment/controller -n system

# View events
kubectl get events -n system --sort-by='.lastTimestamp'
```

## Contributing

1. Fork the project
2. Create a feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Create a Pull Request

## License

This project is licensed under the Apache 2.0 License - see [LICENSE](LICENSE) for details.
