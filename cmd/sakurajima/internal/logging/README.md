# Log Filtering with slog

This package provides comprehensive log filtering capabilities for Go's `log/slog` package. It allows you to filter logs based on various criteria such as message content, attributes, log levels, and custom conditions.

## Quick Start

### Basic HTTP Noise Filtering

```go
import "within.website/x/cmd/sakurajima/internal/logging"

func main() {
    // Filter out common HTTP noise (health checks, metrics, etc.)
    logging.InitSlogWithHTTPFilter("INFO")

    slog.Info("API request", "path", "/api/users")    // ✅ Logged
    slog.Info("Health check", "path", "/health")      // ❌ Filtered out
}
```

### Component-Based Filtering

```go
func main() {
    // Only log from specific components
    logging.InitSlogWithComponentFilter("DEBUG", "auth", "database", "api")

    slog.Info("user login", "component", "auth")          // ✅ Logged
    slog.Info("cache miss", "component", "cache")         // ❌ Filtered out
}
```

### Advanced Configuration

```go
func main() {
    logging.InitSlogWithFilters("DEBUG", &logging.FilteringConfig{
        NoiseHTTP:         true,                          // Filter HTTP noise
        AllowedComponents: []string{"auth", "api"},       // Component allowlist
        BlockedMessages:   []string{"debug trace"},       // Message blocklist
        MinLevel:          ptr(slog.LevelInfo),           // Additional level filter
    })
}
```

## Filter Types

### 1. Message-Based Filters

Filter logs based on message content:

```go
// Block logs containing specific substrings
filter := FilterByMessage("debug trace", "temporary file")

// Allow only logs containing specific substrings
filter := FilterByMessageAllow("error", "warning")
```

### 2. Attribute-Based Filters

Filter logs based on structured attributes:

```go
// Block logs with specific attribute values
filter := FilterByAttribute("env", "test", "development")

// Allow only logs with specific attribute values
filter := FilterByAttributeAllow("component", "auth", "api")
```

### 3. Level-Based Filters

Additional level filtering beyond the global level:

```go
// Only allow ERROR and above during certain conditions
filter := FilterByLevel(slog.LevelError)
```

### 4. Custom Filters

Create complex custom filtering logic:

```go
sensitiveFilter := func(ctx context.Context, r slog.Record) bool {
    // Filter out logs containing sensitive data
    var hasSensitiveData bool
    r.Attrs(func(a slog.Attr) bool {
        if a.Key == "password" || a.Key == "secret" {
            hasSensitiveData = true
            return false
        }
        return true
    })
    return !hasSensitiveData
}

logger := logging.GetFilteredLogger(sensitiveFilter)
```

## Combining Filters

### AND Logic (all filters must pass)

```go
filter := CombineFilters(
    FilterNoiseHTTP(),
    FilterByComponent("api"),
    FilterByLevel(slog.LevelInfo),
)
```

### OR Logic (any filter can pass)

```go
filter := AnyFilter(
    FilterByAttributeAllow("component", "auth"),
    FilterByLevel(slog.LevelError),
)
```

## Common Use Cases

### HTTP Server Logging

```go
func setupHTTPLogging() http.Handler {
    httpLogger := logging.GetFilteredLogger(
        logging.FilterNoiseHTTP(),
        logging.FilterByAttribute("debug", true), // Skip debug in production
    )

    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        httpLogger.Info("request",
            "method", r.Method,
            "path", r.URL.Path,
            "remote_addr", r.RemoteAddr,
        )
        // ... handle request
    })
}
```

### Environment-Specific Filtering

```go
func setupLogging(env string) {
    var config *logging.FilteringConfig

    switch env {
    case "production":
        config = &logging.FilteringConfig{
            NoiseHTTP:       true,
            MinLevel:        ptr(slog.LevelInfo),
            BlockedMessages: []string{"debug", "trace"},
        }
    case "development":
        config = nil // No filtering in development
    case "testing":
        config = &logging.FilteringConfig{
            AllowedComponents: []string{"test"},
        }
    }

    logging.InitSlogWithFilters("DEBUG", config)
}
```

### Rate-Limited Logging

```go
func rateLimitedFilter() logging.LogFilter {
    lastLog := make(map[string]time.Time)
    return func(ctx context.Context, r slog.Record) bool {
        key := r.Message
        now := time.Now()

        if lastTime, exists := lastLog[key]; exists {
            if now.Sub(lastTime) < time.Minute {
                return false // Rate limit: same message within 1 minute
            }
        }

        lastLog[key] = now
        return true
    }
}
```

## Built-in Filters

| Filter                                   | Description                                              |
| ---------------------------------------- | -------------------------------------------------------- |
| `FilterNoiseHTTP()`                      | Filters common HTTP noise (`/health`, `/metrics`, etc.)  |
| `FilterByMessage(...)`                   | Blocks logs containing specified message substrings      |
| `FilterByMessageAllow(...)`              | Allows only logs containing specified message substrings |
| `FilterByLevel(level)`                   | Filters logs below the specified level                   |
| `FilterByAttribute(key, values...)`      | Blocks logs with specified attribute values              |
| `FilterByAttributeAllow(key, values...)` | Allows only logs with specified attribute values         |
| `FilterByComponent(components...)`       | Allows only logs from specified components               |

## Performance Considerations

- Filters are applied in order, so put the most selective filters first
- Attribute scanning can be expensive for logs with many attributes
- Consider using level-based filtering before more complex attribute filtering
- Filters are applied on every log call, so keep filter logic lightweight

## Best Practices

1. **Use built-in filters when possible** - they're optimized and well-tested
2. **Combine filters efficiently** - use `CombineFilters` for AND logic
3. **Filter early** - apply filters at initialization rather than per-logger
4. **Document filter behavior** - make it clear what logs are being filtered
5. **Test filter logic** - ensure filters work as expected in different scenarios
6. **Monitor filtered logs** - consider metrics for how many logs are being filtered
