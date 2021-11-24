# 添加 Status 状态字段

## 添加 kd 状态字段

在  `/api/v1/redis_types.go` 的 `RedisStatus` 中添加需要展示的字段。

这里添加一个副本数量。

```go
type RedisStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Replicas int `json:"replicas"`
}
```

偷懒， 没有在创建或删除 pod 时进行精细控制。 而是使用 `defer` 在 `Reconcile` 退出的时候进行一次最终的赋值管理。

```go
func (r *RedisReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	fmt.Println("进入 redis Reconcile, 检查调谐状态")
	defer fmt.Println("退出 redis Reconcile 调谐状态")

	_ = log.FromContext(ctx)

	redis := &myappv1.Redis{}

	defer func() {
		// 状态赋值
		redis.Status.Replicas = len(redis.Finalizers)
		// 状态更新，偷懒忽略错误判断
		_ = r.Status().Update(ctx, redis)
	}()

// ...省略
}
```

> 注意: 必须要使用 `r.Status().Update()` 方法更新， 否则不会展示 `Status` 字段。

编译发布， 并测试

```yaml
# kg redis my-op-redis -o yaml

apiVersion: myapp.tangx.in/v1
kind: Redis
metadata:
  finalizers:
  - my-op-redis-0
  - my-op-redis-1
  generation: 1
  name: my-op-redis
  namespace: k8s-operator-demo-system
spec:
  image: redis:5-alpine
  port: 8333
  replicas: 2
status:
  replicas: 2
```


## 添加 kg 打印字段


> https://book.kubebuilder.io/reference/generating-crd.html#additional-printer-columns


在 `/api/v1/redis_types.go` 中， 使用 `+kubebuilder:printcolumn` 添加需要展示的字段。

展示字段有三个属性:

1. `name`: 名字， 对外展示的名字，可以与实际属性名不一致。 
2. `type`: 类型， 字段的属性， 例如 `string, bool, integer` 等。
3. `JSONPath`: 属性路径。 可以通过 `kg redis <name> -o json` 查看属性实际路径。

> 注意: 展示属性可以是资源对象中的任意属性， 不一定非的是 Status

下述代码中， 分别展示了 `.status.replicas`, `.spec.image`, `.metadata.uid` 以及一个 **不存在的 `.spec.alias` 

```go
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Replicas",type=integer,JSONPath=`.status.replicas`
//+kubebuilder:printcolumn:name="ImageName",type=string,JSONPath=`.spec.image`
//+kubebuilder:printcolumn:name="Uuid",type=string,JSONPath=`.metadata.uid`
//+kubebuilder:printcolumn:name="Alias",type=string,JSONPath=`.spec.alias`
```


编译发布，并测试

```bash
kg redis

NAME          REPLICAS   IMAGENAME        ALIAS
my-op-redis   2          redis:5-alpine
```

> 注意: 展示字段的注解代码 `//+kubebuilder:priintcolumn` **必须`贴合`** `//+kubebuilder:subresource:status`。中间不能有空格， **上下都行**， 还是建写在 **status 的下方**。


### 非发布状态确认

在不发布的情况下， 使用 `make install` 生成最新部署配置文件

在 `/config/crd/bases/myapp.tangx.in_redis.yaml` 可以查看

```yaml
spec:
  versions:
  - additionalPrinterColumns:
    - jsonPath: .status.replicas
      name: Replicas
      type: integer
    - jsonPath: .spec.image
      name: ImageName
      type: string
    - jsonPath: .spec.alias
      name: Alias
      type: string
    # ...
```
