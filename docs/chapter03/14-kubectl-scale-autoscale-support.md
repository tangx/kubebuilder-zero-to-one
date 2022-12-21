# 支持 kubectl scale 和 kubectl autoscale 命令

在 k8s 自定义资源中有关于 scale 和 hpa 的 `subresources` 字段， 只有这些字段被定义的时候才能支持 `scale 和 autoscale` 命令

官方定义如下
> https://kubernetes.io/zh/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#scale-subresource


在 kubebuilde 中， 使用 `//+kubebuilder:subresource:scale` 增加注解， 生成对应的配置。

注意， 未知需要在 `//+kubebuilder:subresource:status` 下方

```go
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas,selectorpath=.status.selector
```

三个关键字段:

1. `specpath`: `specReplicasPath` 指定定制资源内与 `scale.spec.replicas` 对应的 JSON 路径。
    + 此字段为 **必需值** 。
    + 只可以使用 `.spec` 下的 JSON 路径，只可使用带句点的路径。
    + 如果定制资源的 specReplicasPath 下没有取值，则针对 /scale 子资源执行 GET 操作时会返回错误。

2. `statuspath`: `statusReplicasPath` 指定定制资源内与 `scale.status.replicas` 对应的 JSON 路径。
    + 此字段为 **必需值** 。
    + 只可以使用 `.status` 下的 JSON 路径，只可使用带句点的路径。
    + 如果定制资源的 `statusReplicasPath` 下没有取值，则针对 /scale 子资源的 副本个数状态值默认为 0。

3. `selectorpath`: `labelSelectorPath` 指定定制资源内与 `scale.status.selector` 对应的 JSON 路径。
    + 此字段为可选值。
    + **此字段必须设置才能使用 HPA** 。
    + 只可以使用 `.status 或 .spec` 下的 JSON 路径，只可使用带句点的路径。
    + 如果定制资源的 labelSelectorPath 下没有取值，则针对 /scale 子资源的 选择算符状态值默认为空字符串。
    + 此 JSON 路径所指向的字段必须是一个字符串字段（而不是复合的选择算符结构）， 其中包含标签选择算符串行化的字符串形式。


使用之后 `make install` 编译之后, 可以在 `subresources` 下找到响应字段。

```yaml
# config/crd/xxx.yml


---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  annotations:
    controller-gen.kubebuilder.io/version: v0.7.0
  creationTimestamp: null
  name: redis.myapp.tangx.in
spec:
# ... 省略
    subresources:
      scale:
        labelSelectorPath: .spec.selector
        specReplicasPath: .spec.replicas
        statusReplicasPath: .status.replicas
```


## kube scale

使用 `kubectl scale` 命令进行扩缩容

```bash
# k scale --replicas=2 redis/my-op-redis
redis.myapp.tangx.in/my-op-redis scaled



# kgp
NAME                           READY   STATUS    RESTARTS   AGE
my-op-redis-0                  1/1     Running   0          29m
my-op-redis-1                  1/1     Running   0          29m
```


## kube autoscale

```bash
# k autoscale redis my-op-redis --min=3 --max=10
horizontalpodautoscaler.autoscaling/my-op-redis autoscaled


# kgp
NAME                           READY   STATUS    RESTARTS   AGE
my-op-redis-0                  1/1     Running   0          29m
my-op-redis-1                  1/1     Running   0          29m
my-op-redis-2                  1/1     Running   0          4s
```