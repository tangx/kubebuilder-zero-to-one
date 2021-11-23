# Pod 扩容与缩容

代码分支越来越多 **增/删/改** 都有了， 于是选择拆分为 3 个分支。

```go
// 扩容
func (r *RedisReconciler) increaseReconcile(ctx context.Context, redis *myappv1.Redis) (ctrl.Result, error) {
// ...
}

// 缩容
func (r *RedisReconciler) decreaseReconcile(ctx context.Context, redis *myappv1.Redis) (ctrl.Result, error) {
// ...
}

// 删除
func (r *RedisReconciler) deleteReconcile(ctx context.Context, redis *myappv1.Redis) (ctrl.Result, error) {
// ...
}
```

所谓 **扩容/缩容**， 在通过 finalizers 管理的时候就是 `redis.spec.replicas` 与 `len(redis.finalizers)` 的大小比较。

```go
// 缩容
	if len(redis.Finalizers) > redis.Spec.Replicas {
		return r.decreaseReconcile(ctx, &redis)
	}
```

在实现过程中，保证行为幂等就行。

缩容代码如下

```go
func DecreaseRedis2(ctx context.Context, client client.Client, redis *appv1.Redis) error {
	isUpdated := false
	for _, name := range redis.Finalizers[redis.Spec.Replicas:] {
		pod := getPod2(redis, name)

		if err := client.Delete(ctx, pod); err != nil {
			return err
		}

		deleteFromFinalizers(redis, name)
		isUpdated = true
	}

	if isUpdated {
		return client.Update(ctx, redis)
	}

	return nil
}
```

缩容的实现就是从 `redis.finalizers` 中取出被管理的 pod 名字并删除。 由于我们创建 pod 的时候

1. 使用了 id 编号作为名字的 suffix (ex. `redis-0 / redis-1`)
2. 并且是按照顺序的 push 到 `redis.finalizers` 中的。

因此这里的缩容行为逻辑就更简单的一点， 最后后面的几个 POD 。

```go
	// 删除最后面的几个 Pod
	for _, name := range redis.Finalizers[redis.Spec.Replicas:] {
		pod := getPod2(redis, name)

		if err := client.Delete(ctx, pod); err != nil {
			return err
		}

		deleteFromFinalizers(redis, name)
	}
```
