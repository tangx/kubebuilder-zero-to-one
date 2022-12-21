# 通过 webhook 进行字段验证

## 通过 kubebuilder 生成代码

```bash
# 创建 api
kubebuilder create   api   --group myapp --version v1 --kind Redis 

# 创建 api 的 webhook
kubebuilder create webhook --group myapp --version v1 --kind Redis --defaulting --programmatic-validation
```

## 增加 webhook 条件

在 `/api/v1/redis_webhook.go` 中增加检查条件。

检查 webhook 被触发有三个条件 `Create / Update / Delete` 时间节点, 分别对应三个方法。

如下是 **创建时检查**

```go
func (r *Redis) ValidateCreate() error {
	redislog.Info("validate create", "name", r.Name)

	// 条件判断
	if r.ObjectMeta.Name == "tangx-in" {
		return fmt.Errorf("不合法名字: tangx-in")
	}

	if r.Spec.Port < 6379 {
		return fmt.Errorf("端口必须大于等于 6379")
	}

	// TODO(user): fill in your validation logic upon object creation.
	return nil
}
```

## 安装 cert-manager 管理证书

```bash
wget -c https://github.com/jetstack/cert-manager/releases/download/v1.6.1/cert-manager.yaml

ka -f cert-manager.yaml
```

## 反注释 kustomize 渲染配置

`/config/default/kustomization.yaml` 中 ， 反注释遗下内容。

```yaml
bases:
- ../webhook        # 引用 webhook 代码
- ../certmanager    # 引用 cert-manager 代码


## 合并
patchesStrategicMerge:
- manager_webhook_patch.yaml
- webhookcainjection_patch.yaml

# the following config is for teaching kustomize how to do var substitution
vars:
# [CERTMANAGER] To enable cert-manager, uncomment all sections with 'CERTMANAGER' prefix.
- name: CERTIFICATE_NAMESPACE # namespace of the certificate CR
  objref:
    kind: Certificate
    group: cert-manager.io
    version: v1
    name: serving-cert # this name should match the one in certificate.yaml
  fieldref:
    fieldpath: metadata.namespace
- name: CERTIFICATE_NAME
  objref:
    kind: Certificate
    group: cert-manager.io
    version: v1
    name: serving-cert # this name should match the one in certificate.yaml
- name: SERVICE_NAMESPACE # namespace of the service
  objref:
    kind: Service
    version: v1
    name: webhook-service
  fieldref:
    fieldpath: metadata.namespace
- name: SERVICE_NAME
  objref:
    kind: Service
    version: v1
    name: webhook-service
```

## 编译安装

1. 清理环境， 卸载之前的， 避免污染

```bash
make uninstall
```


2. 编译带有 webhook 的新代码, 并发布

```
make docker-build

make install

make deploy
```

### 测试

```yaml
apiVersion: myapp.tangx.in/v1
kind: Redis

metadata:
  name: tangx-in

spec:
  replicas: 1
  port: 3333
```

分别执行两次带有不合法的参数

```bash
ka -f deploy/my-op-redis.yml 
    Error from server (端口必须大于等于 6379): error when creating "deploy/my-op-redis.yml": admission webhook "vredis.kb.io" denied the request: 端口必须大于等于 6379

ka -f deploy 
    Error from server (不合法名字: tangx-in): error when creating "deploy/my-op-redis.yml": admission webhook "vredis.kb.io" denied the request: 不合法名字: tangx-in
```


## 注意: 本地测试

在 `/main.go` 中， 注释以下内容忽略 webhook 部署， 方便调试

```go
	// 本地测试可以注释
	if err = (&myappv1.Redis{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "Redis")
		os.Exit(1)
	}
```