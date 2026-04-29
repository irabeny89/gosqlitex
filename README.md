# gosqlitex

`gosqlitex` is a high-performance SQLite wrapper for Go, optimized for concurrency and safety using SQLite's **Write-Ahead Logging (WAL)** mode.

It manages separate connection pools for reading and writing:

- **Read Pool**: Multiple connections for concurrent read operations (`SELECT`).
- **Write Pool**: A single connection for write operations (`INSERT`, `UPDATE`, `DELETE`) to prevent database locks and ensure write safety.

## Features

- **Optimized for WAL Mode**: Automatically initializes the database in WAL mode.
- **Separate Read/Write Pools**: Maximizes performance by allowing concurrent reads while managing a single writer.
- **Performance Pragmas**: Includes pre-configured SQLite pragmas for optimal speed (MMAP, Cache Size, Busy Timeout, etc.).
- **Pure Go Driver**: Uses `modernc.org/sqlite`, which doesn't require CGO.

## Installation

```bash
go get github.com/irabeny89/gosqlitex
```

## Usage

```go
package main

import (
 "fmt"
 "log"
 "github.com/irabeny89/gosqlitex"
)

func main() {
 // Initialize the client
 client, err := gosqlitex.Open(&gosqlitex.Config{
  DbPath: "app.db",
 })
 if err != nil {
  log.Fatal(err)
 }

 // Ping the database
 if err := client.Ping(); err != nil {
  log.Fatal(err)
 }

 // Execute a write operation
 _, err = client.Exec("CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, name TEXT)")
 if err != nil {
  log.Fatal(err)
 }

 _, err = client.Exec("INSERT INTO users (name) VALUES (?)", "Alice")
 if err != nil {
  log.Fatal(err)
 }

 // Execute a read operation
 var name string
 err = client.QueryRow("SELECT name FROM users WHERE id = ?", 1).Scan(&name)
 if err != nil {
  log.Fatal(err)
 }

 fmt.Printf("User found: %s\n", name)
}
```

## Configuration

### Simple Configuration

The easiest way to get started is by providing a database path. `gosqlitex` will automatically apply optimized pragmas for WAL mode.

```go
// DbPath and Driver are optional - default values will be used if not provided
client, err := gosqlitex.Open(&gosqlitex.Config{
    DbPath: "app.db", // default if not provided
    Driver: "sqlite", // default if not provided
})
```

### Advanced Configuration (Manual DSN)

For full control over the SQLite connection (e.g., in-memory databases, custom pragmas), you can provide manual Data Source Names (DSNs) for both reading and writing.

```go
cnf := &gosqlitex.Config{
    RDsn:   "file:app.db?mode=ro&_pragma=journal_mode(WAL)",
    WDsn:   "file:app.db?mode=rwc&_pragma=journal_mode(WAL)",
}
client, err := gosqlitex.Open(cnf)
```

Example of configuration for an in-memory database

```go
cnf := &gosqlitex.Config{
    RDsn:   ":memory:",
    WDsn:   ":memory:",
}
client, err := gosqlitex.Open(cnf)
```

> [!NOTE]
> When using manual DSNs, `gosqlitex` will use the provided strings directly. Unless its `:memory:` (in-memory database) ensure your `WDsn` has `mode=rwc` (or equivalent) to allow file creation and write access.

## Architecture

`gosqlitex` is designed to handle the nuances of SQLite concurrency:

1. **Read Pool**: Uses multiple connections (default: 8) to allow concurrent read operations.
2. **Write Pool**: Uses a single connection to serialize writes, preventing "database is locked" errors while maintaining high throughput via WAL mode.

## API Reference

### `Open(cnf *Config) (*DbClient, error)`

Initializes the database client.

| Field | Type | Description |
| :--- | :--- | :--- |
| `DbPath` | `string` | Path to the SQLite database file (defaults to `app.db`). |
| `Driver` | `string` | Database driver name (defaults to `sqlite`). |
| `RDsn` | `string` | Manual DSN for the read pool. |
| `WDsn` | `string` | Manual DSN for the write pool. |

### `DbClient` Methods

- **`Query(query string, args ...any) (*sql.Rows, error)`**: Executes a query on the read pool.
- **`QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)`**: Executes a query on the read pool with context.
- **`QueryRow(query string, args ...any) *sql.Row`**: Executes a single-row query on the read pool.
- **`QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row`**: Executes a single-row query on the read pool with context.
- **`Exec(query string, args ...any) (sql.Result, error)`**: Executes a command on the write pool.
- **`ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)`**: Executes a command on the write pool with context.
- **`Begin() (*sql.Tx, error)`**: Starts a transaction on the write pool.
- **`BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)`**: Starts a transaction on the write pool with context.
- **`Ping() error`**: Verifies connectivity for both pools.

## License

MIT
