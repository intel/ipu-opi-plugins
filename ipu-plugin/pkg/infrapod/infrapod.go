package infrapod

import (
	"embed"
	"fmt"
	"time"

	"github.com/bombsimon/logrusr/v4"
	"github.com/intel/ipu-opi-plugins/ipu-plugin/pkg/k8s/render"
	logrus "github.com/sirupsen/logrus"
	ctrl "sigs.k8s.io/controller-runtime"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	"sigs.k8s.io/controller-runtime/pkg/cache"
)

//go:embed bindata/*
var binData embed.FS

type VspP4TemplateVars struct {
	ImageName string
	Namespace string
}

func (v VspP4TemplateVars) ToMap() map[string]string {
	return map[string]string{
		"ImageName": v.ImageName,
		"Namespace": v.Namespace,
	}
}

func NewVspP4TemplateVars(imageName string, namespace string) VspP4TemplateVars {
	return VspP4TemplateVars{
		ImageName: imageName,
		Namespace: namespace,
	}
}

func CreateInfrapod(imageName string, namespace string) error {
	logrusLog := logrus.New()
	log := logrusr.New(logrusLog)
	t := time.Duration(0)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme.Scheme,
		NewCache: func(config *rest.Config, opts cache.Options) (cache.Cache, error) {
			opts.DefaultNamespaces = map[string]cache.Config{
				"dpu-p4-infra": {},
			}
			return cache.New(config, opts)
		},
		// A timout needs to be specified, or else the mananger will wait indefinitely on stop()
		GracefulShutdownTimeout: &t,
	})
	if err != nil {
		log.Error(err, "unable to start manager")
		return err
	}
	vspP4template := NewVspP4TemplateVars(imageName, namespace)

	// Create p4 pod
	// This will create the ServiceAccount, role, rolebindings, and the service for p4runtime
	err = render.ApplyAllFromBinData(log, "vsp-p4",
		vspP4template.ToMap(), binData, mgr.GetClient(),
		nil, mgr.GetScheme())
	if err != nil {
		return fmt.Errorf("failed to start vendor plugin container (p4Image): %v", err)
	}
	return nil
}
