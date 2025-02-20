package infrapod

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

var c client.Client

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
