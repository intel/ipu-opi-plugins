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
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:embed bindata/*
var binData embed.FS

var c client.Client

/*
func CreateInfrapod() error {
	// Using a typed object.
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "infrap4d",
			Name:      "infra",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Image: "vsp-p4",
					Name:  "vsp-p4",
				},
			},
		},
	}
	// c is a created client.
	_ = c.Create(context.Background(), pod)
	return nil
}
*/

func CreateInfrapod() error {
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

	// Create p4 pvc
	//	err := render.ApplyAllFromBinData(log, "vsp-p4", nil, binData, c, nil, scheme.Scheme)
	//	if err != nil {
	//		return fmt.Errorf("failed to start vendor plugin container (p4Image): %v", err)
	//	}
	// Create p4 pod
	err = render.ApplyAllFromBinData(log, "vsp-p4", nil, binData, mgr.GetClient(), nil, mgr.GetScheme())
	if err != nil {
		return fmt.Errorf("failed to start vendor plugin container (p4Image): %v", err)
	}
	return nil
}
