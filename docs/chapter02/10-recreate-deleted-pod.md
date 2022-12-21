# 重建被删除的 Pod

之前遗留了一个问题， 直接用命令行删除的 Pod 不能被重建。 这次就来解决它。

首先来整理之前遗留的问题故障点在哪里？

1. 使用命令 `kubectl delete` 直接删除 pod 的时候， `redis.Finalizers` **不会变更**， 依旧包含被删除的 `pod.Name`。
2. 在创建 Pod 的时候， 判断 Pod 是否存在使用的是 `redis.Finalizers` 提供信息， 而 **没有判断** k8s 中真实的情况。 
3. **没有机制** 通知 `redis operator` 进行检测或重建。


因此全新流程如下

1. Pod 状态变化: `kubectl delete` 删除 Pod 
2. Redis 重新调谐: 通知 Redis operator 变化， 重新启动 **调谐(Reconcile)**
3. 创建 Pod 的逻辑如下
    1. 如果 Pod 在 k8s 中存在， 则跳过。 （为了降低复杂性， 不考虑直接改变 redis.finalizers 的情况）
    2. 如果 Pod 不存在， 创建 Pod。 是否更新 `redis.Finaliers`， 取决于 Pod 是 **新建** 或者 **重建**。
        1. **新建** 如果 `pod.Name` 不存在， 则 append 到末尾。 这点保持不变。
        2. **重建** 如果 `pod.Name` 存在其中， 则跳过。


## 1. 通知 Redis Operator 变化

上一章中已经提到了， redis operator 可以订阅 k8s 事件， `/controllers/redis_controller.go` 代码如下

代码中订阅了 Pod 事件。 当 Pod 发生 **删除事件** 后， 回调  `r.podDeleteHandler` 进行处理。

```go
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
		)
		Complete(r)
}
```

在回调函数 `r.podDeleteHanlder` 中， 就需要实现通知的前置条件， **找到需要通知的 redis operator 对象**。 
至于如何通知， kubebuilder 已经给我们封装好了， 不用过多考虑。


要实现 redis operator 的通知， 其 **关键信息** 在于 `OwnerReferences`。

在 [官方博文 - 使用 finalizers 控制删除行为](https://kubernetes.io/blog/2021/05/14/using-finalizers-to-control-deletion/) 提到过 **父子资源** 之间可以通过 `OwnerReference` 进行关联形成关系树， 有利于资源的控制、跟踪、管理和回收。

在这里可以简单的认为， redis operator 在创建 **子 pod** 的时候， 使用 OwnerReference 注入了一些自身相关的有效性信息。

这部分的代码实现， 可以查看 [07.1 使用 OwnerReference 关系删除 Pod 对象](07.1-delete-pod-by-redis-OwnerReference.md)


```go

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
```

上述 handler 代码中，在删除事件 e 发生时

1. 通过 `e.Object.GetNamespace()` 获取到被删除的 Pod 所在的 namespace。
    + redis operator 和 pod 是在同一个 namespace 下。
2. 通过 `e.Object.GetOwnerReferences()` 获取到与 Pod 所有相关的 **父资源** 对象。
3. 循环便利所有 Owners, 获得 owner 资源名称
    1. 将 owner 的 namespace 和 name 包装一下， 成为 `reconcile.Request` 对象
    2. 将新包装的对象加入到 `q workqueue.RateLimitingInterface` 队列中。
4. 之后一切交给 k8s 完成。


不论处于性能还是安全考虑， 都应该增加如下代码。 非本 Operator 创建的 Pod 资源的生命周期行为应该被忽略。

```go
// 非法 owner 不引起调谐
if owner.APIVersion != "myapp.tangx.in/v1" || owner.Kind != "Redis" {
    continue
}
```

否则任何 Pod 的删除都将引起 redis operator 的 Reconcile 行为。


## 2. Pod 创建流程的变化

### 2.1 OwnerReference 支持


上一节已经提到了， 实现通知的前提是依赖 OwnerReference。

在 `/controllers/helper/redis_helper.go` 创建 pod 对象明细的相关代码中加入相关代码。

```go

func getPod2(redis *appv1.Redis, name string) *corev1.Pod {

	pod := &corev1.Pod{}
	pod.Name = name
	pod.Namespace = redis.Namespace

	// 创建 pod 时添加 OwnerReference
	pod.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
		ownerReference(*redis),
	}

// .. 省略
}

func ownerReference(config appv1.Redis) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion:         config.APIVersion,
		Kind:               config.Kind,
		Name:               config.Name,
		UID:                config.UID,
		Controller:         ptrBool(true),
		BlockOwnerDeletion: ptrBool(true),
	}
}
```


### 2.2 Pod 创建流程行为变更

```go

// CreateRedis 创建 redis pod
func CreateRedisPod2(ctx context.Context, client client.Client, redis *appv1.Redis) error {

	isUpdated := false
	for i := 0; i < redis.Spec.Replicas; i++ {
		name := fmt.Sprintf("%s-%d", redis.Name, i)
		fmt.Println("创建 pod lo :", name)

		// 如果在 k8s 中存在则跳过。 暂不考虑有人直接修改 redis 的 finalizers 的情况
		if isPodExistInK8S(ctx, client, redis.Namespace, name) {
			continue
		}

		pod := getPod2(redis, name)
		if err := client.Create(ctx, pod); err != nil {
			return err
		}

		// 如果 pod.Name 在 finaliers 中， 则为删后重建。
		if isPodExistInFinalizers2(redis, pod.Name) {
			continue
		}

		// 如果 pod.Name 不在 finalizers 中， 则为新增 pod。
		// 使用 Finalizer 管理创建的 Pod。 当 pod 被删除完的时候，才能删除 redis
		redis.Finalizers = append(redis.Finalizers, pod.Name)
		isUpdated = true
	}

	// redis.Finalizers 的变更是在本地内存中， 使用 update 更新到 k8s 中
	if isUpdated {
		return client.Update(ctx, redis)
	}
	return nil
}

// isPodExistInK8S 检测 pod 是否在 k8s 中存在
//   true 为存在
func isPodExistInK8S(ctx context.Context, client client.Client, namespace string, name string) bool {

	key := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}

	// 这里偷懒， 没有进行错误内容检测。
	err := client.Get(ctx, key, &corev1.Pod{})
	return err == nil
}
```

之前提到了， 由于 `kubectl delete` 删除的原因， 导致了 `redis.Finalizers` 的数据失真。 

因此在创建 Pod 时， 

1. 首选需要通过 `isPodExistInK8S` 检查 Pod 是否存在于 k8s 中， 如果 Pod 已存在则忽略； 不存在则继续创建。
2. 使用 `client.Create` 创建 Pod。
3. 使用检查 `pod.Name` 是否存在于 `redis.Finialers` 中。 如果存在则表示 Pod 属于 **删后重建**， 不更新 `redis.Finialers` ； 如果不存在则表示为 **新建**， 需要更新 `redis.Finialers`

