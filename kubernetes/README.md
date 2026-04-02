# Deployment Guide

This directory contains Kubernetes manifests and deployment configuration for the Bina Marga Survey Photo Service.

## Prerequisites

- Docker 20.10+
- Kubernetes 1.25+ (or kind/Minikube for local development)
- kubectl 1.25+
- Google Cloud SDK (for GCR deployments)
- PostgreSQL 15+ (for production databases)

## Directory Structure

```
kubernetes/
├── 00-namespace.yaml       # Namespace definition
├── 01-configmap.yaml       # Non-sensitive configuration
├── 02-secret.yaml          # Sensitive data (credentials)
├── 03-deployment.yaml     # Application deployment
├── 04-service-account.yaml # Service account and PDB
├── 05-ingress.yaml         # Ingress configuration
├── 06-hpa.yaml             # Horizontal Pod Autoscaler
├── kustomization.yaml      # Kustomize configuration
└── README.md               # This file
```

## Quick Start

### Local Development with Docker Compose

```bash
# Start services
make docker-up

# View logs
make docker-logs

# Stop services
make docker-down

# Clean up everything
make docker-clean
```

### Kubernetes Deployment

```bash
# Apply all manifests
make k8s-apply

# Check status
make k8s-status

# View logs
make k8s-logs

# Restart deployment
make k8s-restart

# Delete all resources
make k8s-delete
```

## Configuration

### Environment Variables

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `PORT` | HTTP server port | No | `8080` |
| `LOG_LEVEL` | Logging level | No | `info` |
| `DB_HOST` | PostgreSQL host | Yes | - |
| `DB_PORT` | PostgreSQL port | Yes | `5432` |
| `DB_NAME` | Database name | Yes | - |
| `DB_USERNAME` | Database username | Yes | - |
| `DB_PASSWORD` | Database password | Yes | - |
| `GCS_BUCKET_NAME` | GCS bucket name | Yes | - |
| `GOOGLE_APPLICATION_CREDENTIALS` | GCP credentials JSON | Yes | - |

### Kubernetes Secrets

Before deploying to Kubernetes, you must update the `02-secret.yaml` file with actual credentials:

```bash
# Edit the secret file
kubectl edit secret bm-photo-secrets -n bm-photo
```

Or create the secret from literal values:

```bash
kubectl create secret generic bm-photo-secrets \
  --from-literal=DB_USERNAME=postgres \
  --from-literal=DB_PASSWORD=your-password \
  --from-file=GOOGLE_APPLICATION_CREDENTIALS=./service-account.json \
  -n bm-photo
```

## Production Deployment Checklist

1. **Images**: Use tagged images from GCR, not `:latest`
2. **Secrets**: Update all secrets with production credentials
3. **Database**: Use Cloud SQL or managed PostgreSQL
4. **Ingress**: Configure TLS certificates via cert-manager
5. **Resources**: Adjust CPU/memory limits based on load testing
6. **Scaling**: Set appropriate HPA thresholds
7. **Monitoring**: Enable Prometheus metrics scraping
8. **Backup**: Configure database backups

## Scaling

### Horizontal Pod Autoscaler

The HPA is configured to:
- Scale between 2-10 replicas
- Target 70% CPU utilization
- Target 80% memory utilization
- Scale down gradually (5 minute stabilization window)

### Manual Scaling

```bash
make k8s-scale REPLICAS=5
```

## Troubleshooting

### Check Pod Status

```bash
kubectl get pods -n bm-photo
kubectl describe pod <pod-name> -n bm-photo
```

### View Logs

```bash
# All pods
kubectl logs -n bm-photo -l app.kubernetes.io/name=bm-photo --tail=100 -f

# Specific pod
kubectl logs -n bm-photo <pod-name> --tail=100 -f
```

### Port Forward for Local Testing

```bash
make k8s-port-forward LOCAL_PORT=8080
```

### Execute Shell in Pod

```bash
make k8s-exec CMD="sh"
```
