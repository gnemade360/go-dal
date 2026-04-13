## Examples

Runnable samples live in the separate `go-dal-examples` repository (see `~/ganesh/personal/projects/go-dal-examples`).

### Basic CRUD Service

- Location: `go-dal-examples/cmd/basic-crud`
- Demonstrates:
  - Bootstrapping the PostgreSQL provider
  - Implementing an HTTP API with repository-backed handlers
  - Using UUID primary keys and timestamps

### Adding More Examples

Structure additional demonstrations as sibling directories under `go-dal-examples/cmd/` (e.g., `unit-of-work`, `advanced-queries`). Reference them here and ensure each module's `go.mod` replaces `github.com/gnemade360/go-dal` with the local checkout for development.
