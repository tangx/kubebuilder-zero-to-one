# 发布 crd controller

1. 设置 docker server 网络代理， 避免编译的时候下载所依赖的 `gcr.io` 镜像失败。
参考文章 [设置 docker server 网路代理](https://tangx.in/2021/11/19/docker-server-network-proxy/)


2. 修改 Makefile, 设置默认 image name

```Makefile
VERSION ?= v$(shell cat .version)

# Image URL to use all building/pushing image targets
IMG ?= cr.docker.tangx.in/jtredis/controller:$(VERSION)
```

3. 修改镜像 pull 策略。 在 `/config/manager/manager.yaml` 配置文件中， 添加 `imagePullPolicy` 策略。
由于本地开发， 并不准备上传到云上， 所以设置为 `IfNotPresent`。

```yaml
    spec:
      securityContext:
        runAsNonRoot: true
      containers:
      - command:
        - /manager
        args:
        - --leader-elect
        image: controller:latest
        name: manager

        ## 由于不上传到镜像仓库， 所以这里以本地编译的版本为准
        imagePullPolicy: IfNotPresent
```

4. 执行编译

```bash
make docker-build 
```

5. 发布

```bash
make deploy
```