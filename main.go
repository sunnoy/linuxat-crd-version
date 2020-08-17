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

package main

import (
	"flag"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	cnatv1alpha1 "example/api/v1alpha1"
	"example/controllers"
	// +kubebuilder:scaffold:imports
)

var (
	// 初始化一个 指针scheme ，返回 *Scheme
	scheme = runtime.NewScheme()
	// 初始化一个 logr.Logger对象
	setupLog = ctrl.Log.WithName("setup")
)

// 初始化函数先于main函数运行
func init() {

	//底层是runtime.SchemeBuilder结构体的 AddToScheme 方法，参数是个指针 *runtime.Scheme
	// 从clientset的角度添加scheme，相当与注册，为以后获取作准备
	_ = clientgoscheme.AddToScheme(scheme)

	// 底层是 Builder 结构体的 AddToScheme 方法，参数是 *runtime.Scheme 指针
	// 对自定义的api添加scheme
	_ = cnatv1alpha1.AddToScheme(scheme)

	// +kubebuilder:scaffold:scheme
}

func main() {

	// 解析参数
	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	// 初始化日志
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	// Managers

	// Every controller and webhook is ultimately run by a Manager (pkg/manager).

	// A manager is responsible for
	// 1 running controllers and webhooks, and
	// 2 setting up common dependencies (pkg/runtime/inject), like shared caches and clients,
	// 3 managing leader election (pkg/leaderelection).
	// 4 Managers are generally configured to gracefully shut down controllers on pod termination
	// by wiring up a signal handler (pkg/manager/signals).
	// 初始化 manager
	// 参数 1 *rest.Config 用来和apiserver联系 GetConfigOrDie() 返回一个指针*rest.Config对象
	// 参数 2 Options are the arguments for creating a new Manager
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		// todo 初始化manager的时候的选项
		// 需要使用初始化好的scheme对象
		Scheme: scheme,
		// 监控指标监听地址 通过命令行传入的参数来获取
		MetricsBindAddress: metricsAddr,
		// webhook server serves at
		Port: 9443,
		// 启动manager的时候是否启用选举 该参数通过命令行传入的参数来获取
		LeaderElection: enableLeaderElection,
		// LeaderElectionID determines the name of the configmap that leader election
		// will use for holding the leader lock.
		LeaderElectionID: "8bf23ea1.my.domain",
	})

	// 无法初始化manager是个比较严重的错误 因此需要退出程序
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Controllers
	//
	// Controllers (pkg/controller) use events (pkg/events) to eventually trigger
	// reconcile requests.
	// They may be constructed manually, but are often constructed with a Builder (pkg/builder),

	// builder provides wraps other controller-runtime libraries and exposes simple
	// patterns for building common Controllers.

	//which eases the wiring of event sources (pkg/source), like Kubernetes API object changes, to event handlers
	// (pkg/handler), like "enqueue a reconcile request for the object owner".

	// Predicates (pkg/predicate) can be used to filter which events actually
	// trigger reconciles.  There are pre-written utilities for the common cases, and
	// interfaces and helpers for advanced cases.

	//
	if err = (&controllers.AtReconciler{
		// 需要一个client对象
		Client: mgr.GetClient(),
		// 需要一个日志对象
		Log: ctrl.Log.WithName("controllers").WithName("At"),
		// 需要一个mgr提供的scheme对象
		Scheme: mgr.GetScheme(),
		// 事件记录注册
		Recorder: mgr.GetEventRecorderFor("at-controller"),

		// 使用 SetupWithManager 方法 ，传入初始化后的mgr 返回一个error对象
		// 将实例化的controller注册到manager，注册失败就退出程序
		// todo 通过这个方法引出控制器
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "At")
		os.Exit(1)
	}

	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")

	// 通过manager接口的start方法启动mgr
	// Start starts all registered Controllers and blocks until the Stop channel is closed.
	// 接收一个通道类型的参数
	// 当接受到 SIGTERM and SIGINT 参数的时候就进行退出
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
