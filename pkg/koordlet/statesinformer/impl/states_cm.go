package impl

import (
	"context"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/klog/v2"

	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

const (
	cmInformerName PluginName = "cmInformer"
)

type cmInformer struct {
	cmInformer cache.SharedIndexInformer
	cmRWMutex  sync.RWMutex
	cm         *corev1.ConfigMap
}

func NewCmInformer() *cmInformer {
	return &cmInformer{}
}

func (s *cmInformer) GetCm() *corev1.ConfigMap {
	s.cmRWMutex.RLock()
	defer s.cmRWMutex.RUnlock()
	if s.cm == nil {
		return nil
	}
	return s.cm.DeepCopy()
}

func newCmInformer(client clientset.Interface) cache.SharedIndexInformer {
	tweakListOptionsFunc := func(opt *metav1.ListOptions) {
		opt.FieldSelector = "metadata.name=cfs-quota-burst-cm"
	}
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (apiruntime.Object, error) {
				tweakListOptionsFunc(&options)
				return client.CoreV1().ConfigMaps("koordinator-system").List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				tweakListOptionsFunc(&options)
				return client.CoreV1().ConfigMaps("koordinator-system").Watch(context.TODO(), options)
			},
		},
		&corev1.ConfigMap{},
		time.Hour*12,
		cache.Indexers{},
	)
}

func (s *cmInformer) Setup(ctx *PluginOption, state *PluginState) {
	s.cmInformer = newCmInformer(ctx.KubeClient)
	s.cmInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			cm, ok := obj.(*corev1.ConfigMap)
			if ok {
				s.syncCm(cm)
			} else {
				klog.Errorf("cm informer add func parse ConfigMap failed, obj %T", obj)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldCm, oldOK := oldObj.(*corev1.ConfigMap)
			newCm, newOK := newObj.(*corev1.ConfigMap)
			if !oldOK || !newOK {
				klog.Errorf("unable to convert object to *corev1.ConfigMap, old %T, new %T", oldObj, newObj)
				return
			}
			if newCm.ResourceVersion == oldCm.ResourceVersion {
				klog.V(5).Infof("find configmap %s has not changed", newCm.Name)
				return
			}
			s.syncCm(newCm)
		},
	})
}

func (s *cmInformer) syncCm(newCm *corev1.ConfigMap) {
	klog.V(5).Infof("configmap update detail %v", newCm)
	s.cmRWMutex.Lock()
	defer s.cmRWMutex.Unlock()

	s.cm = newCm.DeepCopy()
	klog.V(5).Infof("Configmap is ", s.cm)
}

func (s *cmInformer) Start(stopCh <-chan struct{}) {
	klog.V(2).Infof("starting configmap informer")
	go s.cmInformer.Run(stopCh)
	klog.V(2).Infof("cm informer started")
}

func (s *cmInformer) HasSynced() bool {
	if s.cmInformer == nil {
		return false
	}
	synced := s.cmInformer.HasSynced()
	klog.V(5).Infof("cm informer has synced %v", synced)
	return synced
}
