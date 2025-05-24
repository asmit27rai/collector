# Setup And Testing 

## FOllow the step

```bash
# Install clusteradm (OCM CLI)
curl -L https://raw.githubusercontent.com/open-cluster-management-io/clusteradm/main/install.sh | bash
```

```bash
bash <(curl -s https://raw.githubusercontent.com/kubestellar/kubestellar/refs/tags/v0.27.2/scripts/create-kubestellar-demo-env.sh) --platform kind
```

```bash
export host_context=kind-kubeflex
export its_cp=its1
export its_context=its1
export wds_cp=wds1
export wds_context=wds1
export wec1_name=cluster1
export wec2_name=cluster2
export wec1_context=cluster1
export wec2_context=cluster2
export label_query_both="location-group=edge"
export label_query_one="name=cluster1"
```

```bash
kubectl --context $wds_context apply -f - <<EOF
apiVersion: control.kubestellar.io/v1alpha1
kind: BindingPolicy
metadata:
  name: nginx-bpolicy
spec:
  clusterSelectors:
  - matchLabels: {location-group: edge}
  downsync:
  - objectSelectors:
    - matchLabels: {app.kubernetes.io/name: nginx}
EOF
```

```bash
kubectl --context $wds_context apply -f - <<EOF
apiVersion: v1
kind: Namespace
metadata:
  labels:
    app.kubernetes.io/name: nginx
  name: nginx
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  namespace: nginx
  labels:
    app.kubernetes.io/name: nginx
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:latest
        ports:
        - containerPort: 80
EOF
```

```bash
cd ~
git clone -b release-1.31 https://github.com/kubernetes/perf-tests.git
```

```bash
export CL2_DIR=~/perf-tests/clusterloader2
cd ~/kubestellar/test/performance/common
./setup-clusterloader2.sh  # For plain Kubernetes
```

```bash
cd $CL2_DIR/testing/load/
nano performance-test-config.yaml
```

```bash
namespaces: 2  # Creates perf-exp-0 and perf-exp-1
K8S_CLUSTER: "true"
OPENSHIFT_CLUSTER: "false" 
tuningSet: "RandomizedLoad"
```

```bash
kubectl config use-context wds1
cd $CL2_DIR
go run cmd/clusterloader.go \
    --testconfig=./testing/load/performance-test-config.yaml \
    --kubeconfig=$HOME/.kube/config \
    --provider=ks \
    --v=2
```

```bash
git clone https://github.com/asmit27rai/collector
cd collector
```

```bash
./collector $HOME/.kube/config wds1 its1 cluster1 2 output s
```
