// Package gosqlitex provides a high-performance SQLite wrapper for Go, optimized for concurrency and safety using SQLite's Write-Ahead Logging (WAL) mode with modernc.org/sqlite driver.
//
// It uses a read pool for concurrent reads and a single-connection write pool to ensure write safety and optimal performance with SQLite WAL mode.
package gosqlitex

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// DBClient is a client that is used for reading and writing to the database.
// It manages a read pool for concurrent reads and a single-connection write pool
// to ensure write safety and optimal performance with SQLite WAL mode.
type DBClient struct {
	// Read pool (multi connections). E.g. SELECT.
	ReadPool *sql.DB
	// Write pool (single connection). E.g. INSERT, UPDATE, DELETE.
	WritePool *sql.DB
}

// Ping checks if the database connection is alive.
func (c *DBClient) Ping() error {
	err := c.ReadPool.Ping()
	if err != nil {
		return err
	}
	err = c.WritePool.Ping()
	if err != nil {
		return err
	}
	return nil
}

// Query executes a query that returns rows, using the read pool. E.g SELECT * FROM users
func (c *DBClient) Query(query string, args ...any) (*sql.Rows, error) {
	return c.ReadPool.Query(query, args...)
}

// QueryContext executes a query that returns rows, using the read pool. E.g SELECT * FROM users
func (c *DBClient) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return c.ReadPool.QueryContext(ctx, query, args...)
}

// QueryRow executes a query that returns a single row, using the read pool. E.g SELECT * FROM users WHERE id = 1
func (c *DBClient) QueryRow(query string, args ...any) *sql.Row {
	return c.ReadPool.QueryRow(query, args...)
}

// QueryRowContext executes a query that returns a single row, using the read pool. E.g SELECT * FROM users WHERE id = 1
func (c *DBClient) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return c.ReadPool.QueryRowContext(ctx, query, args...)
}

// Exec executes a query that returns a result, using the write pool. E.g INSERT, UPDATE, DELETE, CREATE, DROP, etc
func (c *DBClient) Exec(query string, args ...any) (sql.Result, error) {
	return c.WritePool.Exec(query, args...)
}

// ExecContext executes a query that returns a result, using the write pool. E.g INSERT, UPDATE, DELETE, CREATE, DROP, etc
func (c *DBClient) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return c.WritePool.ExecContext(ctx, query, args...)
}

// Begin starts a transaction on the write pool.
func (c *DBClient) Begin() (*sql.Tx, error) {
	return c.WritePool.Begin()
}

// BeginTx starts a transaction on the write pool.
func (c *DBClient) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return c.WritePool.BeginTx(ctx, opts)
}

// Prepare prepares a query for execution. It uses the read pool for read queries and the write pool for write queries.
// Prepare creates a prepared statement for later queries or executions.
// Multiple queries or executions may be run concurrently from the returned statement.
// The caller must call the statement's [*Stmt.Close] method when the statement is no longer needed.
func (c *DBClient) Prepare(query string) (*sql.Stmt, error) {
	if strings.HasPrefix(strings.TrimSpace(strings.ToUpper(query)), "SELECT") {
		return c.ReadPool.Prepare(query)
	}
	return c.WritePool.Prepare(query)
}

// PrepareContext prepares a query for execution. It uses the read pool for read queries and the write pool for write queries.
// PrepareContext creates a prepared statement for later queries or executions.
// Multiple queries or executions may be run concurrently from the returned statement.
// The provided context is used for the preparation of the statement, not for the execution of the statement.
// The caller must call the statement's [*Stmt.Close] method when the statement is no longer needed.
func (c *DBClient) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	if strings.HasPrefix(strings.TrimSpace(strings.ToUpper(query)), "SELECT") {
		return c.ReadPool.PrepareContext(ctx, query)
	}
	return c.WritePool.PrepareContext(ctx, query)
}

