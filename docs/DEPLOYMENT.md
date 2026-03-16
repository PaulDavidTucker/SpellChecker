# Deployment Guide

## Quick Start

### Using Go

```bash
# Clone the repository
git clone <repository-url>
cd SpellChecker

# Run the server
go run cmd/server/main.go

# Server will start on port 8080
```

### Using Docker

```bash
# Build the image
docker build -t spellchecker:latest .

# Run the container
docker run -d \
  -p 8080:8080 \
  --name spellchecker \
  spellchecker:latest
```

### Using Docker Compose

```bash
# Start the service
docker-compose up -d

# View logs
docker-compose logs -f

# Stop the service
docker-compose down
```

## Production Deployment

### Docker Compose (Recommended)

The included `docker-compose.yaml` provides a production-ready setup:

```yaml
version: '3.8'

services:
  spellchecker:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
    environment:
      - DICT_PATH=/dictionaries/en_gb.txt
      - BASE_ALLOWLIST_PATH=/config/base-allowlist.yaml
      - PROFILES_DIR=/config/profiles
      - LISTEN_ADDR=:8080
      - HEALTH_CHECK_URL=http://localhost:8080/health
    restart: unless-stopped

**Features:**
- Dictionary and config baked into image
- Automatic restart on failure
- Docker health checks enabled (uses `-health-check` with configurable URL)
- Stateless design (easy to scale)

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DICT_PATH` | `/dictionaries/en_gb.txt` | Path to dictionary file inside container |
| `BASE_ALLOWLIST_PATH` | `/config/base-allowlist.yaml` | Base allowlist config path |
| `PROFILES_DIR` | `/config/profiles` | Directory containing profile YAML files |
| `LISTEN_ADDR` | `:8080` | Server bind address |
| `HEALTH_CHECK_URL` | `http://localhost:8080/health` | URL for health check endpoint (used by `-health-check` flag and Docker healthcheck) |

### Custom Configuration

#### 1. Custom Dictionary

Place your dictionary file in the `dictionaries/` directory:

```
dictionaries/
├── en_gb.txt          # Default English (British) dictionary
└── custom.txt         # Your custom dictionary
```

Update `docker-compose.yaml`:

```yaml
environment:
  - DICT_PATH=/dictionaries/custom.txt
```

**Dictionary Format:**
```
word frequency
hello 100000
world 50000
test 25000
```

#### 2. Custom Profiles

Add profiles to `config/profiles/`:

```yaml
# config/profiles/my-profile.yaml
profile_id: my-profile
description: "My custom profile"
terms:
  - CompanyName
  - ProductName
  - TechnicalTerm
```

#### 3. Base Allowlist

Edit `config/base-allowlist.yaml` for global terms:

```yaml
terms:
  - Kubernetes
  - Docker
  - PostgreSQL
```

## Kubernetes Deployment

### Basic Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: spellchecker
spec:
  replicas: 3
  selector:
    matchLabels:
      app: spellchecker
  template:
    metadata:
      labels:
        app: spellchecker
    spec:
      containers:
      - name: spellchecker
        image: spellchecker:latest
        ports:
        - containerPort: 8080
        env:
        - name: LISTEN_ADDR
          value: ":8080"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: spellchecker
spec:
  selector:
    app: spellchecker
  ports:
  - port: 80
    targetPort: 8080
  type: ClusterIP
```

### With ConfigMap for Custom Terms

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: spellchecker-config
data:
  custom-profile.yaml: |
    profile_id: custom
    description: "Custom profile"
    terms:
      - MyCompany
      - MyProduct
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: spellchecker
spec:
  template:
    spec:
      containers:
      - name: spellchecker
        image: spellchecker:latest
        volumeMounts:
        - name: config
          mountPath: /config/profiles
        env:
        - name: PROFILES_DIR
          value: "/config/profiles"
      volumes:
      - name: config
        configMap:
          name: spellchecker-config
```

## Health Checks

### HTTP Health Endpoint

```bash
curl http://localhost:8080/health
```

Response:
```json
{
  "status": "ok"
}
```

### CLI Health Check

```bash
# Run inside container or on host with server running
./server -health-check

# With custom URL (e.g., for different environments)
HEALTH_CHECK_URL=http://spellchecker:8080/health ./server -health-check

# Exit codes:
# 0 - Healthy
# 1 - Unhealthy
```

