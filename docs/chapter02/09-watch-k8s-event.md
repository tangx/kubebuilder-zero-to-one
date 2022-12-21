# 监听 k8s 事件

之前的代码遗留了一个问题， 当手动通过命令删除 pod 时候， 不会出发 `redis.Finalizers` 的更新， 也不会重建被删除的 Pod， 实现效果并不好

```bash
kubectl delete pod pod_name
```

## 1. 监听事件

在 `/controllers/redis_controller.go` 中生成了对象和方法监听 k8s 的事件。

ctrl 创建的 `Builder` 可以通过 **链式** 调用方式， 监听多个 k8s 对象的事件。

```go
// SetupWithManager sets up the controller with the Manager.
func (r *RedisReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&myappv1.Redis{}).
		// 监听 pod 事件
		Watches(
			&source.Kind{Type: &corev1.Pod{}},
			handler.Funcs{DeleteFunc: r.podDeleteHandler},
		).
		Watches(
			&source.Kind{Type: &v1.Service{},},
			handler.Funcs{DeleteFunc: r.podDeleteHandler},
		).
		Complete(r)
}
```

上述代码中， 使用 `Watches` 方法监听了 **Pod** 和 **Service** 的事件。 `Watch` 的两个参数

1. 监听事件对象( `source.Source`): 可以是 `source.Kind`、 `source.Channel` 或 `source.Informer`。
2. 监听事件行为( `handler.EventHandler` ): 当事件对象发生满足条件时所触发的行为。
    + Create: 对应 CreateFunc ， 满足 **创建** 条件是触发
    + Update: 对应 UpdateFunc ， 满足 **更新** 条件是触发
    + Delete: 对应 DeleteFunc ， 满足 **删除** 条件是触发
    + Generic: 对应 GenericFunc ， 满足 **未知** 条件是触发

## 2. 事件方法

每个事件方法都支持 2个 参数， **事件** 与 **工作队列** 。

```go
// Funcs implements EventHandler.
type Funcs struct {
	// Create is called in response to an add event.  Defaults to no-op.
	// RateLimitingInterface is used to enqueue reconcile.Requests.
	CreateFunc func(event.CreateEvent, workqueue.RateLimitingInterface)

	// Update is called in response to an update event.  Defaults to no-op.
	// RateLimitingInterface is used to enqueue reconcile.Requests.
	UpdateFunc func(event.UpdateEvent, workqueue.RateLimitingInterface)

	// Delete is called in response to a delete event.  Defaults to no-op.
	// RateLimitingInterface is used to enqueue reconcile.Requests.
	DeleteFunc func(event.DeleteEvent, workqueue.RateLimitingInterface)

	// GenericFunc is called in response to a generic event.  Defaults to no-op.
	// RateLimitingInterface is used to enqueue reconcile.Requests.
	GenericFunc func(event.GenericEvent, workqueue.RateLimitingInterface)
}
```

通过事件， 可以拿到对象应变化的对象。 有了这些对象， 就可以定制更多的行为。

```go
func (r *RedisReconciler) podDeleteHandler(e event.DeleteEvent, q workqueue.RateLimitingInterface) {

	pname := e.Object.GetName()
	pns := e.Object.GetNamespace()
	fmt.Printf("Pod %s in NS %s 被删除\n", pname, pns)

}
```

上述是拿到 Pod 删除事件后， 打印出相关日志信息。

```bash
krm -f deploy/

    Pod my-op-redis-1 in NS k8s-operator-demo-system 被删除
    Pod my-op-redis-0 in NS k8s-operator-demo-system 被删除
```

## 3. 监听的资源与权限

既然是监听 k8s 资源， 那就必须对所监听的资源有响应的权限。

以下的日志报错就是因为没有提前授权 Service 的权限造成的。

```log
E1123 08:10:53.183975       1 reflector.go:138] pkg/mod/k8s.io/client-go@v0.22.1/tools/cache/reflector.go:167: Failed to watch *v1.Service: failed to list *v1.Service: services is forbidden: User "system:serviceaccount:k8s-operator-demo-system:k8s-operator-demo-controller-manager" cannot list resource "services" in API group "" at the cluster scope
```

解决方法和之前提到的一样， 通过 kubebuilder 的注解方法生成对应的 rbac 权限。

```go
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
```