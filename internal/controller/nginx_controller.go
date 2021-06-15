package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	examplev1alpha1 "github.com/thetechnick/example-operator/apis/example/v1alpha1"
)

type NginxReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (r *NginxReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&examplev1alpha1.Nginx{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

func (r *NginxReconciler) Reconcile(
	ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("Nginx", req.NamespacedName.String())

	nginx := &examplev1alpha1.Nginx{}
	if err := r.Get(ctx, req.NamespacedName, nginx); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	labels := map[string]string{
		"app.kubernetes.io/name":       "nginx",
		"app.kubernetes.io/version":    nginx.Spec.Version,
		"app.kubernetes.io/managed-by": "nginx-operator",
	}

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx-" + nginx.Name,
			Namespace: nginx.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:" + nginx.Spec.Version,
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}
	if err := controllerutil.SetControllerReference(nginx, deploy, r.Scheme); err != nil {
		return ctrl.Result{}, fmt.Errorf("setting ownerref: %w", err)
	}

	currentDeploy, err := reconcileDeployment(ctx, log, r.Client, deploy)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("reconciling deployment: %w", err)
	}

	if currentDeploy.Status.ObservedGeneration != currentDeploy.Generation {
		// we don't know whats up - status needs to catch up
		return ctrl.Result{}, nil
	}

	var available bool
	for _, c := range currentDeploy.Status.Conditions {
		if c.Type == appsv1.DeploymentAvailable &&
			c.Status == corev1.ConditionTrue {
			available = true
			break
		}
	}

	if available {
		meta.SetStatusCondition(&nginx.Status.Conditions, metav1.Condition{
			Type:               examplev1alpha1.NginxAvailable,
			Status:             metav1.ConditionTrue,
			Reason:             "Setup",
			Message:            "Nginx is up and running.",
			ObservedGeneration: nginx.Generation,
		})
		nginx.Status.Phase = examplev1alpha1.NginxPhaseReady
	} else {
		meta.SetStatusCondition(&nginx.Status.Conditions, metav1.Condition{
			Type:               examplev1alpha1.NginxAvailable,
			Status:             metav1.ConditionFalse,
			Reason:             "NotAvailable",
			Message:            "Nginx deployment is not available.",
			ObservedGeneration: nginx.Generation,
		})
		nginx.Status.Phase = examplev1alpha1.NginxPhaseNotReady
	}
	nginx.Status.ObservedGeneration = nginx.Generation

	// Simulate "hard work being done"
	time.Sleep(nginx.Spec.ReconcileDelay.Duration)

	return ctrl.Result{}, r.Status().Update(ctx, nginx)
}

// Deployment reconciles a apps/v1, Kind=Deployment.
func reconcileDeployment(
	ctx context.Context,
	log logr.Logger,
	c client.Client,
	desiredDeployment *appsv1.Deployment,
) (currentDeployment *appsv1.Deployment, err error) {
	name := types.NamespacedName{
		Name:      desiredDeployment.Name,
		Namespace: desiredDeployment.Namespace,
	}

	// Lookup current version of the object
	currentDeployment = &appsv1.Deployment{}
	err = c.Get(ctx, name, currentDeployment)
	if err != nil && !errors.IsNotFound(err) {
		// unexpected error
		return nil, fmt.Errorf("getting Deployment: %w", err)
	}

	// Keep the replicas resource cleaned
	// telepresence scales the original deployment to 0 replicas before starting a new one
	if currentDeployment.Spec.Replicas != nil {
		desiredDeployment.Spec.Replicas = currentDeployment.Spec.Replicas
	}

	if errors.IsNotFound(err) {
		// Deployment needs to be created
		log.V(1).Info("creating", "Deployment", name.String())
		if err = c.Create(ctx, desiredDeployment); err != nil {
			return nil, fmt.Errorf("creating Deployment: %w", err)
		}
		// no need to check for updates, object was created just now
		return desiredDeployment, nil
	}

	if !equality.Semantic.DeepEqual(desiredDeployment.Spec, currentDeployment.Spec) {
		// desired and current Deployment .Spec are not equal -> trigger an update
		log.V(1).Info("updating", "Deployment", name.String())
		currentDeployment.Spec = desiredDeployment.Spec
		if err = c.Update(ctx, currentDeployment); err != nil {
			return nil, fmt.Errorf("updating Deployment: %w", err)
		}
	}

	return currentDeployment, nil
}
