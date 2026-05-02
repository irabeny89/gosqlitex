// Package gosqlitex provides a high-performance SQLite wrapper for Go, optimized for concurrency and safety using SQLite's Write-Ahead Logging (WAL) mode.
package gosqlitex

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
	_ "modernc.org/sqlite"
)

// MARK: - Structs

// DbClient is a client that is used for reading and writing to the database.
// It manages a read pool for concurrent reads and a single-connection write pool
// to ensure write safety and optimal performance with SQLite WAL mode.
type DbClient struct {
	// Read pool (multi connections). E.g. SELECT.
	readPool *sql.DB
	// Write pool (single connection). E.g. INSERT, UPDATE, DELETE.
	writePool *sql.DB
}

type sqliteConfig struct {
	// dbPath is the path to the database file E.g "app.db".
	dbPath string
	// driver is the driver to use for the database connection. E.g "sqlite".
	driver string
	// mode is the mode to use for the database connection (e.g. "ro", "rw", "rwc" & "memory". Default is "rwc" ).
	mode string
	// pragmas are the pragmas to use for the database connection. E.g []string{"journal_mode(WAL)", "busy_timeout(5000)", "foreign_keys(ON)"}
	pragma []string
	// dsn is a data source name in case you want to use :memory: or configure db url
	dsn string
}

// Config holds the configuration for opening a new database connection pool.
type Config struct {
	// DbPath is the path to the database file (e.g., "app.db").
	DbPath string
	// Driver is the name of the database driver to use (e.g., "sqlite").
	Driver string
	// rDsn is read data source name in case you want to use :memory: or configure db url
	RDsn string
	// wDsn is write data source name in case you want to use :memory: or configure db url
	WDsn string
}

// MARK: - Interfaces

// MARK: - Const & Var

// MARK: - Methods

func (cnf *sqliteConfig) newPool(maxConn int) (*sql.DB, error) {
	// escape hatch for manual config
	if cnf.dsn != "" {
		db, err := sql.Open(cnf.driver, cnf.dsn)
		if err != nil {
			return nil, err
		}
		db.SetMaxOpenConns(maxConn)
		db.SetMaxIdleConns(maxConn)
		// no max lifetime - db will be open until the application closes it
		db.SetConnMaxLifetime(0)
		return db, nil
	}

	const scheme = "file"
	query := url.Values{
		"mode":    []string{cnf.mode},
		"_pragma": cnf.pragma,
	}
	dsn := "file:" + cnf.dbPath + "?" + query.Encode()
	// fmt.Println("DSN:", dsn)
	db, err := sql.Open(cnf.driver, dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(maxConn)
	db.SetMaxIdleConns(maxConn)
	// no max lifetime - db will be open until the application closes it
	db.SetConnMaxLifetime(0)
	return db, nil
}

// Ping checks if the database connection is alive.
func (c *DbClient) Ping() error {
	err := c.readPool.Ping()
	if err != nil {
		return err
	}
	err = c.writePool.Ping()
	if err != nil {
		return err
	}
	return nil
}

// Query executes a query that returns rows, using the read pool. E.g SELECT * FROM users
func (c *DbClient) Query(query string, args ...any) (*sql.Rows, error) {
	return c.readPool.Query(query, args...)
}

// QueryContext executes a query that returns rows, using the read pool. E.g SELECT * FROM users
func (c *DbClient) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return c.readPool.QueryContext(ctx, query, args...)
}

// QueryRow executes a query that returns a single row, using the read pool. E.g SELECT * FROM users WHERE id = 1
func (c *DbClient) QueryRow(query string, args ...any) *sql.Row {
	return c.readPool.QueryRow(query, args...)
}

// QueryRowContext executes a query that returns a single row, using the read pool. E.g SELECT * FROM users WHERE id = 1
func (c *DbClient) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return c.readPool.QueryRowContext(ctx, query, args...)
}

