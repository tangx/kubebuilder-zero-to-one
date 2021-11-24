# 使用 controllerutil 优化代码

在之前的代码中， 对于 OwnerReference 和 Finalizers 操作我们自己实现了一些方法。 其实这些操作官方已经封好成包了， 开箱即用。

复制 `/controllers/helper` 保存为 `/controllers/helper2`。 前者保存手工代码， 后者保存优化代码。

> https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/controller/controllerutil

## Finalizers 操作

之前

```go

// 添加
func appendFinalizers(redis *appv1.Redis, name string){
	// 如果 pod.Name 不在 finalizers 中， 则为新增 pod。
	// 使用 Finalizer 管理创建的 Pod。 当 pod 被删除完的时候，才能删除 redis
	redis.Finalizers = append(redis.Finalizers, name)
}


// 存在判断
func isPodExistInFinalizers2(redis *appv1.Redis, name string) bool {
	for _, rname := range redis.Finalizers {
		if rname == name {
			return true
		}
	}

	return false
}

// 删除
func deleteFromFinalizers(redis *appv1.Redis, name string) {
	for i, rname := range redis.Finalizers {
		if rname == name {
			redis.Finalizers = append(redis.Finalizers[:i], redis.Finalizers[i+1:]...)

			return
		}
	}
}
```

之后

```go
// 添加
controllerutil.AddFinalizer(redis, name)

// 存在判断
controllerutil.ContainsFinalizer(redis, pod.Name) 

// 删除
controllerutil.RemoveFinalizer(redis, pod.Name)
```


## OwnerReference 操作

之前

```go

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

func ptrBool(b bool) *bool {
	return &b
}
```

之后

```go
// 创建 pod 时添加 OwnerReference
controllerutil.SetOwnerReference(redis, pod, scheme)
```

其中 scheme 为 `RedisReconciler` 中的 Scheme 字段

```go
// RedisReconciler reconciles a Redis object
type RedisReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}
```
