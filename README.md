# gosqlitex

<!--toc:start-->

- [gosqlitex](#gosqlitex)
  - [Features](#features)
  - [Installation](#installation)
  - [Usage](#usage)
  - [Configuration](#configuration)
    - [Simple Configuration](#simple-configuration)
    - [Advanced Configuration (Manual DSN)](#advanced-configuration-manual-dsn)
  - [Architecture](#architecture)
  - [API Reference](#api-reference)
    - [`Open(cnf *Config) (*DbClient, error)`](#opencnf-config-dbclient-error)
    - [`DbClient` Methods](#dbclient-methods)
    - [Testing And Benchmarking](#testing-and-benchmarking)
  - [Migrations CLI](#migrations-cli)
    - [Installation](#installation-1)
    - [Usage](#usage-1)
      - [1. Global Installation](#1-global-installation)
      - [2. Run without installing](#2-run-without-installing)
      - [3. Using `go tool` (Go 1.24+)](#3-using-go-tool-go-124)
      - [Environment Variables](#environment-variables)
      - [Common Commands](#common-commands)
    - [Migration Safety](#migration-safety)
  - [License](#license)

<!--toc:end-->

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

The example below creates an optimized DB client(disk or memory) and intelligently decides if read or write pool should be used.

```go
package main

import (
 "fmt"
 "log"
 "github.com/irabeny89/gosqlitex"
)

func main() {
 // Initialize the client
 client, err := gosqlitex.DiskDB("app.db")
 // Or for in-memory use MemoryDB
 // client, err := gosqlitex.MemoryDB()
 if err != nil {
  log.Fatal(err)
 }
 defer client.Close()

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

You can create each pool yourself with `CreateDSN` and `DBPool`.

```go
package main

import (
 "fmt"
 "log"
 "github.com/irabeny89/gosqlitex"
)

func main() {
	// write-able data source name (disk file path) with optimized defaults
	dsn := gosqlitex.CreateDSN("app.db", false, false)
	db, err := gosqlitex.DBPool(dsn, 10)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
}
```

You can also manually configure a database client.

```go
package main

import (
 "fmt"
 "log"
 "github.com/irabeny89/gosqlitex"
)

func main() {
	// just like regular SQLite datasource name, add pragmas etc
	dsn := "app.db?mode=rwc"
	db, err := gosqlitex.DBPool(dsn, 10)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
}
```

## Architecture

`gosqlitex` is designed to handle the nuances of SQLite concurrency:

> click to visit for diagrams - [here](https://gitdiagram.com/irabeny89/gosqlitex)

1. **Read Pool**: Uses multiple connections to allow concurrent read operations.
2. **Write Pool**: Uses a single connection to serialize writes, preventing "database is locked" errors while maintaining high throughput via WAL mode.

## API Reference

- `DiskDB(path string) (*DbClient, error)`: creates a new DBClient with the given path.
- `MemoryDB() (*DbClient, error)`: creates a new DBClient with an in-memory database.
- `CreateDSN(path string, isRead, isMemory bool) string`: constructs a DSN (Data Source Name) for the database connection.
- `DBPool(dsn string, maxConnections int) (*DbClient, error)`: creates a db pool for sqlite.

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
- **`RunOneMigration(filename, dir, sep string) error`**: Applies a single migration file to the database.
- **`RunMigrations(dir, sep string) error`**: Applies all migrations in the given directory to the database.
- **`RunMigrationsContext(ctx context.Context, dir, sep string) error`**: Applies all migrations in the given directory to the database.
- **`ListMigrations() ([]string, error)`**: Returns a list of all applied migration files.
- **`ListMigrationsContext(ctx context.Context) ([]string, error)`**: Returns a list of all applied migration files.
- **`Ping() error`**: Verifies connectivity for both pools.
- **`Close() error`**: Closes both the read and write connection pools.

### Testing And Benchmarking

Use `go test` to run the test suite. Use `go test -bench=.` to run benchmarks.

- `BenchmarkParallelReads` measures how fast `ReadPool` scales across CPU cores.
- `BenchmarkParallelWrites` measures how `Write-Ahead Logging` handles concurrent writes.

```sh
goos: darwin
goarch: amd64
pkg: github.com/irabeny89/gosqlitex
cpu: Intel(R) Core(TM) i5-1038NG7 CPU @ 2.00GHz
BenchmarkParallelReads/Reads_concurrently-8         	  586167	      2031 ns/op	     460 B/op	      16 allocs/op
BenchmarkParallelReads/Reads_concurrently_on_memory-8         	  632918	      2131 ns/op	     460 B/op	      16 allocs/op
BenchmarkParallelWrites/Writes_concurrently-8                 	   38444	     31028 ns/op	     312 B/op	      11 allocs/op
BenchmarkParallelWrites/Writes_concurrently_on_memory-8       	   38133	     36770 ns/op	     312 B/op	      11 allocs/op
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
go run github.com/irabeny89/gosqlitex/cmd/mig8@latest -db app.db -dir ./migrations
```

#### 3. Using `go tool` (Go 1.24+)

If you are using Go 1.24 or later, you can run it as a tool:

```bash
go tool github.com/irabeny89/gosqlitex/cmd/mig8@latest -db app.db -dir ./migrations
```

The CLI supports flags and environment variables for configuration.

#### Environment Variables

- `DB_PATH`: Path to the SQLite database.
- `MIG_DIR`: Path to the migrations directory.

#### Common Commands

**Help:**

```bash
mig8 --help
```

**Generate a new migration file:**

```bash
mig8 --dir ./migrations --title "add profile table"
```

**Run pending migrations:**

```bash
mig8 -db app.db -dir ./migrations
```

**Run a specific migration:**

```bash
mig8 -db app.db -dir ./migrations --file <migration_name>
```

### Migration Safety

When a migration is applied, `mig8` stores its content in a internal table. If an applied migration file is modified locally, `mig8` will detect the mismatch and error out during the next run to prevent schema drift.

## License

MIT
