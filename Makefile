

kubebuilder.init:
	kubebuilder init --domain tangx.in

kubebuilder.create.api.redis:
	kubebuilder create api --group myapp --version v1 --kind redis
