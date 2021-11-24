package helper2

import (
	"context"
	"fmt"

	appv1 "github.com/tangx/k8s-operator-demo/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// CreateRedis 创建 redis pod
func CreateRedisPod2(ctx context.Context, client client.Client, redis *appv1.Redis, scheme *runtime.Scheme) error {

	isUpdated := false
	for i := 0; i < redis.Spec.Replicas; i++ {
		name := fmt.Sprintf("%s-%d", redis.Name, i)
		fmt.Println("创建 pod lo :", name)

		// 如果在 k8s 中存在则跳过。 暂不考虑有人直接修改 redis 的 finalizers 的情况
		if isPodExistInK8S(ctx, client, redis.Namespace, name) {
			continue
		}

		pod := getPod2(redis, name, scheme)
		if err := client.Create(ctx, pod); err != nil {
			return err
		}

		// 如果 pod.Name 在 finaliers 中， 则为删后重建。
		if controllerutil.ContainsFinalizer(redis, pod.Name) {
			continue
		}

		// 如果 pod.Name 不在 finalizers 中， 则为新增 pod。
		// 使用 Finalizer 管理创建的 Pod。 当 pod 被删除完的时候，才能删除 redis
		// redis.Finalizers = append(redis.Finalizers, pod.Name)
		controllerutil.AddFinalizer(redis, name)
		isUpdated = true
	}

	// redis.Finalizers 的变更是在本地内存中， 使用 update 更新到 k8s 中
	if isUpdated {
		return client.Update(ctx, redis)
	}
	return nil
}

func getPod2(redis *appv1.Redis, name string, scheme *runtime.Scheme) *corev1.Pod {

	pod := &corev1.Pod{}
	pod.Name = name
	pod.Namespace = redis.Namespace

	// 创建 pod 时添加 OwnerReference
	controllerutil.SetOwnerReference(redis, pod, scheme)

	// 增加 label 便于删除
	pod.ObjectMeta.Labels = map[string]string{
		"app": redis.Name,
	}

	pod.Spec.Containers = []corev1.Container{
		{
			Name:            redis.Name,
			Image:           redis.Spec.Image,
			ImagePullPolicy: corev1.PullIfNotPresent,
			Ports: []corev1.ContainerPort{
				{
					ContainerPort: redis.Spec.Port,
				},
			},
		},
	}

	return pod
}

func DeleteRedis2(ctx context.Context, client client.Client, redis *appv1.Redis) error {

	fmt.Println("进入删除循环咯")

	isUpdated := false
	for _, name := range redis.Finalizers {
		pod, err := getPodFromK8s(ctx, client, redis.Namespace, name)
		if err != nil {
			return err
		}

		if err := client.Delete(ctx, pod); err != nil {
			return fmt.Errorf("删除 pod (%s) 失败: %v\n", name, err)
		}

		controllerutil.RemoveFinalizer(redis, pod.Name)
		isUpdated = true
	}

	if isUpdated {
		return client.Update(ctx, redis)
	}
	return nil
}

func DecreaseRedis2(ctx context.Context, client client.Client, redis *appv1.Redis) error {
	isUpdated := false
	for _, name := range redis.Finalizers[redis.Spec.Replicas:] {
		pod, err := getPodFromK8s(ctx, client, redis.Namespace, name)
		if err != nil {
			return err
		}

		if err := client.Delete(ctx, pod); err != nil {
			return err
		}

		controllerutil.RemoveFinalizer(redis, name)
		isUpdated = true
	}

	if isUpdated {
		return client.Update(ctx, redis)
	}

	return nil
}

// isPodExistInK8S 检测 pod 是否在 k8s 中存在
//   true 为存在
func isPodExistInK8S(ctx context.Context, client client.Client, namespace string, name string) bool {

	_, err := getPodFromK8s(ctx, client, namespace, name)
	return err == nil
}

// getPodFromK8s 从 k8s 中获取 pod
func getPodFromK8s(ctx context.Context, client client.Client, namespace string, name string) (*corev1.Pod, error) {

	key := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}

	pod := &corev1.Pod{}

	// 这里偷懒， 没有进行错误内容检测。
	err := client.Get(ctx, key, pod)
	return pod, err
}
