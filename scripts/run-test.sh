#!/bin/bash

set -euxo pipefail

go run main.go -kubeconfig="$HOME/.kube/config" -default-gateway=istio-ingress/default
