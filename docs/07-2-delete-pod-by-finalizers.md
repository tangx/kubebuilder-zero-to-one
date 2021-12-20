# 使用 finalizers 管理 redis operator 创建的 Pod

> https://kubernetes.io/blog/2021/05/14/using-finalizers-to-control-deletion/

上一章使用了 OwnerReference 关联 redis instance 和所创建的 Pod， 这里的删除是通过 k8s 内置的关系处理器处理的。

根据官方博客文档中的阐述， 当一个资源的额 finalizers 没有被清空时， 这个资源将无法被删除。 因此， 本章通过 finalizers 

1. 来建立 redis instance 和所创建 pod 的关系，
2. 以及处理删除逻辑

## 1. 创建 redis instance 与 pod 的关系

在 `/controllers/helper/redis_helper.go` 通过 Finalizers 管理 Pod

```go
// CreateRedis 创建 redis pod
func CreateRedisPod2(ctx context.Context, client client.Client, redis *appv1.Redis) error {

	isUpdated := false
	for i := 0; i < redis.Spec.Replicas; i++ {
		name := fmt.Sprintf("%s-%d", redis.Name, i)
		fmt.Println("创建 pod lo :", name)
		pod := getPod2(redis, name)

		if isPodExist2(redis, pod.Name) {
			continue
		}

		if err := client.Create(ctx, pod); err != nil {
			return err
		}

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
```

1. 所有通过 redis instance 创建新创建的 Pod 使用 `pod.Name` 为 key ， 保存在 `redis.Finalizers` 中进行管理。

```go
		redis.Finalizers = append(redis.Finalizers, pod.Name)
```

2. 为了保证只有出现 pod 变更的时候才进行 redis 的 update 操作（幂等）， 使用了 `isUpdated` 作为信号条件。

```go
	// redis.Finalizers 的变更是在本地内存中， 使用 update 更新到 k8s 中
	if isUpdated {
		return client.Update(ctx, redis)
	}
```

**注意**: `append()` 操作虽然将 `pod.Name` 加入到了 `redis.Finalizers` 中， 单这是在本地内存实现的。 因此必须要使用 `client.Update()` 操作将变更保存到 k8s 中


## 2. 删除 redis instance 与 pod

根据博客中指出: 

1. 在触发删除的时候， redis instance 会多一个 `DeletionTimestamp` 的标识， 具有该标识的实例 **1. 处于制度状态**， 但 **2. 能管理操作其 finalizers 字段** 
2. 在 `redis.Finalizers` 不为空的时候， redis instance 是处于 **删除状态** 被夯住的。

因此如果要进行删除逻辑， 则需要先进行 **标识判断**

在 `/controllers/redis_controller.go` 中

```go
func (r *RedisReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

// 省略
	// 删除 逻辑
	// IsZero 标识这个字段为 nil 或者 零值， 即非删除状态
	// 删除状态则 取反
	if !redis.DeletionTimestamp.IsZero() {

		err = helper.DeleteRedis2(ctx, r.Client, &redis)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("删除 redis 失败:%v", err)
		}

		return ctrl.Result{}, nil
	}
// 省略
}
```

删除逻辑代码如下

```go
func DeleteRedis2(ctx context.Context, client client.Client, redis *appv1.Redis) error {

	fmt.Println("进入删除循环咯")

	isUpdated := false
	for _, name := range redis.Finalizers {
		pod := getPod2(redis, name)

		if err := client.Delete(ctx, pod); err != nil {
			return fmt.Errorf("删除 pod (%s) 失败: %v\n", name, err)
		}

		deleteFromFinalizers(redis, pod.Name)
		isUpdated = true
	}

	if isUpdated {
		if err := client.Update(ctx, redis); err != nil {
			return fmt.Errorf("更新 redis 失败: %v\n", err)
		}

		return client.Delete(ctx, redis)
	}
	return nil
}
```

1. 进入到删除逻辑后, 遍历 `redis.Finalizers` 获取所有被管理的 `pod.Name`， 依次删除
2. 删除成功后， 将 `pod.Name` 从 `redis.Finalizers` 中删除
3. 当本地 `redis.Finalizers` 被清空后， 将状态更新到 k8s 中。 至此 k8s 回收功能就可以清理 redis instance 了。


## 3. 退出调谐 Reconcile

当 Reconcile 错误退出的时候( `err!=nil` ), k8s 认为资源状态没有达到预期， 调谐会不断的进行重试。 即 `Reconcile` 会不断的执行。

```go
func (r *RedisReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	fmt.Println("进入 redis Reconcile, 检查调谐状态")
	defer fmt.Println("退出 redis Reconcile 调谐状态")

	_ = log.FromContext(ctx)

	// TODO(user): your logic here

	redis := v1.Redis{}
	err := r.Get(ctx, req.NamespacedName, &redis)
	if err != nil {
		// 如果 err !=nil , k8s 调谐会不断重试。 因此找不到资源， 则直接返回 err=nil
		// return ctrl.Result{}, fmt.Errorf("Reconcile 获取 redis 失败: %v", err)

		// 找不到返回 nil，成功处理， 退出循环。
		return ctrl.Result{}, nil
	}
}
```

因此如果资源找不到则直接返回 `err==nil`, 表示资源状态达到 **预期**， 停止调谐。
