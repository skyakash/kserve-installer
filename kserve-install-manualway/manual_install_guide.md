# KServe Manual Installation Guide (Operator-Based)

This guide provides the exact commands to replicate the KServe installation on a clean Kubernetes cluster using the Operator Lifecycle Manager (OLM).

## Prerequisites
Ensure `kubectl` is pointed at your clean cluster.

### 1. Install OLM (Operator Lifecycle Manager)
Installs the framework for managing operators.

```bash
curl -L https://github.com/operator-framework/operator-lifecycle-manager/releases/download/v0.28.0/install.sh | bash -s v0.28.0
```

### 2. Install Gateway API CRDs
Required for Istio and Knative newer versions. Note: We use the experimental channel for broader feature support (e.g. GAMMA).

```bash
# Option A: Standard Channel (Recommended for Stability)
# Includes GA features (Gateway, HTTPRoute) and GRPCRoute. Sufficient for most standard KServe inference use cases.
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.1.0/standard-install.yaml

# Option B: Experimental Channel (For advanced features)
# Includes everything in Standard plus experimental features like TCPRoute/TLSRoute.
# kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.1.0/experimental-install.yaml
```

## Operator Installation

### 3. Install OperatorHub Catalog
Usually pre-installed on OpenShift/some K8s, but essential for OLM on Kind/Minikube.

```bash
# Verify if installed
kubectl get catalogsources -n olm

# If missing, install the Community Operators catalog:
cat <<EOF > kserve-install/operatorhub-catalog.yaml
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: operatorhubio-catalog
  namespace: olm
spec:
  sourceType: grpc
  image: quay.io/operatorhubio/catalog:latest
  displayName: Community Operators
  publisher: OperatorHub.io
  updateStrategy:
    registryPoll:
      interval: 60m
EOF

kubectl apply -f kserve-install/operatorhub-catalog.yaml
```

### 4. Create Installation Directory & Namespaces
Organize manifests and prepare namespaces to avoid race conditions.

```bash
mkdir -p kserve-install
kubectl create ns istio-system
kubectl create ns knative-serving
kubectl create ns kserve-test
```

### 5. Install Operators (Cert-Manager, Sail, Knative)
Create the subscription manifests.

**Cert-Manager Subscription:**
```bash
cat <<EOF > kserve-install/cert-manager-sub.yaml
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: cert-manager
  namespace: operators
spec:
  channel: stable
  name: cert-manager
  source: operatorhubio-catalog
  sourceNamespace: olm
EOF
```

**Sail Operator (Istio) Subscription:**
```bash
cat <<EOF > kserve-install/sail-operator-sub.yaml
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: sailoperator
  namespace: operators
spec:
  channel: stable-1.28
  name: sailoperator
  source: operatorhubio-catalog
  sourceNamespace: olm
EOF
```

**Knative Operator Subscription:**
```bash
cat <<EOF > kserve-install/knative-operator-sub.yaml
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: knative-operator
  namespace: operators
spec:
  channel: stable
  name: knative-operator
  source: operatorhubio-catalog
  sourceNamespace: olm
EOF
```

**Apply Subscriptions:**
```bash
kubectl apply -f kserve-install/cert-manager-sub.yaml
kubectl apply -f kserve-install/sail-operator-sub.yaml
kubectl apply -f kserve-install/knative-operator-sub.yaml

# Wait for operators to be ready
kubectl get csv -n operators -w
```

## Component Configuration

### 6. Configure Istio & Ingress Gateway (via Sail Operator)
Deploys the Istio Control Plane and Ingress Gateway using the Operator-native API.

```bash
cat <<EOF > kserve-install/istio.yaml
apiVersion: sailoperator.io/v1
kind: Istio
metadata:
  name: default
spec:
  version: v1.28.3
  namespace: istio-system
  values:
    global:
      meshID: mesh1
      multiCluster:
        clusterName: docker-desktop
EOF

kubectl apply -f kserve-install/istio.yaml
# Verification
kubectl get istio -n istio-system
kubectl get pods -n istio-system
```

### 7. Configure Istio Ingress Gateway (via Helm)
Installs the gateway for traffic entry. (Note: The Sail Operator Istio CRD does not yet support inline gateway configuration in this version).

```bash
helm repo add istio https://istio-release.storage.googleapis.com/charts
helm repo update
helm install istio-ingressgateway istio/gateway -n istio-system --version 1.28.3
```

### 8. Configure Knative Serving (via Operator)
Deploys Knative Serving components.

```bash
cat <<EOF > kserve-install/knative-serving.yaml
apiVersion: operator.knative.dev/v1beta1
kind: KnativeServing
metadata:
  name: knative-serving
  namespace: knative-serving
spec:
  # version: 1.11.0  <-- Removing version allows operator to choose latest/stable
  config:
    domain:
      example.com: ""
EOF

kubectl apply -f kserve-install/knative-serving.yaml
# Verification
kubectl get knativeserving -n knative-serving
kubectl get pods -n knative-serving
```

## KServe Configuration

### 9. Install KServe Control Plane
Installs KServe v0.16.0.

```bash
# Install KServe CRDs and Controller
kubectl apply --server-side -f https://github.com/kserve/kserve/releases/download/v0.16.0/kserve.yaml

# Verification: Wait for CRDs and Controller to catch up
# 1. Check CRDs
kubectl get crd inferenceservices.serving.kserve.io
# 2. Check Controller Pods (should be Running)
kubectl get pods -n kserve

# Install Default Runtimes (Important for sklearn, xgboost, etc.)
kubectl apply -f https://github.com/kserve/kserve/releases/download/v0.16.0/kserve-cluster-resources.yaml

# Verification
kubectl get clusterservingruntimes
# You should see: kserve-sklearnserver, kserve-torchserve, etc.
```

**Critical Fix: Restart Controller**
If KServe starts before Knative is ready, it may enter a "ServerlessModeRejected" state. Restarting it fixes this.

```bash
kubectl rollout restart deployment kserve-controller-manager -n kserve
```

## Verification (Sample App)

### 10. Deploy InferenceService

```bash
cat <<EOF > kserve-install/iris.yaml
apiVersion: "serving.kserve.io/v1beta1"
kind: "InferenceService"
metadata:
  name: "sklearn-iris"
  namespace: kserve-test
spec:
  predictor:
    model:
      modelFormat:
        name: sklearn
      storageUri: "gs://kfserving-examples/models/sklearn/1.0/model"
      resources:
        requests:
          cpu: "100m"
          memory: "512Mi"
        limits:
          cpu: "1"
          memory: "1Gi"
EOF

kubectl apply -f kserve-install/iris.yaml
```

### 11. Test Inference
Wait for the service to be ready (`kubectl get isvc -n kserve-test`).

```bash
# Create input file
cat <<EOF > iris-input.json
{
  "instances": [
    [6.8, 2.8, 4.8, 1.4],
    [6.0, 3.4, 4.5, 1.6]
  ]
}
EOF

# Send request (assuming localhost access to Ingress Gateway)
curl -v -H "Host: sklearn-iris-predictor.kserve-test.example.com" \
     -H "Content-Type: application/json" \
     http://localhost/v1/models/sklearn-iris:predict \
     -d @iris-input.json
```
