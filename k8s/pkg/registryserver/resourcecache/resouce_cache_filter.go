package resourcecache

import "github.com/sirupsen/logrus"

//CacheFilterPolicy uses for filtering resources
type CacheFilterPolicy interface {
	//Filter means resource filter function. Accepts resource. Returns true if the resource should be skipped
	Filter(resource interface{}) bool
}

//NoFilterPolicy returns default filter policy
func NoFilterPolicy() CacheFilterPolicy {
	return cacheFilterFunc(func(obj interface{}) bool {
		return false
	})
}

//FilterByNamespacePolicy returns policy for filtering resources by namespace
func FilterByNamespacePolicy(ns string, nsGetter func(resource interface{}) string) CacheFilterPolicy {
	return cacheFilterFunc(func(obj interface{}) bool {
		logrus.Infof("Start apply filter by namespace %v for %v", ns, obj)
		return ns != nsGetter(obj)
	})
}

type cacheFilterFunc func(obj interface{}) bool

func (f cacheFilterFunc) Filter(resource interface{}) bool {
	return f(resource)
}
