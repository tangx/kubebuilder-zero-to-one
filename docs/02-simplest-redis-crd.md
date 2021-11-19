# 简单跑一跑

## 定义 CRD Redis 对象字段

在 `/api/v1/redis_types.go` 中， 增加 Replicas 和 Port 字段。

```go
type RedisSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of Redis. Edit redis_types.go to remove/update
	// Foo string `json:"foo,omitempty"`
	Replicas int   `json:"replicas,omitempty"`
	Port     int32 `json:"port,omitempty"`
}
```

这个 RedisSpec 对应 `/deploy/my-op-redis.yml` 中的 spec 

```yaml

apiVersion: myapp.tangx.in/v1
kind: Redis

metadata:
  name: my-op-redis

spec:
  replicas: 1
  port: 3333

```

## 编码 Reconcile 调谐逻辑

在 `/controllers/redis_controller.go` 中编码 Reconcile(调谐) 逻辑。

```go
func (r *RedisReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here

	redis := v1.Redis{}
	err := r.Get(ctx, req.NamespacedName, &redis)
	if err != nil {
		return ctrl.Result{}, err
	}

	fmt.Println("得到crd redis 对象: ", redis)

	return ctrl.Result{}, nil
}
```

## 启动调试

```bash
make uninstall
make install
make run
```


新开窗口

```bash
ka -f deploy/

```

在 `make run` 窗口可以看到 调谐 中的输出结果。

```
得到crd redis 对象:  {{Redis myapp.tangx.in/v1} {my-op-redis  default  c0e85341-edf3-4261-92da-6a337d473f0c 775203 1 2021-11-19 15:26:53 +0800 CST <nil> <nil> map[] map[kubectl.kubernetes.io/last-applied-configuration:{"apiVersion":"myapp.tangx.in/v1","kind":"Redis","metadata":{"annotations":{},"name":"my-op-redis","namespace":"default"},"spec":{"port":3333,"replicas":1}}
] [] []  [{kubectl-client-side-apply Update myapp.tangx.in/v1 2021-11-19 15:26:53 +0800 CST FieldsV1 {"f:metadata":{"f:annotations":{".":{},"f:kubectl.kubernetes.io/last-applied-configuration":{}}},"f:spec":{".":{},"f:port":{},"f:replicas":{}}} }]} {1 3333} {}}
```