# kubebuilder 从 0 到 1

从 0 开始写一个 Redis Operator

# 目录

1. [使用 kuberbuilder 初始化项目](./01-kubebuilder-init-project.md)
2. [简单跑一跑](./02-simplest-redis-crd.md)
3. [发布 crd controller](./03-deploy-crd-controller.md)
4. [使用注解完整字段值约束](./04-filed-validation-by-comment.md)
5. [通过 webhook 进行字段验证](./05-filed-validation-by-webhook.md)
6. [使用 Operator 创建并发布一个 Pod](./06-create-pod-by-redis-operator.md)
7. K8S 父子资源删除管理
    1. [使用 OwnerReference 管理 redis operator 创建的 Pod](./07.1-delete-pod-by-redis-OwnerReference.md)
    2. [使用 finalizers 管理 redis operator 创建的 Pod](./07.2-delete-pod-by-finalizers.md)
8. [Pod 扩容与缩容](./08-scale-pod.md)
9. [监听 k8s 事件](./09-watch-k8s-event.md)
10. [重建被删除的 Pod](./10-recreate-deleted-pod.md)
11. [使用 controllerutil 优化代码](./11-official-package-optimize.md)
12. [增加 event 事件支持](./12-add-event.md)
13. [添加 Status 状态字段](./13-add-status.md)


## feed

![image](https://tangx.in/assets/images/wx-qrcode.png)

