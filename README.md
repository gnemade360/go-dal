# go-dal

A flexible and extensible Data Access Layer (DAL) library for Go applications.

## Features

- Generic Repository pattern implementation
- Unit of Work pattern for transaction management
- Specification pattern for complex queries
- Support for multiple database backends
- Type-safe operations with Go generics
- Built-in error handling and validation

## Installation

```bash
go get github.com/gnemade360/go-dal
```

## Usage

### Define an Entity

```go
type User struct {
    dal.BaseEntity
    Username string `db:"username"`
    Email    string `db:"email"`
}

func (u User) TableName() string {
    return "users"
}
```

### Create a Repository

```go
userRepo := repository.NewBaseRepository[User](db, nil)

// Create
user := &User{Username: "john", Email: "john@example.com"}
err := userRepo.Create(ctx, user)

// Find by ID
user, err := userRepo.FindByID(ctx, "user-id")

// Find with specification
spec := repository.NewFieldSpec("email", repository.OpEqual, "john@example.com")
users, err := userRepo.Find(ctx, spec)
```

### Use Unit of Work

```go
uow := unitofwork.NewUnitOfWork(db)

err := uow.InTransaction(ctx, func(uow unitofwork.UnitOfWork) error {
    // All operations within this function are transactional
    userRepo := uow.Repository((*User)(nil)).(repository.Repository[User])
    return userRepo.Create(ctx, user)
})
```

## License

MIT
