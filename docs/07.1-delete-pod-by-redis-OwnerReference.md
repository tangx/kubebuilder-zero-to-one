# 使用 OwnerReference 管理 redis operator 创建的 Pod 

> https://kubernetes.io/blog/2021/05/14/using-finalizers-to-control-deletion/


在上一章的代码可以通过如下命令创建一个 redis 实例， 并随即创建一个 Pod

```bash
ka -f deploy/
```

但是在使用如下命令删除 redis 实例时， 虽然命令行界面提示删除成功， 但是创建的 Pod 依旧存在。

```bash
krm -f deploy/
```

其原因是 **redis 实例** 与 **Pod** 之间 **没有** 建立关联关系。 

那要如何创建关联关系呢？ 可以参考阅读官方博客， 使用 `finalizer` 控制删除。

在 [Owner Reference](https://kubernetes.io/blog/2021/05/14/using-finalizers-to-control-deletion/#owner-references) 一节中提到了资源的 **父子关系** 。

根据这个原理， 更新 `/controllers/helper/redis_helper.go` 的 Pod 创建 API， 加入 OwnerReference 相关的代码。

```go
// CreateRedis 创建 redis pod
func CreateRedisPod(ctx context.Context, client client.Client, config *appv1.Redis) error {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%d", config.Name, time.Now().Unix()),
			Namespace: config.Namespace,
			// 创建 OwnerReference
			OwnerReferences: []metav1.OwnerReference{
				ownerReference(*config),
			},
		},
		// 省略
	}
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


## 编译测试

重新编译，并部署安装

```bash
make docker-build
make install
make deploy
```


```bash
#### 创建
# ka -f deploy
    redis.myapp.tangx.in/my-op-redis created
# kgp
NAME                           READY   STATUS        RESTARTS   AGE
my-op-redis-1637576923         0/1     Running       0          1s


#### 删除
# krm -f deploy/
redis.myapp.tangx.in "my-op-redis" deleted

# kgp
NAME                           READY   STATUS        RESTARTS   AGE
my-op-redis-1637576923         1/1     Terminating   0          10s
```

