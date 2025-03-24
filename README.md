# PgString - Golang PostgreSQL Query Builder

## Overview

PgString is a lightweight, type-safe PostgreSQL query builder for Go that simplifies database interactions by providing a fluent, chainable API for constructing SQL queries. It leverages struct reflection to automatically extract field names and create parameterized queries.

## Features

- Supports most common SQL operations:
    - SELECT
    - INSERT
    - UPDATE
    - DELETE
    - CREATE TABLE
- Automatic field extraction from structs
- Named parameter support
- Flexible query building with method chaining
- Supports various query clauses:
    - WHERE
    - JOIN (INNER, LEFT, RIGHT, FULL OUTER)
    - ORDER BY
    - LIMIT
    - OFFSET
    - GROUP BY
    - HAVING
    - ON CONFLICT
- Type inference for table creation

## Installation

```bash
go get -u github.com/oliverpaddock/pgstring
```

## Quick Examples

### SELECT Queries

```go
type User struct {
    ID    int    `db:"id"`
    Name  string `db:"name"`
    Email string `db:"email"`
}

// Select all users
query := pgstring.Select(&User{}).From("users")

// Select specific fields
query := pgstring.Select([]string{"id", "name"}).From("users").Where("age > @minAge", map[string]any{"minAge": 18})

// Complex SELECT with joins
query := pgstring.Select(&User{})
    .From("users")
    .Join("INNER", "orders", "users.id = orders.user_id")
    .Where("orders.total > @minTotal", map[string]any{"minTotal": 100})
    .OrderBy("users.name")
    .Limit(10)
```

### INSERT Queries

```go
type Product struct {
    Name  string  `db:"name"`
    Price float64 `db:"price"`
}

product := Product{Name: "Widget", Price: 19.99}
query := pgstring.InsertInto("products").Obj(product).Values(product)
```

### UPDATE Queries

```go
type User struct {
    ID    int    `db:"id"`
    Name  string `db:"name"`
    Email string `db:"email"`
}

user := User{ID: 1, Name: "New Name", Email: "new@email.com"}
query := pgstring.Update("users").Set(user).Where("id = @id", user)
```

### DELETE Queries

```go
query := pgstring.Delete().From("users").Where("id = @userId", map[string]any{"userId": 5})
```

### CREATE TABLE

```go
type User struct {
    ID        int    `db:"id,primarykey"`
    Name      string `db:"name,notnull"`
    Email     string `db:"email,unique"`
    Active    bool   `db:"active"`
    CreatedAt time.Time
}

query := pgstring.CreateTable("users", &User{}, pgstring.TableOptionIfNotExists)
```

## Struct Tag Options

- `db:"fieldname"`: Specify custom column name
- `db:"primarykey"`: Mark as primary key
- `db:"notnull"`: Add NOT NULL constraint
- `db:"unique"`: Add UNIQUE constraint
- `db:"-"`: Ignore field

## Table Creation Options

- `TableOptionIfNotExists`: Create table if not exists
- `TableOptionDrop`: Drop existing table before creating
- `TableOptionDropCascade`: Drop table with cascade

## Advanced Features

### Raw SQL Support

```go
query := pgstring.RawSQL("SELECT * FROM users WHERE status = @status")
```

### Complex Conditions

```go
// IN clause
query := pgstring.Select(&User{}).From("users").In("id", []int{1, 2, 3})

// LIKE clause
query := pgstring.Select(&User{}).From("users").Like("name", "%John%")

// Between clause
query := pgstring.Select(&User{}).From("orders").Between("total", 50, 200)
```

## Performance & Safety

- Uses parameterized queries to prevent SQL injection
- Leverages `pgx` for efficient database interactions
- Minimal overhead with reflection

## Limitations

- Requires structs with exported fields
- Limited to PostgreSQL
- Complex queries might require raw SQL

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.

## License

[Specify your license, e.g., MIT]