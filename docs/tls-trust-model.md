# TLS Trust Model

## Overview

The Multi-Org Data & Media Operations Hub is designed for LAN-only deployment. TLS is available as an opt-in feature for environments that require encrypted internal communication.

## TLS Configuration

**Source**: `backend/internal/config/config.go:27,61`

| Variable | Default | Description |
|---|---|---|
| `ENABLE_TLS` | `false` | Enable HTTPS on the backend server |

### Docker Compose Defaults

```yaml
# docker-compose.yml
environment:
  ENABLE_TLS: "false"
```

When `ENABLE_TLS` is `false` (default), the backend listens on plain HTTP. This is appropriate for LAN-only deployments where the network perimeter is trusted.

## Internal CA Trust Model

For deployments that enable TLS, the following trust model applies:

### Certificate Requirements

| Component | Certificate Type | Trust Anchor |
|---|---|---|
| Backend (Go/Gin) | Server certificate | Internal CA or self-signed |
| Frontend (Nginx) | Server certificate for :3000 | Internal CA or self-signed |
| MySQL | Optional server certificate | Internal CA |
| Inter-service | Backend trusts MySQL CA | CA bundle in container |

### Service Configuration

#### Backend (Go Server)

The HTTP server is configured in `backend/cmd/server/main.go:47-54`:

```go
srv := &http.Server{
    Addr:         addr,
    Handler:      r,
    ReadTimeout:  30 * time.Second,
    WriteTimeout: 30 * time.Second,
    IdleTimeout:  120 * time.Second,
}
```

When TLS is enabled, the server would use `srv.ListenAndServeTLS(certFile, keyFile)` instead of `srv.ListenAndServe()`.

#### Frontend (Nginx)

The Nginx reverse proxy (`frontend/nginx.conf`) forwards `/api/*` requests to the backend. In a TLS-enabled deployment:

1. Nginx terminates TLS for client connections on port 3000/443
2. Nginx proxies to the backend over HTTP on the internal Docker network (trusted)
3. Alternatively, Nginx can proxy over HTTPS if the backend has TLS enabled

#### MySQL

MySQL 8.0 supports TLS natively. The connection DSN in `backend/internal/database/database.go:20-27` uses:

```
?charset=utf8mb4&parseTime=True&loc=Local
```

To enable TLS for MySQL connections, append `&tls=custom` and configure the TLS certificate pool in the Go MySQL driver.

## Certificate Renewal

For LAN deployments with internal CA:

| Aspect | Recommendation |
|---|---|
| CA certificate lifetime | 5-10 years |
| Server certificate lifetime | 1-2 years |
| Renewal method | Re-issue from internal CA, update Docker secrets/volumes |
| Rotation impact | Restart containers after certificate update |
| Client trust | Distribute CA certificate to all client machines |

## Client Configuration

### Browser Clients

For internal CA certificates:
1. Import the CA certificate into the operating system trust store
2. Or import into the browser-specific trust store (Firefox)
3. The Vue SPA makes API calls via Nginx, so only the Nginx certificate needs browser trust

### API Clients

For programmatic API access:
1. Include the CA certificate in the HTTP client's trust pool
2. Or use `InsecureSkipVerify` (not recommended, even on LAN)

## Network Architecture

```
                    LAN Boundary
 +---------------------------------------------+
 |                                             |
 |  Browser ----[HTTPS/HTTP]----> Nginx:3000   |
 |                                   |         |
 |                          [HTTP (Docker net)] |
 |                                   |         |
 |                           Backend:8080      |
 |                                   |         |
 |                          [TCP (Docker net)]  |
 |                                   |         |
 |                           MySQL:3306        |
 |                                             |
 +---------------------------------------------+
```

All inter-service communication occurs on the Docker internal network, which is isolated from external networks. The only exposed ports are:
- **3000**: Frontend (Nginx)
- **8080**: Backend API (direct access)
- **3306**: MySQL (direct access)

## Egress Guard Interaction

The egress guard (`backend/internal/middleware/middleware.go:32-45`) restricts inbound connections to RFC 1918 private ranges regardless of TLS status. This provides a defense-in-depth layer even when TLS is not enabled.

Default allowed hosts (from `backend/internal/config/config.go:64`):
```
127.0.0.1, 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16
```

## Security Headers (Nginx)

The frontend Nginx configuration applies security headers:
- `X-Frame-Options: DENY`
- `X-Content-Type-Options: nosniff`
- `X-XSS-Protection: 1; mode=block`
- `Referrer-Policy: strict-origin-when-cross-origin`

These headers are served regardless of TLS status.