// createMigTable creates the migrations table if it does not exist.
func (c *DBClient) createMigTable() error {
	_, err := c.Exec(`
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

// createMigTable creates the migrations table if it does not exist.
func (c *DBClient) createMigTableContext(ctx context.Context) error {
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

// updateMigDB updates the migrations table with the given migration name and query.
func (c *DBClient) updateMigDB(fn string, q []byte) error {
	tx, err := c.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			fmt.Printf("Rollback error: %v\n", err)
		}
	}()
	if _, err = tx.Exec(string(q)); err != nil {
		return err
	}
	if _, err = tx.Exec(`
			INSERT INTO migrations (name, query) VALUES (?, ?)
		`, fn, q); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

// updateMigDB updates the migrations table with the given migration name and query.
func (c *DBClient) updateMigDBContext(ctx context.Context, fn string, q []byte) error {
	tx, err := c.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			fmt.Printf("Rollback error: %v\n", err)
		}
	}()
	if _, err = tx.ExecContext(ctx, string(q)); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `
			INSERT INTO migrations (name, query) VALUES (?, ?)
		`, fn, q); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (c *DBClient) processMigFile(filename, dir string, sep string) error {
	if err := validateFilename(filename, sep); err != nil {
		return err
	}
	// sqlBytes holds the file content expected to be sql
	sqlBytes, err := os.ReadFile(filepath.Join(dir, filename))
	if err != nil {
		return err
	}
	// query holds the ran query that has been ran and stored in the db
	var query []byte
	errQ := c.QueryRow("SELECT query FROM migrations WHERE name = ?",
		filename,
	).Scan(&query)
	if errQ != nil {
		// if the query file content has not been ran yet
		if errors.Is(errQ, sql.ErrNoRows) {
			// write file sql to db
			if err = c.updateMigDB(filename, sqlBytes); err != nil {
				return err
			}
			// move to next file
			return nil
		}
		return errQ // query error
	}
	queryStr := strings.TrimSpace(string(query))       // from db
	sqlBytesStr := strings.TrimSpace(string(sqlBytes)) // from file
	// if the query in db is not equal to the file content
	if !strings.EqualFold(queryStr, sqlBytesStr) {
		// Show diff between `query` in db and file content
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(queryStr, sqlBytesStr, false)
		fmt.Printf("Migration mismatch for %s\n", filename)
		fmt.Printf("diff: %s\n", dmp.DiffPrettyText(diffs))
		return fmt.Errorf("%w: %s", ErrMigContentChanged, filename)
	}
	// migration content has not changed, move to next file
	return nil
}

// RunOneMigration applies a single migration file into the database.
//
// c is the database client.
//
// filename is the name of the migration file to apply.
//
// dir is the path to the migration files.
//
// sep is the separator used in the migration file name. E.g "1_sep_2_sep_3.sql"
func (c *DBClient) RunOneMigration(filename, dir, sep string) error {
	if err := c.createMigTable(); err != nil {
		return err
	}
	if err := c.processMigFile(filename, dir, sep); err != nil {
		return err
	}
	return nil
}

// RunMigrations applies all migrations in the given directory into the database.
//
// c is the database client.
//
// dir is the path to the migration files.
//
// sep is the separator used in the migration file name. E.g "1_sep_2_sep_3.sql"
func (c *DBClient) RunMigrations(dir, sep string) error {
	if err := c.createMigTable(); err != nil {
		return err
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	// Iterate over each file in the directory
	// and ensure it is a valid migration file
	// and compare its content with the one in the database
	// and apply the migration if it hasn't been applied
	for _, f := range files {
		if err := c.processMigFile(f.Name(), dir, sep); err != nil {
			return err
		}
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
func (c *DBClient) RunMigrationsContext(ctx context.Context, dir, sep string) error {
	if err := c.createMigTable(); err != nil {
		return err
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	selectQ, err := c.PrepareContext(ctx, "SELECT query FROM migrations WHERE name = ?")
	if err != nil {
		return err
	}
	// Iterate over each file in the directory
	// and ensure it is a valid migration file
	// and compare its content with the one in the database
	// and apply the migration if it hasn't been applied
	for _, f := range files {
		if err := validateFilename(f.Name(), sep); err != nil {
			return err
		}
		sqlBytes, err := os.ReadFile(filepath.Join(dir, f.Name()))
		if err != nil {
			return err
		}
		var query []byte
		errQ := selectQ.QueryRowContext(
			ctx,
			f.Name(),
		).Scan(&query)
		if errQ != nil {
			if errors.Is(errQ, sql.ErrNoRows) {
				// Migration content not found, write to db
				if err = c.updateMigDB(f.Name(), sqlBytes); err != nil {
					return err
				}
				// move to next file
				continue
			}
			return errQ
		}

		// Compare the `query` in db with file content to avoid applying outdated migrations
		queryStr := strings.TrimSpace(string(query))       // from db
		sqlBytesStr := strings.TrimSpace(string(sqlBytes)) // from file
		if strings.EqualFold(queryStr, sqlBytesStr) {
			// Migration content not changed, move to next file
			continue
		}
		// Show diff between `query` in db and file content
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(string(query), string(sqlBytes), false)
		fmt.Printf("Migration mismatch for %s\n", f.Name())
		fmt.Printf("diff: %s\n", dmp.DiffPrettyText(diffs))
		return fmt.Errorf("%w: %s", ErrMigContentChanged, f.Name())
	}
	return nil
}

// ListMigrationsContext lists all migrations that have been applied to the database.
//
// c is the database client.
//
// ctx is the context.
//
// This function:
// - Creates the migrations table if it doesn't exist
//
// - Returns a list of all applied migration files
func (c *DBClient) ListMigrations() ([]string, error) {
	rows, err := c.ReadPool.Query(`SELECT name FROM migrations`)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			fmt.Printf("Row Close error: %v\n", err)
		}
	}()
	var migrations []string
	for rows.Next() {
		var migration string
		if err = rows.Scan(&migration); err != nil {
			return nil, err
		}
		migrations = append(migrations, migration)
	}
	return migrations, nil
}

// ListMigrationsContext lists all migrations that have been applied to the database.
//
// c is the database client.
//
// ctx is the context.
//
// This function:
// - Creates the migrations table if it doesn't exist
//
// - Returns a list of all applied migration files
func (c *DBClient) ListMigrationsContext(ctx context.Context) ([]string, error) {
	rows, err := c.ReadPool.QueryContext(ctx, `SELECT name FROM migrations`)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			fmt.Printf("Row Close error: %v\n", err)
		}
	}()
	var migrations []string
	for rows.Next() {
		var migration string
		if err = rows.Scan(&migration); err != nil {
			return nil, err
		}
		migrations = append(migrations, migration)
	}
	return migrations, nil
}

// Close closes the database connection.
func (c *DBClient) Close() error {
	if err := c.ReadPool.Close(); err != nil {
		return err
	}
	if err := c.WritePool.Close(); err != nil {
		return err
	}
	return nil
}

// DiskDB creates a new DBClient with the given path.
func DiskDB(path string) (*DBClient, error) {
	rDSN := createDSN(path, true, false)
	wDSN := createDSN(path, false, false)
	return setupPools(rDSN, wDSN)
}

func MemoryDB() (*DBClient, error) {
	path := "file:memdb"
	rDSN := createDSN(path, true, true)
	wDSN := createDSN(path, false, true)
	return setupPools(rDSN, wDSN)
}