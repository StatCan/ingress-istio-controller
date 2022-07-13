#!/bin/bash

set -euxo pipefail

KIND_VERSION=0.14.0
KUBERNETES_VERSION=1.21.12
ISTIO_VERSION=1.14.1
METALLB_VERSION=0.12.1

# Setup temp directory
mkdir -p _out

# Download kind
curl -Lo _out/kind https://github.com/kubernetes-sigs/kind/releases/download/v$KIND_VERSION/kind-linux-amd64 && chmod +x _out/kind

# Setup kind cluster
_out/kind create cluster --image kindest/node:v$KUBERNETES_VERSION --name ingress-istio

# Install metallb (for Load Balancer IPs)
kubectl create namespace metallb-system
kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/v$METALLB_VERSION/manifests/metallb.yaml
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: config
  namespace: metallb-system
data:
  config: |
    address-pools:
      - name: default
        protocol: layer2
        addresses:
          - 172.19.255.200-172.19.255.250
EOF

# Install istio
helm repo add istio https://istio-release.storage.googleapis.com/charts
helm repo update

kubectl create namespace istio-system
helm install istio-base istio/base -n istio-system --version $ISTIO_VERSION
helm install istiod istio/istiod -n istio-system --wait --version $ISTIO_VERSION

kubectl create namespace istio-ingress
kubectl label namespace istio-ingress istio-injection=enabled
helm install istio-ingress istio/gateway -n istio-ingress --wait

echo 'Ready to test!'
