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

## API

### `Open(cnf *Config) (*DbClient, error)`

Opens a new database connection with optimized settings.

### `DbClient` Methods

- `Query(query string, args ...any) (*sql.Rows, error)`: Executes a query on the **read pool**.
- `QueryRow(query string, args ...any) *sql.Row`: Executes a query on the **read pool**.
- `Exec(query string, args ...any) (sql.Result, error)`: Executes a query on the **write pool**.
- `Ping() error`: Verifies both read and write pools are alive.

## License

MIT
