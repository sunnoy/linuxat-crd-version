# 安装kubebuilder

```bash
version=2.3.1
os=$(go env GOOS)
arch=$(go env GOARCH)

# Download Kubebuilder and extract it to tmp
curl -L https://go.kubebuilder.io/dl/${version}/${os}/${arch} | tar -xz -C /tmp/

# Move to a long-term location and put it on your path
# (you'll need to set the KUBEBUILDER_ASSETS env var if you put it somewhere else)
sudo mv /tmp/kubebuilder_${version}_${os}_${arch} /usr/local/kubebuilder
export PATH=$PATH:/usr/local/kubebuilder/bin
```

# 初始化项目

```bash
export GO111MODULE=on

mkdir -p $GOPATH/src/example; cd $GOPATH/src/example

kubebuilder init --domain my.domain

```

## 增加api

```bash
kubebuilder create api \
 --group cnat \
 --version v1alpha1 \
 --controller \
 --resource \
 --kind At
```

# 编辑 crd 定义

api/v1alpha1/at_types.go

```go
// AtSpec defines the desired state of At
type AtSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of At. Edit At_types.go to remove/update
	Schedule string `json:"schedule,omitempty"`
	Command string 	`json:"command,omitempty"`
}

// AtStatus defines the observed state of At
type AtStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Phase string `json:"phase,omitempty"`

}

const (
	PhasePending = "PENDING"
	PhaseRunning = "RUNNING"
	PhaseDone    = "DONE"
)
```

## 生成crd定义文件

```bash
make manifests
```

## 安装生成的文件到集群中

```bash
make install
# 已经安装了相关crd定义文件
kubectl get crds
kubectl describe crd ats

# 此时还没有控制器
```

# 创建资源文件

```yaml
apiVersion: cnat.my.domain/v1alpha1
kind: At
metadata:
  name: at-sample
spec:
  schedule: "2020-01-30T10:02:00Z"
  command: 'echo "Something from the past told me to do this now."'

# kubectl apply -f at-sample.yaml
```
# 添加输出列信息

主要是添加注释

```go
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:JSONPath=".spec.schedule", name=Schedule, type=string
// +kubebuilder:printcolumn:JSONPath=".status.phase", name=Phase, type=string
// At is the Schema for the ats API
type At struct {
  metav1.TypeMeta   `json:",inline"`
  metav1.ObjectMeta `json:"metadata,omitempty"`

  Spec   AtSpec   `json:"spec,omitempty"`
  Status AtStatus `json:"status,omitempty"`
}
```

# 重新运行

```bash
make install
cd src/example && make run
```

# 控制器抽象机制

控制器抽象机制.md





