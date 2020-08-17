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

//Controllers
//
//Controllers (pkg/controller) use events (pkg/events) to eventually trigger
//reconcile requests.  They may be constructed manually, but are often
//constructed with a Builder (pkg/builder), which eases the wiring of event
//sources (pkg/source), like Kubernetes API object changes, to event handlers
//(pkg/handler), like "enqueue a reconcile request for the object owner".
//Predicates (pkg/predicate) can be used to filter which events actually
//trigger reconciles.  There are pre-written utilities for the common cases, and
//interfaces and helpers for advanced cases.

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

//Reconcilers
//
//
//
//Controller logic is implemented in terms of Reconcilers (pkg/reconcile).
//A Reconciler implements a function which takes a reconcile Request containing
//the name and namespace of the object to reconcile, reconciles the object,
//and returns a Response or an error indicating whether to requeue for a
//second round of processing.

// AtReconciler reconciles a At object
type AtReconciler struct {
	//Clients and Caches
	//
	//Reconcilers use Clients (pkg/client) to access API objects.  The default
	//client provided by the manager reads from a local shared cache (pkg/cache)
	//and writes directly to the API server, but clients can be constructed that
	//only talk to the API server, without a cache.  The Cache will auto-populate
	//with watched objects, as well as when other structured objects are
	//requested. The default split client does not promise to invalidate the cache
	//during writes (nor does it promise sequential create/get coherence), and code
	//should not assume a get immediately following a create/update will return
	//the updated resource. Caches may also have indexes, which can be created via
	//a FieldIndexer (pkg/client) obtained from the manager.  Indexes can used to
	//quickly and easily look up all objects with certain fields set.  Reconcilers
	//may retrieve event recorders (pkg/recorder) to emit events using the
	//manager.
	client.Client
	Log logr.Logger
	//	Schemes
	//
	//	Clients, Caches, and many other things in Kubernetes use Schemes
	//  (pkg/scheme) to associate Go types to Kubernetes API Kinds
	//  (Group-Version-Kinds, to be specific).
	Scheme *runtime.Scheme
	// EventRecorder knows how to record events on behalf of an EventSource.
	Recorder record.EventRecorder
}

// 定义调谐器所需要的权限 包含 ats 以及 deployment 等的权限
// 这些权限的注释会自动生成rbac的yaml文件
// 更多参考 https://book.kubebuilder.io/reference/markers.html
// +kubebuilder:rbac:groups=cnat.my.domain,resources=ats,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cnat.my.domain,resources=ats/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch

// Reconcilers 的Reconcile方法

// Reconciler performs a full reconciliation for the object referred to by the Request.
// The Controller will requeue the Request to be processed again if an error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.

// ctrl.Request
// Request contains the information necessary to reconcile a Kubernetes object.  This includes the
// information to uniquely identify the object - its Name and Namespace.

// ctrl.Result == reconcile.Result
// Result contains the result of a Reconciler invocation.
//type Result struct {
//	// Requeue tells the Controller to requeue the reconcile key.  Defaults to false.
//	Requeue bool
//
//	// RequeueAfter if greater than 0, tells the Controller to requeue the reconcile key after the Duration.
//	// Implies that Requeue is true, there is no need to set Requeue to true at the same time as RequeueAfter.
//	RequeueAfter time.Duration
//}
func (r *AtReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	logger := r.Log.WithValues("namespace", req.NamespacedName, "at", req.Name)
	logger.Info("== Reconciling At")

	// 实例化一个空的资源对象，指针类型
	instance := &cnatv1alpha1.At{}

	// 通过 AtReconciler 的get方法通过k8s client 来获取相关的资源对象
	// 在给定的namespace获取 cnatv1alpha1.At 对象
	// 因为r是指针，获取完成就放到缓存了，不用期待返回值

	// Get retrieves an obj for the given object key from the Kubernetes Cluster.
	// obj must be a struct pointer so that obj can be updated with the response
	// returned by the Server.

	// 执行完成后，从apiserver获取cnatv1alpha1.At{} 就会填充相关的字段 因为他是指针类型
	err := r.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, nil
	}

	// 因为是指针类型，因此可以进行相关的字段填充
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
		// Package controllerutil contains utility functions for working with and implementing Controllers.

		// SetControllerReference sets owner as a Controller OwnerReference on controlled.
		// This is used for garbage collection of the controlled object and for
		// reconciling the owner object on changes to controlled (with a Watch + EnqueueRequestForOwner).
		// Since only one OwnerReference can be a controller, it returns an error if
		// there is another OwnerReference with Controller flag set.
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

// 向manager注册
// 传入实例化后的mgr
func (r *AtReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// NewControllerManagedBy 方法接收mgr参数 返回一个 Builder
	// 构造器提供创建controller的接口

	// 下面通过一些 builder 的方法来进行controller的配置 //

	// builder provides wraps other controller-runtime libraries and exposes simple
	// patterns for building common Controllers.

	// ControllerManagedBy returns a new controller builder that will be started by the provided Manager
	return ctrl.NewControllerManagedBy(mgr).

		//
		//
		// For defines
		//1 the type of Object being *reconciled*,
		//2 configures the ControllerManagedBy to respond to create / delete / update events by *reconciling the object*.
		// 参数接收一个资源对象结构体 返回 *builder
		// 控制器调协的对象是哪个
		For(&cnatv1alpha1.At{}).

		// Owns defines
		//1 types of Objects being *generated* by the ControllerManagedBy,
		//2 configures the ControllerManagedBy to respond to create / delete / update events by *reconciling the owner object*.
		// 控制器管理的对象是哪个 接收事件的对象是哪个
		Owns(&cnatv1alpha1.At{}).
		// pod也被该控制器管理以及接收事件
		// 控制器管理的对象是哪个 接收事件的对象是哪个
		Owns(&corev1.Pod{}).

		// Complete builds the Application ControllerManagedBy.
		// 将 Reconcilers 注册到controller里面
		Complete(r)
}
