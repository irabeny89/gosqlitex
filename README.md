# gosqlitex

`gosqlitex` is a high-performance SQLite wrapper for Go, optimized for concurrency and safety using SQLite's **Write-Ahead Logging (WAL)** mode.

It manages separate connection pools for reading and writing:

- **Read Pool**: Multiple connections for concurrent read operations (`SELECT`).
- **Write Pool**: A single connection for write operations (`INSERT`, `UPDATE`, `DELETE`) to prevent database locks and ensure write safety.

## Features

- **Pure Go Driver**: Uses `modernc.org/sqlite`, which doesn't require CGO.
- **Optimized for WAL Mode**: Automatically initializes the database in WAL mode.
- **Performance Pragmas**: Includes pre-configured SQLite pragmas for optimal speed (MMAP, Cache Size, Busy Timeout, etc.).
- **Isolated Connection Pools**: A dedicated single-connection writer pool to completely bypass write-lock corruption, combined with a multi-connection reader pool scaling flawlessly across CPU cores.
- **Safe Multi-Environment Path Architecture**: Dynamic DSN string parsers that handle physical paths safely on macOS/Unix (with file:///), relative footprints, and isolated, non-leaking shared in-memory contexts (file:memdb_xxx?cache=shared) for robust unit testing.
- **High-Throughput Tuning**: Operational structures running on modern sync.WaitGroup mechanics, paired with fine-tuned, active memory-mapping (mmap) and page cache configurations.

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
 client, err := gosqlitex.NewDBClient(&gosqlitex.DBConfig{
    Dsn: "app.db",
    // Pragma: []string{} <-- optional custom pragmas if you don't want the default ones
 })
 if err != nil {
  log.Fatal(err)
 }
 defer client.Close()

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
client, err := gosqlitex.NewDBClient(&gosqlitex.DBConfig{
    Dsn: "app.db", // default if not provided
})
```

### Advanced Configuration (Manual DSN)

For full control over the SQLite connection (e.g., in-memory databases, custom pragmas), you can provide manual Data Source Names (DSNs) for both reading and writing.

```go
cnf := &gosqlitex.DBConfig{
  Dsn: app.db,
  Pragma: []string{
    "journal_mode=WAL",
    "synchronous=NORMAL",
    "cache_size=10000",
    "mmap_size=30000000000",
    "busy_timeout=5000",
  },
}
client, err := gosqlitex.NewDBClient(cnf)
```

Example of configuration for an in-memory database

```go
cnf := &gosqlitex.DBConfig{
  Dsn: ":memory:",
  // Pragma: []string{
  //   "journal_mode=WAL",
  //   "synchronous=NORMAL",
  //   "cache_size=10000",
  //   "mmap_size=30000000000",
  //   "busy_timeout=5000",
  // }, <-- optional, default pragmas will be applied
}
client, err := gosqlitex.NewDBClient(cnf)
```

## Architecture

`gosqlitex` is designed to handle the nuances of SQLite concurrency:

1. **Read Pool**: Uses multiple connections (default: 8) to allow concurrent read operations.
2. **Write Pool**: Uses a single connection to serialize writes, preventing "database is locked" errors while maintaining high throughput via WAL mode.

## API Reference

### `Open(cnf *Config) (*DbClient, error)`

Initializes the database client.

| Field    | Type       | Description                                                        |
| :------- | :--------- | :----------------------------------------------------------------- |
| `Dsn`    | `string`   | Path to the SQLite database file or memory e.g app.db or :memory:. |
| `Pragma` | `[]string` | List of pragma to apply. It already comes with optimized defaults. |

### `DbClient` Methods

- **`Query(query string, args ...any) (*sql.Rows, error)`**: Executes a query on the read pool.
- **`QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)`**: Executes a query on the read pool with context.
- **`QueryRow(query string, args ...any) *sql.Row`**: Executes a single-row query on the read pool.
- **`QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row`**: Executes a single-row query on the read pool with context.
- **`Exec(query string, args ...any) (sql.Result, error)`**: Executes a command on the write pool.
- **`ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)`**: Executes a command on the write pool with context.
- **`Prepare(query string) (*sql.Stmt, error)`**: Prepares a statement. Automatically routes `SELECT` queries to the read pool and all other queries to the write pool.
- **`PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)`**: Prepares a statement with context.
- **`Begin() (*sql.Tx, error)`**: Starts a transaction on the write pool.
- **`BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)`**: Starts a transaction on the write pool with context.
- **`RunMigrationsContext(ctx context.Context, dir, sep string) error`**: Applies all migrations in the given directory to the database.
- **`ListMigrationsContext(ctx context.Context) ([]string, error)`**: Returns a list of all applied migration files.
- **`Ping() error`**: Verifies connectivity for both pools.
- **`Close() error`**: Closes both the read and write connection pools.

### Testing And Benchmarking

Use `go test` to run the test suite. Use `go test -bench=.` to run benchmarks.

- `BenchmarkParallelReads` measures how fast `ReadPool` scales across CPU cores.

```sh
goos: darwin
goarch: amd64
pkg: github.com/irabeny89/gosqlitex
cpu: Intel(R) Core(TM) i5-1038NG7 CPU @ 2.00GHz
BenchmarkParallelReads-8   	  432874	      2738 ns/op	     460 B/op	      16 allocs/op
PASS
ok  	github.com/irabeny89/gosqlitex	3.713s
```

- `BenchmarkParallelWrites` measures how `Write-Ahead Logging` handles concurrent writes.

```sh
goos: darwin
goarch: amd64
pkg: github.com/irabeny89/gosqlitex
cpu: Intel(R) Core(TM) i5-1038NG7 CPU @ 2.00GHz
BenchmarkParallelWrites-8   	   29565	     38224 ns/op	     312 B/op	      11 allocs/op
PASS
ok  	github.com/irabeny89/gosqlitex	2.723s
```

## Migrations CLI

`gosqlitex` includes a CLI tool called `mig8` for easy migration management.

### Installation

Install the binary to your `$GOPATH/bin`:

```bash
go install github.com/irabeny89/gosqlitex/cmd/mig8@latest
```

### Usage

You can use the CLI tool in several ways:

#### 1. Global Installation

Install the binary to your `$GOPATH/bin`:

```bash
go install github.com/irabeny89/gosqlitex/cmd/mig8@latest
```

Then use it directly:

```bash
mig8 -db app.db -run
```

#### 2. Run without installing

Use `go run` to execute the latest version directly:

```bash
go run github.com/irabeny89/gosqlitex/cmd/mig8@latest -db app.db -run
```

#### 3. Using `go tool` (Go 1.24+)

If you are using Go 1.24 or later, you can run it as a tool:

```bash
go tool github.com/irabeny89/gosqlitex/cmd/mig8@latest -db app.db -run
```

The CLI supports flags and environment variables for configuration.

#### Environment Variables

- `DB_PATH`: Path to the SQLite database.
- `MIG_DIR`: Path to the migrations directory.

#### Common Commands

**Generate a new migration file:**

```bash
mig8 -dir ./migrations -file add_profile_table
```

**Run pending migrations:**

```bash
mig8 -db app.db -dir ./migrations -run
```

**List applied migrations:**

```bash
mig8 -db app.db -list
```

### Migration Safety

When a migration is applied, `mig8` stores its content in a internal table. If an applied migration file is modified locally, `mig8` will detect the mismatch and error out during the next run to prevent schema drift.

## License

MIT
