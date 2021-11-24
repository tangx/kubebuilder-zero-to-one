# 增加 event 事件支持

k8s 官方 controller 都实现了 Events 消息信息， 如下

```bash
kubectl describe deployment k8s-operator-demo-controller-manager

Events:
  Type    Reason             Age   From                   Message
  ----    ------             ----  ----                   -------
  Normal  ScalingReplicaSet  15m   deployment-controller  Scaled up replica set k8s-operator-demo-controller-manager-75cc59d8ff to 1
  Normal  ScalingReplicaSet  14m   deployment-controller  Scaled down replica set k8s-operator-demo-controller-manager-b9d9f7886 to 0
```

我们自定义的 Operator 同样可以实现。


## operator 支持 event

1. 在 `/controllers/redis_controller.go` 中定义 `RedisReconcile` 的时候， 添加 `EventRecord` 字段。

```go
// RedisReconciler reconciles a Redis object
type RedisReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	// 添加事件
	EventRecord record.EventRecorder
}
```


2. 在 `/main.go` 中， 创建 mgr 的时候使用 `mgr.GetEventRecorderFor("RedisOperator")` 初始化 `EventRecorder`。

其中 `RedisOperator` 的值可以任意定义， 在 Event 日志中为 `FROM` 字段的值。 通常使用控制器名字 **OperatorName**， 例如这里使用 `RedisOperator`。

```go
	if err = (&controllers.RedisReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),

		// 添加事件记录名称
		EventRecord: mgr.GetEventRecorderFor("RedisOperator"),

	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Redis")
		os.Exit(1)
	}

```

3. 在需要的地方添加事件日志输出

使用 `r.EventRecord.Event()` 方法记录事件日志

```go
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
```

其中， Event 需要一下几个字段

```go
Event(object runtime.Object, eventtype, reason, message string)
```

1. `object`: 记录日志的对象。 也就是 `kd redis <name>` 中的 name。
2. `eventtype`: **Type 字段**， 事件类型， 可以是任意值。 
    + k8s corev1 api 中提供了 2个 官方值: `Normal` 和 `Warning`

```go
const (
	// Information only and will not cause any problems
	EventTypeNormal string = "Normal"
	// These events are to warn that something might go wrong
	EventTypeWarning string = "Warning"
)
```

3. `reason`: **Reason 字段**， 事件原因， 可以理解为事件分类/类型。 
4. `message`: **Message 字段**， 事件消息， 详细描述。


## 测试

```bash
kd redis my-op-redis

# ...省略

Spec:
  Image:     redis:5-alpine
  Port:      8333
  Replicas:  1
Events:
Type     Reason  Age                From           Message
----     ------  ----               ----           -------
Normal   扩容      66s (x2 over 66s)  RedisOperator  redis 副本数设置为 3
Warning  缩容      49s                RedisOperator  redis 副本数设置为 1
Normal   扩容      38s (x5 over 72s)  RedisOperator  redis 副本数设置为 1
```


