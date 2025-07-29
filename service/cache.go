package service

import (
	"sync"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Cache struct {
	initialize sync.Once

	configmaps map[types.NamespacedName]*corev1.ConfigMap
	services   map[types.NamespacedName]*corev1.Service
	nodes      map[types.NamespacedName]*corev1.Node
}

func (c *Cache) init() {
	c.configmaps = make(map[types.NamespacedName]*corev1.ConfigMap)
	c.services = make(map[types.NamespacedName]*corev1.Service)
	c.nodes = make(map[types.NamespacedName]*corev1.Node)
}

func (c *Cache) Insert(obj interface{}) bool {
	c.initialize.Do(c.init)

	switch o := obj.(type) {
	case *corev1.ConfigMap:
		if o.Labels != nil && o.Labels["nacosbridge.io/config"] == "true" {
			c.configmaps[NamespacedName(o)] = o
			return true
		}
		return false
	case *corev1.Service:
		c.services[NamespacedName(o)] = o
	case *corev1.Node:
		c.nodes[NamespacedName(o)] = o
	default:
		return false
	}
	return true
}

func (c *Cache) Delete(obj interface{}) bool {
	c.initialize.Do(c.init)

	switch o := obj.(type) {
	case *corev1.ConfigMap:
		if _, ok := c.configmaps[NamespacedName(o)]; ok {
			delete(c.configmaps, NamespacedName(o))
			return true
		}
	case *corev1.Service:
		if _, ok := c.services[NamespacedName(o)]; ok {
			delete(c.services, NamespacedName(o))
			return true
		}
	case *corev1.Node:
		if _, ok := c.nodes[NamespacedName(o)]; ok {
			delete(c.nodes, NamespacedName(o))
			return true
		}
	}
	return false
}

func NamespacedName(obj client.Object) types.NamespacedName {
	return types.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}
}