### Docker Health Check

The Dockerfile includes a health check that uses the `-health-check` flag:

```dockerfile
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
  CMD ["/spellcheck", "-health-check"]
```

This runs the health check internally by making an HTTP request to the health endpoint. The URL can be customized via the `HEALTH_CHECK_URL` environment variable (default: `http://localhost:8080/health`).

## Monitoring

### Basic Metrics (Future Enhancement)

Planned Prometheus metrics:

```
spellcheck_requests_total
spellcheck_request_duration_seconds
spellcheck_misspellings_found_total
spellcheck_dictionary_size
```

### Logging

The service logs to stdout:

```
2026/03/16 00:00:00 Loaded dictionary: 94231 words
2026/03/16 00:00:00 Loaded 3 profiles
2026/03/16 00:00:00 Listening on :8080
2026/03/16 00:00:01 POST /check 200 783.496µs
```

### Log Aggregation

Docker Compose with log rotation:

```yaml
services:
  spellchecker:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

## Security Considerations

### 1. Network Security

- Service listens on localhost by default in Docker
- Use reverse proxy (nginx, traefik) for external access
- Enable TLS in production

### 2. Input Validation

- JSON parsing limits request size
- Base64 decoding prevents injection
- No code execution from user input

### 3. Container Security

- Uses `scratch` image (minimal attack surface)
- No shell available
- Read-only filesystem where possible

### 4. Reverse Proxy Example (nginx)

```nginx
server {
    listen 443 ssl;
    server_name spellcheck.example.com;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    location / {
        proxy_pass http://spellchecker:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        
        # Rate limiting
        limit_req zone=api burst=10 nodelay;
    }
}
```

## Troubleshooting

### Service Won't Start

**Problem:** Dictionary file not found

**Solution:**
```bash
# Check dictionary path
ls dictionaries/en_gb.txt

# Verify Dockerfile copies dictionaries
```

### High Memory Usage

**Problem:** Dictionary too large

**Solution:**
- Use smaller dictionary file
- Reduce number of profiles
- Profile the memory usage

### Slow Response Times

**Problem:** High latency

**Solution:**
- Check CPU resources
- Increase prefix length (already optimized)
- Add caching layer (future enhancement)

### Profile Not Loading

**Problem:** Profile YAML invalid

**Solution:**
```bash
# Validate YAML syntax
yamllint config/profiles/my-profile.yaml

# Check profile_id matches filename
```

## Performance Tuning

### Dictionary Size

- Default: ~94,000 words
- Memory usage: ~50MB
- Load time: ~2-3 seconds

### Concurrent Requests

- Stateless design supports concurrent requests
- No shared mutable state
- Each request creates its own result set

### Scaling

**Vertical:**
- More CPU = faster lookups
- More RAM = larger dictionaries

**Horizontal:**
- Run multiple instances behind load balancer
- Each instance loads its own copy
- No session affinity required

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Build and Deploy

on:
  push:
    branches: [ main ]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.26'
    
    - name: Run tests
      run: go test ./...
    
    - name: Build binary
      run: go build -o server ./cmd/server
    
    - name: Build Docker image
      run: docker build -t spellchecker:${{ github.sha }} .
    
    - name: Push to registry
      run: |
        docker tag spellchecker:${{ github.sha }} registry.example.com/spellchecker:latest
        docker push registry.example.com/spellchecker:latest
```

## Backup and Recovery

### Dictionary Backup

The dictionary is baked into the image, but you may want to backup custom terms:

```bash
# Backup custom profiles
tar czf profiles-backup.tar.gz config/profiles/

# Backup base allowlist
cp config/base-allowlist.yaml base-allowlist-backup.yaml
```

### Recovery

```bash
# Restore from backup
tar xzf profiles-backup.tar.gz
cp base-allowlist-backup.yaml config/base-allowlist.yaml

# Rebuild and restart
docker-compose down
docker-compose up --build -d
```

## Migration Guide

### From v1 to v2

If upgrading:

1. Backup existing config
2. Update dictionary format if changed
3. Test with harness: `cd harness && go run .`
4. Deploy new version
5. Verify health endpoint

### Database-Free Operation

No database to migrate! All configuration is file-based:
- YAML files in `config/`
- Text files in `dictionaries/`
