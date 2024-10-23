package webhook

import (
	"context"
	"fmt"
	"net/http"
	"time"

	webhookutil "github.com/clay-wangzhi/koordinator/pkg/webhook/util"
	"github.com/clay-wangzhi/koordinator/pkg/webhook/util/framework"
	"github.com/clay-wangzhi/koordinator/pkg/webhook/util/health"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	webhookcontroller "github.com/clay-wangzhi/koordinator/pkg/webhook/util/controller"
	"sigs.k8s.io/controller-runtime/pkg/webhook/conversion"

	"k8s.io/klog/v2"
)

type GateFunc func() (enabled bool)

var (
	// handlerMap contains all admission webhook handlers.
	handlerMap        = map[string]admission.Handler{}
	handlerGates      = map[string]GateFunc{}
	HandlerBuilderMap = map[string]framework.HandlerBuilder{}
)

func filterActiveHandlers() {
	disablePaths := sets.NewString()
	for path := range HandlerBuilderMap {
		if fn, ok := handlerGates[path]; ok {
			if !fn() {
				disablePaths.Insert(path)
			}
		}
	}
	for _, path := range disablePaths.List() {
		delete(HandlerBuilderMap, path)
	}
}

func SetupWithWebhookOpt(opt *manager.Options) {
	opt.WebhookServer = webhook.NewServer(webhook.Options{
		Host:    "0.0.0.0",
		Port:    webhookutil.GetPort(),
		CertDir: webhookutil.GetCertDir(),
	})
}

func SetupWithManager(mgr manager.Manager) error {
	server := mgr.GetWebhookServer()

	// register admission handlers
	filterActiveHandlers()
	for path, handlerBuilder := range HandlerBuilderMap {
		handler := handlerBuilder.WithControllerManager(mgr).Build()
		server.Register(path, &webhook.Admission{Handler: handler})
		handlerMap[path] = handler
		klog.V(3).Infof("Registered webhook handler %s", path)
	}

	// register conversion webhook
	server.Register("/convert", conversion.NewWebhookHandler(mgr.GetScheme()))

	// register health handler
	server.Register("/healthz", &health.Handler{})

	return nil
}

// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=mutatingwebhookconfigurations,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=validatingwebhookconfigurations,verbs=get;list;watch;update;patch

func Initialize(ctx context.Context, cfg *rest.Config) error {
	c, err := webhookcontroller.New(cfg, handlerMap)
	if err != nil {
		return err
	}
	go func() {
		c.Start(ctx)
	}()

	timer := time.NewTimer(time.Second * 20)
	defer timer.Stop()
	select {
	case <-webhookcontroller.Inited():
		return nil
	case <-timer.C:
		return fmt.Errorf("failed to start webhook controller for waiting more than 20s")
	}
}

func Checker(req *http.Request) error {
	// Firstly wait webhook controller initialized
	select {
	case <-webhookcontroller.Inited():
	default:
		return fmt.Errorf("webhook controller has not initialized")
	}
	return health.Checker(req)
}

func WaitReady() error {
	startTS := time.Now()
	var err error
	for {
		duration := time.Since(startTS)
		if err = Checker(nil); err == nil {
			return nil
		}

		if duration > time.Second*5 {
			klog.Warningf("Failed to wait webhook ready over %s: %v", duration, err)
		}
		time.Sleep(time.Second * 2)
	}

}

func addHandlersWithGate(m map[string]framework.HandlerBuilder, fn GateFunc) {
	for path, handlerBuilder := range m {
		if len(path) == 0 {
			klog.Warningf("Skip handler with empty path.")
			continue
		}
		if path[0] != '/' {
			path = "/" + path
		}
		_, found := HandlerBuilderMap[path]
		if found {
			klog.V(1).Infof("conflicting webhook builder path %v in handler map", path)
		}
		HandlerBuilderMap[path] = handlerBuilder
		if fn != nil {
			handlerGates[path] = fn
		}
	}
}
