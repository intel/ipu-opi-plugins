package infrapod

import (
	"embed"
	"fmt"
	"os"
	"time"

	"github.com/bombsimon/logrusr/v4"
	"github.com/go-logr/logr"
	"github.com/intel/ipu-opi-plugins/ipu-plugin/pkg/k8s/render"
	"github.com/intel/ipu-opi-plugins/ipu-plugin/pkg/types"
	logrus "github.com/sirupsen/logrus"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	"sigs.k8s.io/controller-runtime/pkg/cache"
)

//go:embed bindata/*
var binData embed.FS

type InfrapodMgrOcImpl struct {
	log           logr.Logger
	mgr           manager.Manager
	vspP4Template VspP4TemplateVars
}
type VspP4TemplateVars struct {
	ImageName string
	Namespace string
	HostName  string
}

func (v VspP4TemplateVars) ToMap() map[string]string {
	return map[string]string{
		"ImageName": v.ImageName,
		"Namespace": v.Namespace,
		"HostName":  v.HostName,
	}
}

func NewVspP4TemplateVars(imageName string, namespace string) (VspP4TemplateVars, error) {
	hostName, err := os.Hostname()
	if err != nil {
		return VspP4TemplateVars{}, fmt.Errorf("Failed to get error hostname: %v", err)
	}
	return VspP4TemplateVars{
		ImageName: imageName,
		Namespace: namespace,
		HostName:  hostName,
	}, nil
}

func NewInfrapodMgr(imageName string, namespace string) (types.InfrapodMgr, error) {
	// TODO: refactor entire logging framework to use a logr
	// We are using https://github.com/bombsimon/logrusr temporarily
	// here which is a logr implementation of logrus
	logrusLog := logrus.New()
	log := logrusr.New(logrusLog)
	vspP4template, err := NewVspP4TemplateVars(imageName, namespace)
	if err != nil {
		log.Error(err, "unable to get template vars : %v", err)
		return nil, err
	}
	// The duration below indicates the amount of time the pod
	// should wait before starting again
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
		log.Error(err, "unable to start manager :%v", err)
		return nil, err
	}
	return &InfrapodMgrOcImpl{
		log:           log,
		mgr:           mgr,
		vspP4Template: vspP4template,
	}, nil
}

func (infrapodMgr *InfrapodMgrOcImpl) CreateCrs() error {
	// Create p4 pod
	// This will create ->
	//  ServiceAccount
	//  role
	//  rolebindings
	//  persistentvolumes
	//  persistentvolumeclaims
	//  service for p4runtime
	//  P4 pod
	err := render.OperateAllFromBinData(infrapodMgr.log, "vsp-p4",
		infrapodMgr.vspP4Template.ToMap(), binData, infrapodMgr.mgr.GetClient(),
		nil, infrapodMgr.mgr.GetScheme(), false)
	if err != nil {
		infrapodMgr.log.Error(err, "failed to start vendor plugin container")
		return fmt.Errorf("failed to start vendor plugin container (p4Image:%s) due to: %v", infrapodMgr.vspP4Template.ImageName, err)
	}
	return nil
}
func (infrapodMgr *InfrapodMgrOcImpl) DeleteCrs() error {
	// Delete p4 pod
	// This will delete ->
	//  ServiceAccount
	//  role
	//  rolebindings
	//  persistentvolumes
	//  persistentvolumeclaims
	//  service for p4runtime
	//  P4 pod
	err := render.OperateAllFromBinData(infrapodMgr.log, "vsp-p4",
		infrapodMgr.vspP4Template.ToMap(), binData, infrapodMgr.mgr.GetClient(),
		nil, infrapodMgr.mgr.GetScheme(), true)
	if err != nil {
		infrapodMgr.log.Error(err, "failed to start vendor plugin container")
		return fmt.Errorf("failed to start vendor plugin container (p4Image:%s) due to: %v", infrapodMgr.vspP4Template.ImageName, err)
	}
	return nil
}
