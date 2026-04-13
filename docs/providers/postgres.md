## PostgreSQL Provider

The PostgreSQL provider (`dal/providers/postgres`) is the reference implementation for SQL backends.

### Setup

```go
provider := postgres.New()
provider.SetDSN("postgres://user:pass@localhost:5432/db?sslmode=disable")

ctx := context.Background()
if err := provider.Connect(ctx); err != nil {
    log.Fatalf("connect: %v", err)
}
defer provider.Close()
```

### Capabilities

- Connection pooling with configurable open/idle limits.
- Query helpers: `Exec`, `Query`, `QueryRow` with context support.
- Transaction support via `BeginTx` returning the standard DAL transaction interface.
- Dialect features: numbered placeholders (`$1`), quoted identifiers, support for `RETURNING`.

### Health Checks

Use `provider.Ping(ctx)` to verify connectivity during readiness checks.

### Usage Tips

- Inject the provider into repositories or units of work.
- When running locally, pair with the `go-dal-examples/cmd/basic-crud` service for end-to-end testing.
