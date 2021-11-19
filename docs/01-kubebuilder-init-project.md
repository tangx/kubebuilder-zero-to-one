# 使用 kuberbuilder 初始化项目



```bash
kubebuilder init --domain tangx.in               
kubebuilder create api --group myapp --version v1 --kind Redis
```

```yaml
apiVersion: myapp.tangx.in/v1
kind: Redis

metadata:
  name: my-op-redis

spec:
  replicas: 1
  port: 3333
```