# Offline / Network Guard

## Overview

The Multi-Org Data & Media Operations Hub is designed for LAN-only deployment. All network access is restricted to private IP ranges, and no outbound connections to external services are made.

## Egress Guard Middleware

**Implementation**: `backend/internal/middleware/middleware.go:32-72`

### How It Works

The `EgressGuardMiddleware` checks every incoming request's client IP against a configured allowlist:

```go
func EgressGuardMiddleware(allowedHosts []string) gin.HandlerFunc {
    return func(c *gin.Context) {
        clientIP := c.ClientIP()
        if !isAllowedHost(clientIP, allowedHosts) {
            appErrors.RespondWithError(c, http.StatusForbidden, appErrors.Forbidden(
                fmt.Sprintf("access denied: client IP %s is not in allowed network range", clientIP),
            ))
            return
        }
        c.Next()
    }
}
```

### IP Matching Logic

The `isAllowedHost()` function supports two matching modes:

1. **CIDR range matching**: If the allowlist entry contains `/`, it is parsed as a CIDR block and `cidr.Contains(ip)` is used
2. **Exact IP matching**: If no `/`, the client IP must match exactly (with special handling for `localhost` mapping to both `127.0.0.1` and `::1`)

```go
func isAllowedHost(clientIP string, allowedHosts []string) bool {
    ip := net.ParseIP(clientIP)
    if ip == nil { return false }

    for _, host := range allowedHosts {
        if strings.Contains(host, "/") {
            _, cidr, err := net.ParseCIDR(host)
            if err != nil { continue }
            if cidr.Contains(ip) { return true }
        } else {
            if clientIP == host { return true }
            if host == "localhost" && (clientIP == "127.0.0.1" || clientIP == "::1") {
                return true
            }
        }
    }
    return false
}
```

Source: `backend/internal/middleware/middleware.go:47-72`

## Default Allowed Hosts

**Configuration**: `backend/internal/config/config.go:64`

```go
AllowedHosts: parseAllowedHosts(envOrDefault("ALLOWED_HOSTS",
    "127.0.0.1,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16"))
```

| CIDR Range | Description | RFC |
|---|---|---|
| `127.0.0.1` | Loopback (exact match) | RFC 5735 |
| `10.0.0.0/8` | Class A private network | RFC 1918 |
| `172.16.0.0/12` | Class B private network | RFC 1918 |
| `192.168.0.0/16` | Class C private network | RFC 1918 |

### Docker Compose Override

In `docker-compose.yml:45`:

```yaml
ALLOWED_HOSTS: "10.0.0.0/8,172.16.0.0/12,192.168.0.0/16,127.0.0.1/8"
```

Note: The Docker Compose configuration uses `127.0.0.1/8` (CIDR) instead of `127.0.0.1` (exact), covering the entire loopback range.

## Blocked-Target Behavior

When a request arrives from an IP outside the allowed ranges:

1. **Response**: HTTP 403 Forbidden
2. **Error payload**:
```json
{
  "code": "FORBIDDEN",
  "message": "access denied: client IP 203.0.113.1 is not in allowed network range",
  "correlationId": "uuid-string"
}
```
3. **Logging**: Warning logged via `logging.Warn("middleware", "egress_guard", "blocked request from non-LAN IP")`
4. **Request processing**: Aborted immediately (`c.Abort()` via `RespondWithError`)

## Middleware Registration

The egress guard is registered as one of the first middleware in the chain:

```go
// backend/internal/router/router.go:41
r.Use(middleware.EgressGuardMiddleware(cfg.AllowedHosts))
```

This means it runs before authentication, RBAC, or any handler logic. Blocked IPs never reach the application layer.

## Configuration Verification

### Checking Current Configuration

The allowed hosts are logged on server startup:

```go
// backend/cmd/server/main.go:56
logging.Info("server", "startup",
    fmt.Sprintf("egress guard active, allowed hosts: %v", cfg.AllowedHosts))
```

### Modifying Allowed Hosts

Set the `ALLOWED_HOSTS` environment variable to a comma-separated list of IPs and/or CIDR ranges:

```bash
ALLOWED_HOSTS="192.168.1.0/24,10.0.0.0/8"
```

### Verifying from Outside LAN

```bash
# From an external IP, this should return 403:
curl -v http://hub-server:8080/health
# Expected: 403 Forbidden

# From within the LAN:
curl -v http://hub-server:8080/health
# Expected: 200 OK
```

## No Outbound Connections

The application makes no outbound HTTP/HTTPS calls to external services:

| Feature | Network Behavior |
|---|---|
| Authentication | Local JWT generation/validation only |
| Database | Docker internal network (mysql:3306) |
| Ingestion connectors | Local filesystem / LAN network shares only |
| Integration endpoints | Configured by admin; restricted to LAN URLs |
| Report generation | Local file system writes |
| Media streaming | Local file system reads |

### Integration Endpoint Restrictions

Integration endpoints (`POST /integrations/endpoints`) are configured with URLs by the system admin. The egress guard does not restrict outbound calls from the backend to integration URLs, but since the system is deployed on a LAN, all configured URLs are expected to be LAN-accessible.

## Docker Network Isolation

```yaml
# docker-compose.yml
services:
  mysql:     # Internal network only, exposed on :3306
  backend:   # Internal network + :8080
  frontend:  # Internal network + :3000
```

All three services communicate over the Docker default bridge network. The only ports exposed to the host are:
- **3000**: Frontend (Nginx reverse proxy)
- **8080**: Backend API
- **3306**: MySQL (for direct database access)

## Health Check (Unguarded)

The `/health` endpoint is registered before the egress guard in the route tree:

```go
// backend/internal/router/router.go:43-47
r.GET("/health", func(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "status":    "healthy",
        "timestamp": time.Now().UTC().Format(time.RFC3339),
    })
})
```

However, since the egress guard is applied as global middleware via `r.Use()`, it still applies to the health endpoint. Only LAN clients can reach it.
