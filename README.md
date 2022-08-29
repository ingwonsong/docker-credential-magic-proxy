# `docker-credential-magic-proxy`

## Overview

`docker-credential-magic-proxy` is a HTTP proxy injecting the authentication header for accessing private docker registries.
The credentials in `$HOME/.docker/config.json` or `$DOCKER_CONFIG/config.json` will be used for generating the authentication header.
In addition, the docker credential helpers of GCR, ECR, and ACR are included to support the repositories.

Please note that the name of this project is inspired from https://github.com/docker-credential-magic/docker-credential-magic.

## Build

```bash
HUB=${YOUR_DOCKER_REPO} make publish
```

## Let's run

Here, we use GKE for demo purpose. The other platform (AWS or Azure) can be used with the similar settings.

### Preparation

If GKE is used, the workload identity need to be enabled.

```bash
gcloud iam service-accounts add-iam-policy-binding GCP-SERVICE-ACCOUNT-NAME@PROJECT-NAME.iam.gserviceaccount.com \
    --role roles/iam.workloadIdentityUser \
    --member "serviceAccount:PROJECT-NAME.svc.id.goog[magic/magic-service-account]"
```

### Deploy Proxy

```bash
kubectl create namespace magic
cat <<EOF | kubectl apply -n magic -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: magic-service-account
  # In GCP, to access the private registry using the workload identity, service account need to be set up.
  # e.g.)
  # annotations:
  #   "iam.gke.io/gcp-service-account": "GCP-SERVICE-ACCOUNT-NAME@PROJECT-NAME.iam.gserviceaccount.com"
---
apiVersion: v1
kind: Pod
metadata:
  name: docker-credential-magic-proxy
  labels:
    app: docker-credential-magic-proxy
spec:
  serviceAccountName: magic-service-account
  containers:
  - name: proxy
    image: ghcr.io/ingwonsong/docker-credential-magic-proxy/proxy:latest
    args:
    - "--proxy-port"
    - "5000"
EOF
```

### Run Crane without local credentials

```
# Port forwarding to local address.
kubectl port-forward -n magic docker-credential-magic-proxy 5000:5000
# DOCKER_CONFIG is given here to ignore ~/.docker/config.json
DOCKER_CONFIG=/tmp crane ls localhost:5000/forwardto/gcr.io/YOUR-PRIVATE-REPO
```
