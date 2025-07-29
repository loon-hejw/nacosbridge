package service

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	DELAY = 10 * time.Second
)

type Registry interface {
	Name() string
	Config(config map[string]string) error
	Build(services []Service) error
}

type opAdd struct {
	obj interface{}
}

type opDelete struct {
	obj interface{}
}

type Server struct {
	cache         *Cache
	updateChan    chan interface{}
	svcRegistry   []Registry
	logger        logr.Logger
	statusUpdater StatusUpdater
}

func NewService(statusUpdater StatusUpdater) *Server {
	return &Server{
		cache:      &Cache{},
		updateChan: make(chan interface{}),
		svcRegistry: []Registry{
			&Nacos{},
		},
		logger:        log.Log.WithName("service"),
		statusUpdater: statusUpdater,
	}
}

func (s *Server) OnAdd(obj interface{}, isInInitialList bool) {
	s.updateChan <- opAdd{obj: obj}
}

func (s *Server) OnUpdate(oldObj, newObj interface{}) {
	s.updateChan <- opAdd{obj: newObj}
}

func (s *Server) OnDelete(obj interface{}) {
	s.updateChan <- opDelete{obj: obj}
}

func (s *Server) Start(ctx context.Context) error {

	var (
		pending <-chan time.Time
		t       *time.Timer
	)

	for {
		select {
		case <-ctx.Done():
			return nil
		case op := <-s.updateChan:
			if s.onUpdate(op) {
				if t != nil {
					t.Stop()
				}
				t = time.NewTimer(DELAY)
				pending = t.C
			}
		case <-pending:
			if err := s.rebuild(); err != nil {
				s.logger.Error(err, "failed to rebuild")
			}
		}
	}
}

func (s *Server) onUpdate(obj interface{}) bool {
	switch obj := obj.(type) {
	case opAdd:
		return s.cache.Insert(obj.obj)
	case opDelete:
		return s.cache.Delete(obj.obj)
	}
	return false
}

func (s *Server) rebuild() error {

	var config string
	for _, c := range s.cache.configmaps {
		if c.Labels != nil && c.Labels["nacosbridge.io/config"] == "true" {
			content, ok := c.Data["config.json"]
			if !ok {
				continue
			}
			config = content
			break
		}
	}

	if config == "" {
		return fmt.Errorf("config not found")
	}

	registryConfig := &Config{}
	if err := registryConfig.Load(config); err != nil {
		return err
	}

	sus := make([]StatusUpdate, 0)
	for nn, svc := range s.cache.services {

		if svc.Labels == nil {
			continue
		}
		if _, ok := svc.Labels["nacosbridge.io/service"]; !ok {
			continue
		}

		mutator := func(obj client.Object) client.Object {
			svc := obj.(*corev1.Service)
			if svc.Labels == nil || svc.Labels["nacosbridge.io/external"] != "true" {
				return svc
			}
			cp := svc.DeepCopy()
			for k, specPort := range cp.Spec.Ports {
				nodePort, ok := svc.Labels["nacosbridge.io/nodeport-"+specPort.Name]
				if !ok {
					continue
				}
				port, err := strconv.Atoi(nodePort)
				if err != nil {
					continue
				}
				cp.Spec.Ports[k].NodePort = int32(port)
			}
			if reflect.DeepEqual(cp, svc) {
				return svc
			}
			if cp.Labels == nil {
				cp.Labels = make(map[string]string)
			}
			cp.Labels["nacosbridge.io/spec-changed"] = "true"
			cp.Spec.Type = corev1.ServiceTypeNodePort
			return cp
		}

		sus = append(sus, StatusUpdate{
			NamespacedName: nn,
			Resource:       svc,
			Mutator:        StatusMutatorFunc(mutator),
		})
	}

	for _, su := range sus {
		s.statusUpdater.Send(su)
	}

	for _, sr := range s.svcRegistry {
		ns := registryConfig.WatchNamespace[sr.Name()]
		selectNamespace := make(map[string]bool)
		if ns != "" {
			for _, n := range strings.Split(ns, ",") {
				selectNamespace[n] = true
			}
		}
		if err := sr.Config(GenerateServiceConfig(sr.Name(), registryConfig.ServiceConfig)); err != nil {
			s.logger.Error(err, "failed to config", "service", sr.Name())
			continue
		}

		nodeIps := make([]string, 0)
		for _, node := range s.cache.nodes {
			for _, address := range node.Status.Addresses {
				if address.Type == corev1.NodeInternalIP {
					nodeIps = append(nodeIps, address.Address)
				}
			}
		}

		serviceInfos := make([]Service, 0)
		for _, svc := range s.cache.services {
			serviceInfos = append(serviceInfos, GenerateServiceInfos(svc, nodeIps, selectNamespace)...)
		}
		if err := sr.Build(serviceInfos); err != nil {
			return err
		}
	}
	return nil
}