// Exec executes a query that returns a result, using the write pool. E.g INSERT, UPDATE, DELETE, CREATE, DROP, etc
func (c *DbClient) Exec(query string, args ...any) (sql.Result, error) {
	return c.writePool.Exec(query, args...)
}

// ExecContext executes a query that returns a result, using the write pool. E.g INSERT, UPDATE, DELETE, CREATE, DROP, etc
func (c *DbClient) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return c.writePool.ExecContext(ctx, query, args...)
}

// Begin starts a transaction on the write pool.
func (c *DbClient) Begin() (*sql.Tx, error) {
	return c.writePool.Begin()
}

// BeginTx starts a transaction on the write pool.
func (c *DbClient) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return c.writePool.BeginTx(ctx, opts)
}

// Prepare prepares a query for execution. It uses the read pool for read queries and the write pool for write queries.
// Prepare creates a prepared statement for later queries or executions.
// Multiple queries or executions may be run concurrently from the returned statement.
// The caller must call the statement's [*Stmt.Close] method when the statement is no longer needed.
func (c *DbClient) Prepare(query string) (*sql.Stmt, error) {
	if strings.HasPrefix(strings.TrimSpace(strings.ToUpper(query)), "SELECT") {
		return c.readPool.Prepare(query)
	}
	return c.writePool.Prepare(query)
}

// PrepareContext prepares a query for execution. It uses the read pool for read queries and the write pool for write queries.
// PrepareContext creates a prepared statement for later queries or executions.
// Multiple queries or executions may be run concurrently from the returned statement.
// The provided context is used for the preparation of the statement, not for the execution of the statement.
// The caller must call the statement's [*Stmt.Close] method when the statement is no longer needed.
func (c *DbClient) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	if strings.HasPrefix(strings.TrimSpace(strings.ToUpper(query)), "SELECT") {
		return c.readPool.PrepareContext(ctx, query)
	}
	return c.writePool.PrepareContext(ctx, query)
}

