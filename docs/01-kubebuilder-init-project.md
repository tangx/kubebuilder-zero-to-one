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

```bash
# 安装
make install

# 卸载
make uninstall
```

查看 crd

```bash

k get crd |grep tangx.in

    redis.myapp.tangx.in                       2021-11-19T06:16:43Z
```