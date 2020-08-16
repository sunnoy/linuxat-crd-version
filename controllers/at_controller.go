/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	// Core GoLang contexts
	"context"
	"fmt"
	"strings"
	"time"

	// 3rd party and SIG contexts
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"k8s.io/client-go/tools/record"

	// Local Operator contexts
	cnatv1alpha1 "example/api/v1alpha1"
)

// AtReconciler reconciles a At object
type AtReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	Recorder record.EventRecorder
}

// 定义调谐器所需要的权限 包含 ats 以及 deployment 等的权限
// 这些权限的注释会自动生成rbac的yaml文件
// 更多参考 https://book.kubebuilder.io/reference/markers.html
// +kubebuilder:rbac:groups=cnat.my.domain,resources=ats,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cnat.my.domain,resources=ats/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch

func (r *AtReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	logger := r.Log.WithValues("namespace", req.NamespacedName, "at", req.Name)
	logger.Info("== Reconciling At")

	// 实例化一个空的资源对象
	instance := &cnatv1alpha1.At{}

	// 通过 AtReconciler 的get方法通过k8s client 来获取相关的资源对象
	// 在给定的namespace获取 cnatv1alpha1.At 对象
	err := r.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, nil
	}

	if instance.Status.Phase == "" {
		instance.Status.Phase = cnatv1alpha1.PhasePending
	}

	switch instance.Status.Phase {

	case cnatv1alpha1.PhasePending:
		logger.Info("Phase: Pending")

		logger.Info("checking schedules", "target", instance.Spec.Schedule)

		d, err := timeUntilSchedule(instance.Spec.Schedule)

		if err != nil {
			logger.Error(err, "schedule paring failre")
			return ctrl.Result{}, nil
		}

		logger.Info("schedule paring successed", "result", fmt.Sprintf("diff=%v", d))
		if d > 0 {
			return ctrl.Result{RequeueAfter: d}, nil
		}

		logger.Info("it is time", "ready to run ", instance.Spec.Command)
		instance.Status.Phase = cnatv1alpha1.PhaseRunning

		r.Recorder.Event(instance, "Normal", "PhaseChange", cnatv1alpha1.PhasePending)
	case cnatv1alpha1.PhaseRunning:
		logger.Info("Phase: runing")

		pod := newPodForCR(instance)

		// 将pod资源归到at实例名下
		if err := controllerutil.SetControllerReference(instance, pod, r.Scheme); err != nil {
			return reconcile.Result{}, nil
		}

		found := &corev1.Pod{}
		err := r.Get(context.TODO(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, found)

		if err != nil && errors.IsNotFound(err) {
			err = r.Create(context.TODO(), pod)
			if err != nil {
				return reconcile.Result{}, err
			}
			logger.Info("pod start", "name", pod.Name)
		} else if err != nil {
			return reconcile.Result{}, err
		} else if found.Status.Phase == corev1.PodFailed || found.Status.Phase == corev1.PodSucceeded {
			logger.Info("container termiaing", "reason", found.Status.Reason, "message", found.Status.Message)
			instance.Status.Phase = cnatv1alpha1.PhaseDone
		} else {
			return reconcile.Result{}, nil
		}
		r.Recorder.Event(instance, "Running", "PhaseChange", cnatv1alpha1.PhaseRunning)
	case cnatv1alpha1.PhaseDone:

		logger.Info("Phase: done")

		r.Recorder.Event(instance, "Done", "PhaseChange", cnatv1alpha1.PhaseDone)
		return reconcile.Result{}, nil

	default:
		logger.Info("NOP")
		return reconcile.Result{}, nil

	}

	// Update the At instance, setting the status to the respective phase:
	err = r.Status().Update(context.TODO(), instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	return ctrl.Result{}, nil
}

// 创建集群的基础资源对象 pod
func newPodForCR(cr *cnatv1alpha1.At) *corev1.Pod {
	labels := map[string]string{
		"app": cr.Name,
	}

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-pod",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "busybox",
					Image:   "busybox",
					Command: strings.Split(cr.Spec.Command, " "),
				},
			},
			RestartPolicy: corev1.RestartPolicyOnFailure,
		},
	}
}

// timeUntilSchedule parses the schedule string and returns the time until the schedule.
// When it is overdue, the duration is negative.
func timeUntilSchedule(schedule string) (time.Duration, error) {
	now := time.Now().UTC()
	layout := "2006-01-02T15:04:05Z"
	// 根据时间格式解析出执行的时间
	s, err := time.Parse(layout, schedule)
	if err != nil {
		return time.Duration(0), err
	}
	return s.Sub(now), nil
}

func (r *AtReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cnatv1alpha1.At{}).
		Owns(&cnatv1alpha1.At{}).
		// 控制器看到pod的事件
		Owns(&corev1.Pod{}).
		Complete(r)
}

