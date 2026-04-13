# Go DAL - Database Abstraction Layer

A flexible and extensible Database Abstraction Layer (DAL) for Go applications that provides a clean separation between business logic and database operations.

## Features

- **Database Agnostic**: Write your business logic once, switch databases without changing application code
- **Repository Pattern**: Clean separation of concerns with repository-based data access
- **Provider Architecture**: Pluggable database providers (PostgreSQL, MySQL, MongoDB, etc.)
- **Migration Support**: Built-in database migration capabilities
- **Transaction Support**: Consistent transaction handling across different databases
- **Type-Safe**: Leverages Go's type system for compile-time safety

## Installation

```bash
go get github.com/gnemade360/go-dal
```

## Quick Start

### 1. Define Your Model

```go
package models

import "time"

type User struct {
    ID        string    `json:"id" db:"id"`
    Name      string    `json:"name" db:"name"`
    Email     string    `json:"email" db:"email"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
    UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

func (u *User) GetID() string {
    return u.ID
}

func (u *User) SetID(id string) {
    u.ID = id
}
```

### 2. Create a Provider

```go
import (
    "context"
    "github.com/gnemade360/go-dal/dal/providers/postgres"
)

// Create PostgreSQL provider
provider := postgres.New()
provider.SetDSN("postgres://user:pass@localhost:5432/mydb?sslmode=disable")

// Connect to database
ctx := context.Background()
if err := provider.Connect(ctx); err != nil {
    log.Fatal("Failed to connect:", err)
}
defer provider.Close()
```

### 3. Use Repository for Data Access

```go
import "github.com/gnemade360/go-dal/dal/repository"

// Create repository
userRepo := repository.NewUserRepository(provider)

// Create a new user
user := &models.User{
    Name:  "John Doe",
    Email: "john@example.com",
}
if err := userRepo.Create(ctx, user); err != nil {
    log.Fatal("Failed to create user:", err)
}

// Find user by ID
found, err := userRepo.FindByID(ctx, user.ID)
if err != nil {
    log.Fatal("Failed to find user:", err)
}

// Update user
user.Name = "Jane Doe"
if err := userRepo.Update(ctx, user); err != nil {
    log.Fatal("Failed to update user:", err)
}

// List all users
users, err := userRepo.FindAll(ctx)
if err != nil {
    log.Fatal("Failed to list users:", err)
}

// Delete user
if err := userRepo.Delete(ctx, user.ID); err != nil {
    log.Fatal("Failed to delete user:", err)
}
```

## Architecture

```
go-dal/
├── dal/
│   ├── interfaces/          # Core interfaces
│   │   ├── database.go      # Database, Transaction, Dialect interfaces
│   │   └── repository.go    # Repository interface
│   │
│   ├── providers/           # Database-specific implementations
│   │   ├── postgres/        # PostgreSQL provider
│   │   ├── mysql/           # MySQL provider (planned)
│   │   └── mongodb/         # MongoDB provider (planned)
│   │
│   ├── repository/          # Repository implementations
│   │   └── simple.go        # Basic repository implementation
│   │
│   └── migrations/          # Migration support
│       └── migrator.go      # Database migration utilities
│
└── cmd/                     # Example applications
    └── basic-crud/          # Basic CRUD example
```

## Running the Example

The repository includes a complete Basic CRUD example that demonstrates the DAL in action.

### Prerequisites

- Go 1.21+
- Podman or Docker
- Make (optional)

### Start the Database

Using Podman (recommended):
```bash
make up
# or directly:
./scripts/podman-postgres.sh start
```

Using Docker:
```bash
make docker-up
# or directly:
docker-compose up -d
```

### Run the Example Application

```bash
make run
# or directly:
go run cmd/basic-crud/main.go
```

### Test the API

```bash
# Health check
curl http://localhost:8080/health

# Create a user
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice Cooper","email":"alice@example.com"}'

# List all users
curl http://localhost:8080/users

# Get specific user (replace {id} with actual UUID)
curl http://localhost:8080/users/{id}

# Update user
curl -X PUT http://localhost:8080/users/{id} \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice Updated","email":"alice.new@example.com"}'

# Delete user
curl -X DELETE http://localhost:8080/users/{id}
```

## Core Concepts

### Provider
The Provider is responsible for database-specific operations. It implements the `Database` interface and handles:
- Connection management
- Query execution
- Transaction handling
- Database-specific SQL dialect

### Repository
The Repository pattern provides a collection-like interface for accessing domain objects. It abstracts the data access logic and provides a more object-oriented view of the persistence layer.

### Entity
Entities are domain objects that have a distinct identity. In go-dal, entities should implement methods for ID management:
```go
type Entity interface {
    GetID() string
    SetID(string)
}
```

## Adding New Providers

To add support for a new database:

1. Create a new provider package in `dal/providers/`
2. Implement the `Database` interface:

```go
type Database interface {
    Connect(ctx context.Context) error
    Close() error
    Execute(ctx context.Context, query string, args ...interface{}) (Result, error)
    Query(ctx context.Context, query string, args ...interface{}) (Rows, error)
    QueryRow(ctx context.Context, query string, args ...interface{}) Row
    Begin(ctx context.Context) (Transaction, error)
    GetDialect() Dialect
}
```

3. Implement the `Dialect` interface for database-specific SQL generation
4. Create provider-specific repository implementations if needed

## Migrations

The DAL includes migration support through the `dal-migrate` command:

```bash
# Run migrations
go run cmd/dal-migrate/main.go up

# Rollback migrations
go run cmd/dal-migrate/main.go down

# Create a new migration
go run cmd/dal-migrate/main.go create <migration_name>
```

## Configuration

The DAL can be configured through environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `DATABASE_URL` | Database connection string | Provider-specific |
| `MAX_CONNECTIONS` | Maximum database connections | 25 |
| `MAX_IDLE_CONNECTIONS` | Maximum idle connections | 5 |
| `CONNECTION_TIMEOUT` | Connection timeout in seconds | 30 |

## Extending the Library

### Custom Repository Methods

You can extend repositories with custom methods:

```go
type UserRepository struct {
    *repository.BaseRepository
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
    query := "SELECT * FROM users WHERE email = $1"
    // Implementation...
}
```

### Custom Entity Types

Create custom entities by implementing the Entity interface:

```go
type Product struct {
    ID    string `db:"id"`
    Name  string `db:"name"`
    Price float64 `db:"price"`
}

func (p *Product) GetID() string { return p.ID }
func (p *Product) SetID(id string) { p.ID = id }
```

## Troubleshooting

### Database Connection Issues

1. Verify your connection string format:
   - PostgreSQL: `postgres://user:pass@host:port/db?sslmode=disable`
   - MySQL: `user:pass@tcp(host:port)/db?parseTime=true`

2. Check if the database server is running:
   ```bash
   # For Podman
   podman ps

   # For Docker
   docker ps
   ```

3. Verify network connectivity to the database

### Migration Issues

1. Ensure the migrations table exists
2. Check migration file naming (must follow timestamp pattern)
3. Verify SQL syntax for your specific database

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## Roadmap

- [x] PostgreSQL provider
- [x] Basic repository pattern
- [x] Migration support
- [ ] MySQL provider
- [ ] MongoDB provider
- [ ] SAP HANA provider
- [ ] Redis cache layer
- [ ] Query builder
- [ ] Soft deletes
- [ ] Audit logging
- [ ] Connection pooling optimization

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For issues, questions, or contributions, please visit the [GitHub repository](https://github.com/gnemade360/go-dal).