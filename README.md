<div align="center">

# 📦 arkbase

### The GitOps-native database backup daemon with an embedded modern dashboard.

[![Go Report Card](https://goreportcard.com/badge/github.com/blesswinsamuel/arkbase)](https://goreportcard.com/report/github.com/blesswinsamuel/arkbase)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Docker Image](https://img.shields.io/badge/docker-ghcr.io%2Fblesswinsamuel%2Farkbase-blue?logo=docker)](https://github.com/blesswinsamuel/arkbase/pkgs/container/arkbase)
[![OpenAPI](https://img.shields.io/badge/OpenAPI-3.1-green?logo=openapi-initiative)](http://localhost:8080/docs)

<p align="center">
  <a href="#-features">Features</a> •
  <a href="#-quickstart">Quickstart</a> •
  <a href="#-configuration-reference">Configuration</a> •
  <a href="#-web-dashboard--api-docs">Dashboard & API</a> •
  <a href="#-kubernetes-deployment">Kubernetes</a> •
  <a href="#-license">License</a>
</p>

</div>

---

## ⚡ What is `arkbase`?

`arkbase` bridges the gap between **declarative GitOps automation** and **visual operational control**:

- **100% Config-File Driven**: Every database, schedule, S3 bucket, encryption key, and retention policy is codified in a single `config.yaml`. No click-ops or hidden internal database state.
- **Embedded Web Dashboard**: A single binary embeds a modern, responsive [shadcn/ui](https://ui.shadcn.com/) + React dashboard. View backup histories, storage trends, countdown schedules, and trigger on-demand backups with one click.
- **Client-Side AES-256-GCM Encryption**: Encrypt backups before they leave your machine or cluster using authenticated streaming encryption.
- **Grandfather-Father-Son (GFS) Retention**: Granular retention rules (`keep_last`, `hourly`, `daily`, `weekly`, `monthly`, `yearly`) automatically prune stale snapshots across local disks and S3 buckets.
- **Observability Built-In**: Prometheus `/metrics` endpoint out of the box, structured JSON logging, and notifications for Discord, Telegram, and Webhooks.

---

## 📊 Comparison

| Feature | Legacy Cron Scripts | Heavy UI Tools (Databasus) | `arkbase` |
| :--- | :---: | :---: | :---: |
| **GitOps / Declarative Config** | ✅ | ❌ (UI/DB state only) | **✅ (Single `config.yaml`)** |
| **Modern Web Dashboard** | ❌ | ✅ | **✅ (Embedded in single binary)** |
| **Container Footprint** | Small | Heavy (Runs internal DB) | **Tiny (<40 MB, unprivileged)** |
| **Client-Side AES-GCM** | Manual OpenSSL pipes | ✅ | **✅ (Built-in streaming)** |
| **Interactive API Docs** | ❌ | ❌ | **✅ (OpenAPI 3.1 + Scalar at `/docs`)** |
| **Prometheus Metrics** | ❌ | Partial | **✅ Native (`/metrics`)** |

---

## 🚀 Quickstart

### Instant Local Dev (Zero-Docker with `local-compose`, Nix & Bun)

If you have [`local-compose`](https://github.com/blesswinsamuel/local-compose), [Nix](https://nixos.org/), and [Bun](https://bun.sh/) installed, clone and run:

```bash
local-compose up -d
```
This automatically:
- Starts a native PostgreSQL 17 instance via Nix (storing data in `./data/postgres`) and auto-seeds it with sample tables and records (`testdata/seed.sql`).
- Starts the `arkbase` Go daemon via `task dev:api` (`go run ./cmd/arkbase run --config config.dev.yaml`).
- Launches the Vite React frontend on `http://localhost:18081` (`0.0.0.0:18081`) via `task dev:web` with live HMR using Bun.

---

### Run with Docker or Docker Compose

#### 1. Create a `config.yaml`

```yaml
server:
  port: 8080

destinations:
  local_storage:
    type: filesystem
    path: /backups/{{ .Database }}/{{ .Database }}_{{ .Timestamp }}.sql.gz
    retention:
      keep_last: 7
      daily: 7

databases:
  myapp_db:
    engine: postgres
    host: localhost
    port: 5432
    username: postgres
    password: "mypassword"
    database: myapp
    schedule: "0 */6 * * *" # Every 6 hours
    destinations:
      - local_storage
```

### 2. Run with Docker

```bash
docker run -d \
  --name arkbase \
  -p 8080:8080 \
  -v $(pwd)/config.yaml:/etc/arkbase/config.yaml:ro \
  -v $(pwd)/backups:/backups \
  ghcr.io/blesswinsamuel/arkbase:latest
```

Open **`http://localhost:8080`** in your browser to access the dashboard!

---

### 3. Run with Docker Compose

```yaml
services:
  arkbase:
    image: ghcr.io/blesswinsamuel/arkbase:latest
    container_name: arkbase
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - ./config.yaml:/etc/arkbase/config.yaml:ro
      - ./backups:/backups
      - arkbase_data:/var/lib/arkbase
    environment:
      - PG_PASSWORD=supersecret
      - S3_SECRET_KEY=s3secret

volumes:
  arkbase_data:
```

---

## 🛠️ CLI Usage

`arkbase` is distributed as a single static Go binary:

```bash
# Start the scheduler daemon and web dashboard
arkbase run --config config.yaml

# Run an immediate one-off backup for a specific database
arkbase backup production_pg --config config.yaml

# Restore a database from a backup file in a storage target
arkbase restore production_pg --destination offsite_s3 --file backups/postgres/production_pg/production_pg_20260914.sql.gz.enc --config config.yaml

# Export the OpenAPI 3.1 JSON schema
arkbase export-openapi --out openapi.json
```

---

## ⚙️ Configuration Reference

`arkbase` expands environment variables in `${VAR}` or `${VAR:-default}` syntax.

```yaml
server:
  port: 8080                  # HTTP server port (default: 8080)
  data_dir: /var/lib/arkbase  # SQLite run history & temp staging directory
  auth:
    enabled: false            # Set true to enforce HTTP Basic Auth
    username: admin
    password: "${ADMIN_PASSWORD}"

destinations:
  # Filesystem target
  local_backup:
    type: filesystem
    path: /backups/{{ .Database }}/{{ .Database }}_{{ .Timestamp }}.sql.gz
    retention:
      keep_last: 7            # Keep the 7 newest snapshots
      daily: 14               # Keep 1 snapshot per day for 14 days
      weekly: 4               # Keep 1 snapshot per week for 4 weeks
      monthly: 6              # Keep 1 snapshot per month for 6 months

  # S3 Object Storage target (AWS S3, MinIO, Cloudflare R2, Wasabi, Backblaze B2)
  s3_backup:
    type: s3
    endpoint: s3.us-east-1.amazonaws.com
    bucket: my-company-backups
    region: us-east-1
    access_key: "${S3_ACCESS_KEY}"
    secret_key: "${S3_SECRET_KEY}"
    insecure: false
    path: backups/{{ .Engine }}/{{ .Database }}/{{ .Database }}_{{ .Timestamp }}.sql.gz.enc
    encryption:
      enabled: true
      passphrase: "${BACKUP_ENCRYPTION_KEY}" # AES-256-GCM streaming encryption
    retention:
      keep_last: 10
      hourly: 24
      daily: 7
      weekly: 4
      monthly: 12
      yearly: 3

databases:
  main_postgres:
    engine: postgres          # Currently supports postgres
    host: pg.internal
    port: 5432
    username: postgres
    password: "${PG_PASSWORD}"
    database: main_db
    schedule: "0 2 * * *"     # Cron expression
    destinations:
      - local_backup
      - s3_backup

notifications:
  - name: healthchecks
    type: webhook
    url: "https://hc-ping.com/${HEALTHCHECK_UUID}"
    on_events: [success]

  - name: discord_alerts
    type: discord
    webhook_url: "${DISCORD_WEBHOOK_URL}"
    on_events: [failure]

  - name: telegram_alerts
    type: telegram
    bot_token: "${TELEGRAM_BOT_TOKEN}"
    chat_id: "${TELEGRAM_CHAT_ID}"
    on_events: [failure]
```

---

## 🌐 Web Dashboard & API Docs

### Modern Dashboard
Access live status metrics, upcoming backup schedules, and view colorized terminal execution logs for past runs.

### Interactive OpenAPI 3.1 & Scalar Docs
`arkbase` automatically hosts interactive documentation powered by [Scalar](https://scalar.com/):
- **Web UI**: Navigate to `http://localhost:8080/docs`
- **OpenAPI Spec**: `http://localhost:8080/openapi.json`
- **Prometheus Metrics**: `http://localhost:8080/metrics`

---

## ☸️ Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: arkbase
  namespace: database
spec:
  replicas: 1
  selector:
    matchLabels:
      app: arkbase
  template:
    metadata:
      labels:
        app: arkbase
    spec:
      securityContext:
        runAsNonRoot: true
        runAsUser: 10001
        runAsGroup: 10001
        fsGroup: 10001
      containers:
        - name: arkbase
          image: ghcr.io/blesswinsamuel/arkbase:latest
          ports:
            - containerPort: 8080
              name: http
          volumeMounts:
            - name: config
              mountPath: /etc/arkbase/config.yaml
              subPath: config.yaml
              readOnly: true
            - name: data
              mountPath: /var/lib/arkbase
      volumes:
        - name: config
          secret:
            secretName: arkbase-config
        - name: data
          persistentVolumeClaim:
            claimName: arkbase-data-pvc
---
apiVersion: v1
kind: Service
metadata:
  name: arkbase
  namespace: database
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io/port: "8080"
spec:
  ports:
    - port: 8080
      targetPort: http
  selector:
    app: arkbase
```

---

## 🔒 Security

- **Streaming AES-256-GCM**: Encryption keys are derived using PBKDF2 with SHA-256 (100,000 rounds). Files are verified with authentication tags before decryption.
- **Unprivileged Runtime**: Containers run under UID `10001:10001` with zero root requirements.
- **Zero Telemetry**: `arkbase` does not track you or phone home.

---

## 📄 License

Licensed under the [Apache License, Version 2.0](LICENSE).
