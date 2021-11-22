# 使用 Operator 创建并发布一个 Pod


## 1. 组装 k8s api 创建 pod

创建 `/controllers/helper` 目录， 这里面的代码实现 k8s Workloads 的创建。 具体实现就是封装 k8s workloads 的 api 对象

```go
// CreateRedis 创建 redis pod
func CreateRedisPod2(ctx context.Context, client client.Client, config *appv1.Redis) error {
	pod := &corev1.Pod{}
	pod.Name = config.Name
	pod.Namespace = config.Namespace
	pod.Spec.Containers = []corev1.Container{
		{
			Name:            config.Name,
			Image:           config.Spec.Image,
			ImagePullPolicy: corev1.PullIfNotPresent,
			Ports: []corev1.ContainerPort{
				{
					ContainerPort: config.Spec.Port,
				},
			},
		},
	}
	// ctx := context.Background()
	return client.Create(ctx, pod)
}
```

补充说明一下，为什么要把 `helper` 放在 `/controllers` 目录下。

1. 其一， helper 是 controllers 的实现操作的行为， 算 controller 的一部分
2. 其二， `/Dockerfile` 中在编译 go 代码的时候是有选择性的将代码目录复制进去的。 如果在根目录上自建目录( ex. `/helper` )， 那么就需要额外修改 Dockerfile， 

```Dockerfile
# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/

##  添加
COPY helper/ helper
```

否则编译不通过，报错说找不到 helper 目录。

```bash
Step 10/15 : RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager main.go
 ---> Running in 66ae51d64eca
controllers/redis_controller.go:30:2: no required module provides package github.com/tangx/k8s-operator-demo/helper; to add it:
        go get github.com/tangx/k8s-operator-demo/helper
The command '/bin/sh -c CGO_ENABLED=0 GOOS=linux G
```


## 2. 添加主机并使用 rbac 授权 Operator 操作 k8s

上面实现了创建 k8s pod 的 api 之后， 将 operator 编译并发不到 k8s 集群中。

```bash
# 编译并发布
make docker-build

make install
make deploy

## 创建 operator 实例
ka -f deploy/
```

在创建 redis 实例的时候， operator controller pod 报错如下， serviceaccount 没有权限操作 pods。

```log
2021-11-20T09:10:59.042Z	ERROR	controller.redis	Reconciler error	{"reconciler group": "myapp.tangx.in", "reconciler kind": "Redis", "name": "my-op-redis", "namespace": "default", "error": "创建 redis pod 失败: pods is forbidden: User \"system:serviceaccount:k8s-operator-demo-system:k8s-operator-demo-controller-manager\" cannot create resource \"pods\" in API group \"\" in the namespace \"default\""}
```

这时因为生成的 `/config/rbac/role.yaml` 文件中的 rbac 权限不够。

在 `/controllers/redis_controller.go` 的 `Reconcile` 方法上方，添加注解

格式如下

```go
//+kubebuilder:rbac:groups="资源组名称",resources=资源名称,verbs=get;list;watch;create;update;patch;delete（操作动作）
```


为了要对 pod 具有操作权限， 需要对  增加对应的 kubebuilder 注解。

```go
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete

func (r *RedisReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// statement
}
```

这样我们实现的 operator 就可以对 pod 进行增删改等操作了。


如果不知道 **资源组名称** 和 **资源名称**， 可以使用命令 `kubectl api-resources` 查看

```bash
kubectl api-resources
NAME          SHORTNAMES   APIVERSION         NAMESPACED   KIND
pods          po           v1                 true         Pod
deployments   deploy       apps/v1            true         Deployment
```

1. `groups`: 对应的 **NAME** 字段就是 **资源名称**
2. `rexources`: 对应的 **APIVERSION** 字段， 去掉 `/版本号` 就是 **资源组名称**
    + 因此 pods 隶属于 k8s core 组。  **资源组名称** 就是空 **`""`**
    + deployments 隶属于 k8s apps 组， **资源组名称** 就是 **`apps`**
3. `verbs`: 对应的就是操作动作。

同样在 yaml 文件也可以快速看出隶属于什么 apis

```yaml
# kgd deployment-name -o yaml
apiVersion: apps/v1
kind: Deployment

# kgp -o yaml pod-name
apiVersion: v1
kind: Pod
```


## 3. 注释 webhook 方便测试

为了方便本地测试， 对 `/main.go` 中的 `SetupWebhookWithManager` webhook 相关代码增加了一个条件判断。

如下

```go
	// 本地测试可以注释
	if env := os.Getenv("ENV"); env != "local" {
		if err = (&myappv1.Redis{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Redis")
			os.Exit(1)
		}
	}
```

同时在 Makefile 中也增加相应的环境变量注入

```Makefile
.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	ENV=local go run ./main.go
```

这样就不用在 **测试和编译** 之间来回注释/反注释这段代码了。

