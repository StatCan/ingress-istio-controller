#!/bin/bash

set -euxo pipefail

cat <<EOF | kubectl apply -f -
apiVersion: networking.istio.io/v1beta1
kind: Gateway
metadata:
  name: default
  namespace: istio-ingress
spec:
  selector:
    istio: ingress
  servers:
    - port:
        number: 80
        name: http
        protocol: HTTP
      hosts:
        - '*'
---
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: default
  annotations:
    ingressclass.kubernetes.io/is-default-class: 'true'
spec:
  controller: ingress.statcan.gc.ca/ingress-istio-controller
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: test
  namespace: default
spec:
  rules:
    - host: example.ca
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: test
                port:
                  number: 80
EOF
