# KServe Manual Installation Guide

This guide describes how to install the KServe stack manually using individual operators and Helm, bypassing the `kserve-installer` meta-operator.

## 1. Prerequisites
- OLM (Operator Lifecycle Manager) installed.

```bash
curl -L https://github.com/operator-framework/operator-lifecycle-manager/releases/download/v0.28.0/install.sh | bash -s v0.28.0
```

## 2. Install Dependencies (Operators)

Apply the following subscriptions to install dependencies via OLM:

```bash
# Install Sail Operator (Istio)
kubectl apply -f kserve-installer/config/manifests/bases/istio.csv.yaml # This is a placeholder, use standard OperatorHub if available

# Install Knative Operator
kubectl apply -f https://github.com/knative/operator/releases/download/knative-v1.14.0/operator.yaml

# Install Cert-Manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.16.5/cert-manager.yaml
```

## 3. Configure Control Planes

Once operators are running, configure the individual stacks:

### Istio
```bash
kubectl create ns istio-system
kubectl apply -f kserve-installer/internal/controller/manifests/istio.yaml
```

### Knative
```bash
kubectl create ns knative-serving
kubectl apply -f kserve-installer/internal/controller/manifests/knative-serving.yaml
```

### KServe
Apply the KServe CRDs and Controller:
```bash
kubectl apply -f kserve-installer/internal/controller/manifests/kserve.yaml
```

## 4. Install Ingress Gateway
```bash
helm upgrade --install istio-ingressgateway istio/gateway \
  -n istio-ingress --create-namespace
```

## 5. Verify
```bash
kubectl apply -f kserve-installer/iris.yaml -n default
kubectl get inferenceservice sklearn-iris
```
