/*
Copyright 2021.

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
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	myappv1 "github.com/tangx/k8s-operator-demo/api/v1"
	"github.com/tangx/k8s-operator-demo/controllers/helper2"
)

// RedisReconciler reconciles a Redis object
type RedisReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	// 添加事件
	EventRecord record.EventRecorder
}

//+kubebuilder:rbac:groups=myapp.tangx.in,resources=redis,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=myapp.tangx.in,resources=redis/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=myapp.tangx.in,resources=redis/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Redis object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *RedisReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	fmt.Println("进入 redis Reconcile, 检查调谐状态")
	defer fmt.Println("退出 redis Reconcile 调谐状态")

	_ = log.FromContext(ctx)

	// TODO(user): your logic here

	redis := &myappv1.Redis{}
	defer func() {
		// 状态赋值
		redis.Status.Replicas = len(redis.Finalizers)
		// 状态更新，偷懒忽略错误判断
		_ = r.Status().Update(ctx, redis)
	}()

	err := r.Get(ctx, req.NamespacedName, redis)
	if err != nil {
		// 如果 err !=nil , k8s 调谐会不断重试。 因此找不到资源， 则直接返回 err=nil
		// return ctrl.Result{}, fmt.Errorf("Reconcile 获取 redis 失败: %v", err)

		// 找不到返回 nil，成功处理， 退出循环。
		return ctrl.Result{}, nil
	}

	// 打印 redis 对象
	output(redis)

	// 删除 逻辑
	// IsZero 标识这个字段为 nil 或者 零值， 即非删除状态
	// 删除状态则 取反
	if !redis.DeletionTimestamp.IsZero() {
		return r.deleteReconcile(ctx, redis)
	}

	// 缩容
	if len(redis.Finalizers) > redis.Spec.Replicas {
		return r.decreaseReconcile(ctx, redis)
	}

	return r.increaseReconcile(ctx, redis)

}

// SetupWithManager sets up the controller with the Manager.
func (r *RedisReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&myappv1.Redis{}).
		// 监听 pod 事件
		Watches(
			&source.Kind{
				Type: &corev1.Pod{},
			},
			handler.Funcs{
				DeleteFunc: r.podDeleteHandler,
			},
		).
		Watches(
			&source.Kind{
				Type: &v1.Service{},
			},
			handler.Funcs{
				DeleteFunc: r.podDeleteHandler,
			},
		).
		Complete(r)
}

func (r *RedisReconciler) podDeleteHandler(e event.DeleteEvent, q workqueue.RateLimitingInterface) {

	ns := e.Object.GetNamespace()

	for _, owner := range e.Object.GetOwnerReferences() {
		// 非法 owner 不引起调谐
		if owner.APIVersion != "myapp.tangx.in/v1" || owner.Kind != "Redis" {
			continue
		}

		// 入队， 通知 redis operator 变更， 进行重新 调谐。
		q.Add(
			reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: ns,
					Name:      owner.Name,
				},
			},
		)
	}
}

func (r *RedisReconciler) increaseReconcile(ctx context.Context, redis *myappv1.Redis) (ctrl.Result, error) {

	// 添加事件日志
	r.EventRecord.Event(redis,
		corev1.EventTypeNormal, "扩容",
		fmt.Sprintf("%s 副本数设置为 %d", redis.Name, redis.Spec.Replicas),
	)

	// 创建 逻辑
	err := helper2.CreateRedisPod2(ctx, r.Client, redis, r.Scheme)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("创建 redis pod 失败: %v", err)
	}

	return ctrl.Result{}, nil
}

func (r *RedisReconciler) decreaseReconcile(ctx context.Context, redis *myappv1.Redis) (ctrl.Result, error) {

	r.EventRecord.Event(redis,
		corev1.EventTypeWarning, "缩容",
		fmt.Sprintf("%s 副本数设置为 %d", redis.Name, redis.Spec.Replicas),
	)

	err := helper2.DecreaseRedis2(ctx, r.Client, redis)

	return ctrl.Result{}, err
}

func (r *RedisReconciler) deleteReconcile(ctx context.Context, redis *myappv1.Redis) (ctrl.Result, error) {

	r.EventRecord.Event(redis,
		"Deleting", "删除",
		fmt.Sprintf("删除 %s", redis.Name),
	)

	err := helper2.DeleteRedis2(ctx, r.Client, redis)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("删除 redis 失败:%v", err)
	}

	return ctrl.Result{}, nil
}

func output(v interface{}) {
	data, err := json.Marshal(v)

	if err != nil {
		return
	}

	fmt.Printf("得到crd redis 对象: %s \n\n", data)
}
