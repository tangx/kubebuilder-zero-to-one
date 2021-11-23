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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	myappv1 "github.com/tangx/k8s-operator-demo/api/v1"
	"github.com/tangx/k8s-operator-demo/controllers/helper"
)

// RedisReconciler reconciles a Redis object
type RedisReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=myapp.tangx.in,resources=redis,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=myapp.tangx.in,resources=redis/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=myapp.tangx.in,resources=redis/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete

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

	redis := myappv1.Redis{}
	err := r.Get(ctx, req.NamespacedName, &redis)
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
		r.deleteReconcile(ctx, &redis)
	}

	// 缩容
	if len(redis.Finalizers) > redis.Spec.Replicas {
		return r.decreaseReconcile(ctx, &redis)
	}

	return r.increaseReconcile(ctx, &redis)

}

// SetupWithManager sets up the controller with the Manager.
func (r *RedisReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&myappv1.Redis{}).
		Complete(r)
}

func (r *RedisReconciler) increaseReconcile(ctx context.Context, redis *myappv1.Redis) (ctrl.Result, error) {

	// 创建 逻辑
	err := helper.CreateRedisPod2(ctx, r.Client, redis)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("创建 redis pod 失败: %v", err)
	}

	return ctrl.Result{}, nil
}

func (r *RedisReconciler) decreaseReconcile(ctx context.Context, redis *myappv1.Redis) (ctrl.Result, error) {

	err := helper.DecreaseRedis2(ctx, r.Client, redis)

	return ctrl.Result{}, err
}

func (r *RedisReconciler) deleteReconcile(ctx context.Context, redis *myappv1.Redis) (ctrl.Result, error) {

	err := helper.DeleteRedis2(ctx, r.Client, redis)
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
