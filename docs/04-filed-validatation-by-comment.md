# 使用注解完整字段值约束

在 `/api/v1/redis_types.go` 中，使用注解完成字段值约束。

1. 约束条件必须以 `//+kubebuilder:validation:<METHOD>:=<VALUE>` 为格式， 符号之间 **没有空格**。
2. 约束条件必须 **紧邻** 字段， 且在字段上方。

> https://book.kubebuilder.io/reference/markers/crd-validation.html

```go
type RedisSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Replicas int `json:"replicas,omitempty"`

	//+kubebuilder:validation:Minimum:=1234
	//+kubebuilder:validation:Maximum:=54321
	Port int32 `json:"port,omitempty"`
}
```

重新编译安装

```bash
make install
```

使用命令查看查看

```yaml
kg crd redis.myapp.tangx.in


    Spec:
    Description:  RedisSpec defines the desired state of Redis
    Properties:
        Port:
        Format:   int32
        Maximum:  54321
        Minimum:  1234
        Type:     integer
        Replicas:
        Type:  integer
    Type:      object
```

将 `/deploy/my-op-redis.yml` 的端口改为 `333` 测试

```yaml
apiVersion: myapp.tangx.in/v1
kind: Redis

metadata:
  name: my-op-redis

spec:
  replicas: 1
  port: 333

```

部署， 并得到报错信息。

```
ka -f deploy

    The Redis "my-op-redis" is invalid: spec.port: Invalid value: 333: spec.port in body should be greater than or equal to 1234
```