func (c *DbClient) createMigTable(ctx context.Context) error {
	_, err := c.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS migrations (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			query BLOB NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TRIGGER IF NOT EXISTS update_mig_updated_at 
		AFTER UPDATE ON migrations
		BEGIN
			UPDATE migrations SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
		END;
	`)
	return err
}

func (c *DbClient) updateDB(ctx context.Context, fn string, q []byte) error {
	tx, err := c.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, string(q))
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
			INSERT INTO migrations (name, query) VALUES (?, ?)
		`, fn, q)
	if err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

// RunMigrationsContext applies all migrations in the given directory into the database.
//
// c is the database client.
//
// ctx is the context.
//
// dir is the path to the migration files.
//
// sep is the separator used in the migration file name. E.g "1_sep_2_sep_3.sql"
//
// This function:
// - Creates the migrations table if it doesn't exist
//
// - Reads all files in the specified directory
//
// - Validates that each file is a valid migration file
//
// - Checks if the migration has already been applied
//
// - Applies the migration if it hasn't been applied
//
// - Records the migration in the migrations table
//
// - Rolls back the transaction if any error occurs
func (c *DbClient) RunMigrationsContext(ctx context.Context, dir, sep string) error {
	if err := c.createMigTable(ctx); err != nil {
		return err
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, f := range files {
		if err = validateFile(f, sep); err != nil {
			return err
		}
		var query []byte
		err = c.QueryRowContext(
			ctx,
			`SELECT query FROM migrations WHERE name = ?`,
			f.Name(),
		).Scan(&query)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}

		sqlBytes, errFile := os.ReadFile(filepath.Join(dir, f.Name()))
		if errFile != nil {
			return errFile
		}

		if err == nil {
			// Migration already applied
			dmp := diffmatchpatch.New()
			if !bytes.Equal(query, sqlBytes) {
				diff := dmp.DiffMain(string(query), string(sqlBytes), false)
				fmt.Printf("Migration mismatch for %s\n", f.Name())
				fmt.Println(dmp.DiffPrettyText(diff))
				return fmt.Errorf("migration content changed for %s", f.Name())
			}
			continue
		}

		// apply migration
		if err = c.updateDB(ctx, f.Name(), sqlBytes); err != nil {
			return err
		}
	}
	return nil
}

// Close closes the database connection.
func (c *DbClient) Close() error {
	if err := c.readPool.Close(); err != nil {
		return err
	}
	if err := c.writePool.Close(); err != nil {
		return err
	}
	return nil
}

func validateFile(f os.DirEntry, sep string) error {
	if f.IsDir() {
		return errors.New("only files are allowed in migrations folder")
	}
	// split the filename on the first separator to get the timestamp.
	v, _, ok := strings.Cut(f.Name(), sep)
	if !ok {
		return errors.New("migration file name separator not found")
	}
	// check if the timestamp is a valid integer
	if _, err := strconv.Atoi(v); err != nil {
		return errors.New("migration file name prefix is not a number")
	}
	return nil
}

// Open opens a new database connection with optimized SQLite settings.
// It returns a DbClient which manages separate read and write pools.
// The default configuration uses WAL mode and optimized pragmas for performance.
func Open(cnf *Config) (*DbClient, error) {
	// sqlite performance default
	var (
		rMaxConn = 8
		wMaxConn = 1
		dbPath   = "app.db"
		driver   = "sqlite"
		pragma   = []string{
			"journal_mode(WAL)",    // WAL (Write-Ahead Logging) mode for better concurrency
			"busy_timeout(5000)",   // Wait 5 seconds for a lock to be released
			"foreign_keys(ON)",     // Enforce foreign key constraints
			"cache_size(64)",       // 64MB cache for caching pages in memory
			"temp_store(MEMORY)",   // Use memory for temporary tables/sorts - faster sort/joins
			"mmap_size(268435456)", // 256MB for memory mapping - read faster and less disk I/O
		}
	)

	if cnf.Driver != "" {
		driver = cnf.Driver
	}

	//! NOTE: sqlite WAL mode requires -shm and -wal files.
	//! Initialize the writer connection first (1 connection).
	//! This is because WAL mode requires the "writer" connection (rwc) to create the -shm and -wal files,
	//! and the "reader" connection (ro) cannot create them.

	// handle escape hatch for manual config
	if cnf.RDsn != "" && cnf.WDsn != "" {
		rConfig := &sqliteConfig{dsn: cnf.RDsn, driver: driver}
		wConfig := &sqliteConfig{dsn: cnf.WDsn, driver: driver}

		w, err := wConfig.newPool(wMaxConn)
		if err != nil {
			return nil, err
		}
		// Ping the writer to ensure the file is created and WAL mode is initialized.
		if err := w.Ping(); err != nil {
			return nil, err
		}

		r, err := rConfig.newPool(rMaxConn)
		if err != nil {
			return nil, err
		}

		client := &DbClient{
			readPool:  r,
			writePool: w,
		}

		return client, nil
	}

	if cnf.DbPath != "" {
		dbPath = cnf.DbPath
	}

	rConfig := &sqliteConfig{
		dbPath: dbPath,
		driver: driver,
		mode:   "ro",
		pragma: pragma,
	}
	wConfig := &sqliteConfig{
		dbPath: dbPath,
		driver: driver,
		mode:   "rwc",
		pragma: append(pragma, "synchronous(NORMAL)"),
	}

	w, err := wConfig.newPool(wMaxConn)
	if err != nil {
		return nil, err
	}
	// Ping the writer to ensure the file is created and WAL mode is initialized.
	if err := w.Ping(); err != nil {
		return nil, err
	}

	r, err := rConfig.newPool(rMaxConn)
	if err != nil {
		return nil, err
	}

	client := &DbClient{
		readPool:  r,
		writePool: w,
	}

	return client, nil
}
