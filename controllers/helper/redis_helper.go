package helper

import (
	"context"
	"fmt"
	"time"

	appv1 "github.com/tangx/k8s-operator-demo/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "redis",
					Image: config.Spec.Image,
					Ports: []corev1.ContainerPort{
						{
							Name:          "redis-port",
							Protocol:      "TCP",
							ContainerPort: config.Spec.Port,
						},
					},
				},
			},
		},
	}

	// ctx := context.Background()
	return client.Create(ctx, pod)
}

// CreateRedis 创建 redis pod
func CreateRedisPod2(ctx context.Context, client client.Client, redis *appv1.Redis) error {

	isUpdated := false
	for i := 0; i < redis.Spec.Replicas; i++ {
		name := fmt.Sprintf("%s-%d", redis.Name, i)
		fmt.Println("创建 pod lo :", name)

		// 如果在 k8s 中存在则跳过。 暂不考虑有人直接修改 redis 的 finalizers 的情况
		if isPodExistInK8S(ctx, client, redis.Namespace, name) {
			continue
		}

		pod := getPod2(redis, name)
		if err := client.Create(ctx, pod); err != nil {
			return err
		}

		// 如果 pod.Name 在 finaliers 中， 则为删后重建。
		if isPodExistInFinalizers2(redis, pod.Name) {
			continue
		}

		// 如果 pod.Name 不在 finalizers 中， 则为新增 pod。
		// 使用 Finalizer 管理创建的 Pod。 当 pod 被删除完的时候，才能删除 redis
		redis.Finalizers = append(redis.Finalizers, pod.Name)
		isUpdated = true
	}

	// redis.Finalizers 的变更是在本地内存中， 使用 update 更新到 k8s 中
	if isUpdated {
		return client.Update(ctx, redis)
	}
	return nil
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

func ptrBool(b bool) *bool {
	return &b
}

func getPod2(redis *appv1.Redis, name string) *corev1.Pod {

	pod := &corev1.Pod{}
	pod.Name = name
	pod.Namespace = redis.Namespace

	// 创建 pod 时添加 OwnerReference
	pod.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
		ownerReference(*redis),
	}

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

func isPodExistInFinalizers2(redis *appv1.Redis, name string) bool {
	for _, rname := range redis.Finalizers {
		if rname == name {
			return true
		}
	}

	return false
}

func DeleteRedis2(ctx context.Context, client client.Client, redis *appv1.Redis) error {

	fmt.Println("进入删除循环咯")

	isUpdated := false
	for _, name := range redis.Finalizers {
		pod := getPod2(redis, name)

		if err := client.Delete(ctx, pod); err != nil {
			return fmt.Errorf("删除 pod (%s) 失败: %v\n", name, err)
		}

		deleteFromFinalizers(redis, pod.Name)
		isUpdated = true
	}

	if isUpdated {

		// if err := client.Update(ctx, redis); err != nil {
		// 	return fmt.Errorf("更新 redis 失败: %v\n", err)
		// }

		/*
			当 finalizers 被清空后， k8s 会删除 redis instance
			不需要手动删除
		*/
		// return client.Delete(ctx, redis)

		return client.Update(ctx, redis)
	}
	return nil
}

func deleteFromFinalizers(redis *appv1.Redis, name string) {
	for i, rname := range redis.Finalizers {
		if rname == name {
			redis.Finalizers = append(redis.Finalizers[:i], redis.Finalizers[i+1:]...)

			return
		}
	}
}

func DecreaseRedis2(ctx context.Context, client client.Client, redis *appv1.Redis) error {
	isUpdated := false
	for _, name := range redis.Finalizers[redis.Spec.Replicas:] {
		pod := getPod2(redis, name)

		if err := client.Delete(ctx, pod); err != nil {
			return err
		}

		deleteFromFinalizers(redis, name)
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
	// client.Get(ctx, )
	// key := reconcile.Request{
	// 	NamespacedName: types.NamespacedName{
	// 		Namespace: namespace,
	// 		Name:      name,
	// 	},
	// }
	key := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}

	// 这里偷懒， 没有进行错误内容检测。
	err := client.Get(ctx, key, &corev1.Pod{})
	return err == nil
}
