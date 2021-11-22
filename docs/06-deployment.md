
## problem 01

```bash
Step 10/15 : RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager main.go
 ---> Running in 66ae51d64eca
controllers/redis_controller.go:30:2: no required module provides package github.com/tangx/k8s-operator-demo/helper; to add it:
        go get github.com/tangx/k8s-operator-demo/helper
The command '/bin/sh -c CGO_ENABLED=0 GOOS=linux G

```


```Dockerfile
# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/

##  添加
COPY helper/ helper
```


## problem 02

```
2021-11-20T09:10:59.042Z	ERROR	controller.redis	Reconciler error	{"reconciler group": "myapp.tangx.in", "reconciler kind": "Redis", "name": "my-op-redis", "namespace": "default", "error": "创建 redis pod 失败: pods is forbidden: User \"system:serviceaccount:k8s-operator-demo-system:k8s-operator-demo-controller-manager\" cannot create resource \"pods\" in API group \"\" in the namespace \"default\""}
```