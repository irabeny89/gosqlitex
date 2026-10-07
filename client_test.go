package gosqlitex

import (
	"context"
	"fmt"
	"sync"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

const d = "test.db"

func TestOpen(t *testing.T) {
	t.Run("expect db to open and ping successfully", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), d)
		client, err := DiskDB(path)

		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}
		defer client.Close()
		if err := client.Ping(); err != nil {
			t.Errorf("failed to ping database: %v", err)
		}
	})
	t.Run("expect db to open and ping successfully on memory", func(t *testing.T) {
		client, err := MemoryDB()

		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}
		defer client.Close()
		if err := client.Ping(); err != nil {
			t.Errorf("failed to ping database: %v", err)
		}
	})
}

func TestExecAndQuery(t *testing.T) {
	createQ := "CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, name TEXT)"
	insertQ := "INSERT INTO users (name) VALUES (?)"
	selectQ := "SELECT name FROM users WHERE id = ?"
	selectAllQ := "SELECT name FROM users"

	t.Run("expect exec and query to work", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), d)
		client, err := DiskDB(path)
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}
		defer client.Close()

		// Test Exec
		if _, err := client.Exec(createQ); err != nil {
			t.Fatalf("failed to create table: %v", err)
		}
		if _, err := client.Exec(insertQ, "Alice"); err != nil {
			t.Fatalf("failed to insert data: %v", err)
		}

		// Test QueryRow
		var name string
		if err := client.QueryRow(selectQ, 1).Scan(&name); err != nil {
			t.Fatalf("failed to query row: %v", err)
		}
		if name != "Alice" {
			t.Errorf("expected Alice, got %s", name)
		}

		// Test Query
		rows, err := client.Query(selectAllQ)
		if err != nil {
			t.Fatalf("failed to query rows: %v", err)
		}
		defer rows.Close()

		count := 0
		for rows.Next() {
			count++
			var n string
			if err := rows.Scan(&n); err != nil {
				t.Errorf("failed to scan row: %v", err)
			}
		}
		if count != 1 {
			t.Errorf("expected 1 row, got %d", count)
		}
	})
	t.Run("expect exec and query to work on memory", func(t *testing.T) {
		client, err := MemoryDB()
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}
		defer client.Close()

		// Test Exec
		if _, err := client.Exec(createQ); err != nil {
			t.Fatalf("failed to create table: %v", err)
		}
		if _, err := client.Exec(insertQ, "Alice"); err != nil {
			t.Fatalf("failed to insert data: %v", err)
		}

		// Test QueryRow
		var name string
		if err := client.QueryRow(selectQ, 1).Scan(&name); err != nil {
			t.Fatalf("failed to query row: %v", err)
		}
		if name != "Alice" {
			t.Errorf("expected Alice, got %s", name)
		}

		// Test Query
		rows, err := client.Query(selectAllQ)
		if err != nil {
			t.Fatalf("failed to query rows: %v", err)
		}
		defer rows.Close()

		count := 0
		for rows.Next() {
			count++
			var n string
			if err := rows.Scan(&n); err != nil {
				t.Errorf("failed to scan row: %v", err)
			}
		}
		if count != 1 {
			t.Errorf("expected 1 row, got %d", count)
		}
	})
}

func TestConcurrency(t *testing.T) {
	createQ := "CREATE TABLE counters (id INTEGER PRIMARY KEY, val INTEGER)"
	insertQ := "INSERT INTO counters (id, val) VALUES (1, 0)"
	selectQ := "SELECT val FROM counters WHERE id = 1"
	updateQ := "UPDATE counters SET val = val + 1 WHERE id = 1"

	t.Run("expect concurrent reads while writing", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), d)
		client, err := DiskDB(path)
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}
		defer client.Close()

		// Setup initial structural data
		if _, err = client.Exec(createQ); err != nil {
			t.Fatalf("failed to create table: %v", err)
		}
		if _, err = client.Exec(insertQ); err != nil {
			t.Fatalf("failed to insert data: %v", err)
		}

		var wg sync.WaitGroup
		errChan := make(chan error, 110) // Buffer to hold errors from background threads
		// Spawn 100 concurrent READ operations using the ReadPool
		for i := 0; i < 100; i++ {
			wg.Go(func() {
				var val int
				err := client.QueryRow(selectQ).Scan(&val)
				if err != nil {
					errChan <- fmt.Errorf("concurrent read failed: %w", err)
				}
			})
		}
		// Spawn 10 concurrent WRITE operations using the WritePool
		for i := 0; i < 10; i++ {
			wg.Go(func() {
				// Exec routes cleanly to client.WritePool
				_, err := client.Exec(updateQ)
				if err != nil {
					errChan <- fmt.Errorf("concurrent write failed: %w", err)
				}
			})
		}
		// Wait for all concurrent routines to cross the finish line
		wg.Wait()
		close(errChan)
		// Report any thread errors back to Go's test reporter
		for err := range errChan {
			t.Error(err)
		}
		// Validate the final state to confirm no writes were dropped
		var finalVal int
		err = client.QueryRow(selectQ).Scan(&finalVal)
		if err != nil {
			t.Fatalf("failed to read final count: %v", err)
		}
		if finalVal != 10 {
			t.Errorf("expected final value to be 10, got %d", finalVal)
		}
	})
	t.Run("expect concurrent reads while writing on memory", func(t *testing.T) {
		client, err := MemoryDB()
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}
		defer client.Close()

		// 1. Setup initial structural data
		if _, err = client.Exec(createQ); err != nil {
			t.Fatalf("failed to create table: %v", err)
		}
		if _, err = client.Exec(insertQ); err != nil {
			t.Fatalf("failed to insert data: %v", err)
		}

		var wg sync.WaitGroup
		errChan := make(chan error, 110) // Buffer to hold errors from background threads
		// 2. Spawn 100 concurrent READ operations using the ReadPool
		for i := 0; i < 100; i++ {
			wg.Go(func() {
				var val int
				err := client.QueryRow(selectQ).Scan(&val)
				if err != nil {
					errChan <- fmt.Errorf("concurrent read failed: %w", err)
				}
			})
		}
		// 3. Spawn 10 concurrent WRITE operations using the WritePool
		for i := 0; i < 10; i++ {
			wg.Go(func() {
				// Exec routes cleanly to client.WritePool
				_, err := client.Exec(updateQ)
				if err != nil {
					errChan <- fmt.Errorf("concurrent write failed: %w", err)
				}
			})
		}
		// Wait for all concurrent routines to cross the finish line
		wg.Wait()
		close(errChan)
		// 4. Report any thread errors back to Go's test reporter
		for err := range errChan {
			t.Error(err)
		}
		// 5. Validate the final state to confirm no writes were dropped
		var finalVal int
		err = client.QueryRow(selectQ).Scan(&finalVal)
		if err != nil {
			t.Fatalf("failed to read final count: %v", err)
		}
		if finalVal != 10 {
			t.Errorf("expected final value to be 10, got %d", finalVal)
		}
	})
}

func TestTransactionsAndContext(t *testing.T) {
	createQ := "CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, name TEXT)"
	insertQ := "INSERT INTO users (name) VALUES (?)"
	selectQ := "SELECT name FROM users WHERE name = ?"
	selectOQ := "SELECT name FROM users ORDER BY name"
	t.Run("expect transactions and context to work", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), d)
		client, err := DiskDB(path)
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}
		defer client.Close()

		ctx := context.Background()
		// Test ExecContext
		if _, err = client.ExecContext(ctx, createQ); err != nil {
			t.Fatalf("ExecContext failed: %v", err)
		}

		// Test Transaction (Begin)
		tx, err := client.Begin()
		if err != nil {
			t.Fatalf("Begin failed: %v", err)
		}
		if _, err = tx.Exec(insertQ, "Bob"); err != nil {
			tx.Rollback()
			t.Fatalf("transaction insert failed: %v", err)
		}
		if err := tx.Commit(); err != nil {
			t.Fatalf("Commit failed: %v", err)
		}

		// Test QueryRowContext
		var name string
		if err = client.QueryRowContext(ctx, selectQ, "Bob").Scan(&name); err != nil {
			t.Fatalf("QueryRowContext failed: %v", err)
		}
		if name != "Bob" {
			t.Errorf("expected Bob, got %s", name)
		}

		// Test Transaction with Context (BeginTx)
		tx, err = client.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("BeginTx failed: %v", err)
		}
		if _, err = tx.ExecContext(ctx, insertQ, "Charlie"); err != nil {
			tx.Rollback()
			t.Fatalf("transaction insert with context failed: %v", err)
		}
		if err := tx.Commit(); err != nil {
			t.Fatalf("Commit (BeginTx) failed: %v", err)
		}

		// Test QueryContext
		rows, err := client.QueryContext(ctx, selectOQ)
		if err != nil {
			t.Fatalf("QueryContext failed: %v", err)
		}
		defer rows.Close()

		var names []string
		for rows.Next() {
			var n string
			if err := rows.Scan(&n); err != nil {
				t.Fatalf("rows.Scan failed: %v", err)
			}
			names = append(names, n)
		}
		if len(names) != 2 || names[0] != "Bob" || names[1] != "Charlie" {
			t.Errorf("unexpected results: %v", names)
		}
	})
	t.Run("expect transactions and context to work on memory", func(t *testing.T) {
		client, err := MemoryDB()
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}
		defer client.Close()

		ctx := context.Background()
		// Test ExecContext
		if _, err = client.ExecContext(ctx, createQ); err != nil {
			t.Fatalf("ExecContext failed: %v", err)
		}

		// Test Transaction (Begin)
		tx, err := client.Begin()
		if err != nil {
			t.Fatalf("Begin failed: %v", err)
		}
		if _, err = tx.Exec(insertQ, "Bob"); err != nil {
			tx.Rollback()
			t.Fatalf("transaction insert failed: %v", err)
		}
		if err := tx.Commit(); err != nil {
			t.Fatalf("Commit failed: %v", err)
		}

		// Test QueryRowContext
		var name string
		if err = client.QueryRowContext(ctx, selectQ, "Bob").Scan(&name); err != nil {
			t.Fatalf("QueryRowContext failed: %v", err)
		}
		if name != "Bob" {
			t.Errorf("expected Bob, got %s", name)
		}

		// Test Transaction with Context (BeginTx)
		tx, err = client.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("BeginTx failed: %v", err)
		}
		if _, err = tx.ExecContext(ctx, insertQ, "Charlie"); err != nil {
			tx.Rollback()
			t.Fatalf("transaction insert with context failed: %v", err)
		}
		if err := tx.Commit(); err != nil {
			t.Fatalf("Commit (BeginTx) failed: %v", err)
		}

		// Test QueryContext
		rows, err := client.QueryContext(ctx, selectOQ)
		if err != nil {
			t.Fatalf("QueryContext failed: %v", err)
		}
		defer rows.Close()

		var names []string
		for rows.Next() {
			var n string
			if err := rows.Scan(&n); err != nil {
				t.Fatalf("rows.Scan failed: %v", err)
			}
			names = append(names, n)
		}
		if len(names) != 2 || names[0] != "Bob" || names[1] != "Charlie" {
			t.Errorf("unexpected results: %v", names)
		}
	})
}

func TestPrepare(t *testing.T) {
	createQ := "CREATE TABLE items (id INTEGER PRIMARY KEY, val TEXT)"
	insertQ := "INSERT INTO items (val) VALUES (?)"
	selectQ := "SELECT val FROM items WHERE id = ?"
	selectOQ := "SELECT val FROM items"

	t.Run("expect Prepare to work", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), d)
		client, err := DiskDB(path)
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}
		defer client.Close()

		if _, err = client.Exec(createQ); err != nil {
			t.Fatalf("failed to create table: %v", err)
		}

		// Test Prepare (Write)
		stmt, err := client.Prepare(insertQ)
		if err != nil {
			t.Fatalf("Prepare (Write) failed: %v", err)
		}
		if _, err = stmt.Exec("item1"); err != nil {
			t.Fatalf("stmt.Exec failed: %v", err)
		}
		stmt.Close()

		// Test Prepare (Read)
		stmt, err = client.Prepare(selectQ)
		if err != nil {
			t.Fatalf("Prepare (Read) failed: %v", err)
		}
		var val string
		if err = stmt.QueryRow(1).Scan(&val); err != nil {
			t.Fatalf("stmt.QueryRow failed: %v", err)
		}
		if val != "item1" {
			t.Errorf("expected item1, got %s", val)
		}
		stmt.Close()

		// Test Prepare (Read with lowercase and whitespace)
		stmt, err = client.Prepare(selectOQ)
		if err != nil {
			t.Fatalf("Prepare (Read lowercase) failed: %v", err)
		}
		if err = stmt.QueryRow(1).Scan(&val); err != nil {
			t.Fatalf("stmt.QueryRow (lowercase) failed: %v", err)
		}
		if val != "item1" {
			t.Errorf("expected item1, got %s", val)
		}
		stmt.Close()

		// Test PrepareContext
		ctx := context.Background()
		stmt, err = client.PrepareContext(ctx, selectQ)
		if err != nil {
			t.Fatalf("PrepareContext failed: %v", err)
		}
		if err = stmt.QueryRowContext(ctx, 1).Scan(&val); err != nil {
			t.Fatalf("stmt.QueryRowContext failed: %v", err)
		}
		if val != "item1" {
			t.Errorf("expected item1, got %s", val)
		}
		stmt.Close()
	})
	t.Run("expect Prepare to work on memory", func(t *testing.T) {
		client, err := MemoryDB()
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}
		defer client.Close()

		if _, err = client.Exec(createQ); err != nil {
			t.Fatalf("failed to create table: %v", err)
		}

		// Test Prepare (Write)
		stmt, err := client.Prepare(insertQ)
		if err != nil {
			t.Fatalf("Prepare (Write) failed: %v", err)
		}
		if _, err = stmt.Exec("item1"); err != nil {
			t.Fatalf("stmt.Exec failed: %v", err)
		}
		stmt.Close()

		// Test Prepare (Read)
		stmt, err = client.Prepare(selectQ)
		if err != nil {
			t.Fatalf("Prepare (Read) failed: %v", err)
		}
		var val string
		if err = stmt.QueryRow(1).Scan(&val); err != nil {
			t.Fatalf("stmt.QueryRow failed: %v", err)
		}
		if val != "item1" {
			t.Errorf("expected item1, got %s", val)
		}
		stmt.Close()

		// Test Prepare (Read with lowercase and whitespace)
		stmt, err = client.Prepare(selectOQ)
		if err != nil {
			t.Fatalf("Prepare (Read lowercase) failed: %v", err)
		}
		if err = stmt.QueryRow(1).Scan(&val); err != nil {
			t.Fatalf("stmt.QueryRow (lowercase) failed: %v", err)
		}
		if val != "item1" {
			t.Errorf("expected item1, got %s", val)
		}
		stmt.Close()

		// Test PrepareContext
		ctx := context.Background()
		stmt, err = client.PrepareContext(ctx, selectQ)
		if err != nil {
			t.Fatalf("PrepareContext failed: %v", err)
		}
		if err = stmt.QueryRowContext(ctx, 1).Scan(&val); err != nil {
			t.Fatalf("stmt.QueryRowContext failed: %v", err)
		}
		if val != "item1" {
			t.Errorf("expected item1, got %s", val)
		}
		stmt.Close()
	})
}

func TestClose(t *testing.T) {
	t.Run("expect close to work", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), d)
		client, err := DiskDB(path)
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}
		if err := client.Close(); err != nil {
			t.Errorf("Close failed: %v", err)
		}
		// Ping should fail after close
		if err := client.Ping(); err == nil {
			t.Error("Ping succeeded after Close, expected error")
		}
	})
	t.Run("expect close to work on memory", func(t *testing.T) {
		client, err := MemoryDB()
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}
		if err := client.Close(); err != nil {
			t.Errorf("Close failed: %v", err)
		}
		// Ping should fail after close
		if err := client.Ping(); err == nil {
			t.Error("Ping succeeded after Close, expected error")
		}
	})
}

func TestMigrations(t *testing.T) {
	t.Run("expect migrations to work", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), d)
		client, err := DiskDB(path)
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}
		defer client.Close()

		ctx := context.Background()
		migDir := t.TempDir()
		sep := "_"

		// Create a valid migration file
		mig1 := fmt.Sprintf("20230101000000%sinit.sql", sep)
		sql1 := "CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, name TEXT);"
		if err = os.WriteFile(filepath.Join(migDir, mig1), []byte(sql1), 0644); err != nil {
			t.Fatalf("failed to write mig1: %v", err)
		}
		// Run migrations
		if err = client.RunMigrationsContext(ctx, migDir, sep); err != nil {
			t.Fatalf("RunMigrationsContext failed: %v", err)
		}
		// Verify table exists
		if _, err = client.Exec("INSERT INTO users (name) VALUES (?)", "Alice"); err != nil {
			t.Errorf("table users does not exist or insert failed: %v", err)
		}
		// List migrations
		migs, err := client.ListMigrationsContext(ctx)
		if err != nil {
			t.Fatalf("ListMigrationsContext failed: %v", err)
		}
		if len(migs) != 1 || migs[0] != mig1 {
			t.Errorf("unexpected migrations list: %v", migs)
		}
		// Run again (should skip)
		if err = client.RunMigrationsContext(ctx, migDir, sep); err != nil {
			t.Fatalf("RunMigrationsContext (second run) failed: %v", err)
		}
		// Add second migration
		mig2 := fmt.Sprintf("20230101000001%sadd_posts.sql", sep)
		sql2 := "CREATE TABLE posts (id INTEGER PRIMARY KEY, title TEXT);"
		if err = os.WriteFile(filepath.Join(migDir, mig2), []byte(sql2), 0644); err != nil {
			t.Fatalf("failed to write mig2: %v", err)
		}
		if err = client.RunMigrationsContext(ctx, migDir, sep); err != nil {
			t.Fatalf("RunMigrationsContext (third run) failed: %v", err)
		}

		migs, err = client.ListMigrationsContext(ctx)
		if err != nil {
			t.Fatalf("ListMigrationsContext failed: %v", err)
		}
		if len(migs) != 2 {
			t.Errorf("expected 2 migrations, got %d", len(migs))
		}
		// Test content change error
		if err = os.WriteFile(filepath.Join(migDir, mig1), []byte("CHANGED"), 0644); err != nil {
			t.Fatalf("failed to change mig1: %v", err)
		}
		if err = client.RunMigrationsContext(ctx, migDir, sep); err == nil || !strings.Contains(err.Error(), "migration content changed") {
			t.Errorf("expected content changed error, got %v", err)
		}
	})
}

func TestInvalidMigrations(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_invalid_mig.db")
	client, err := DiskDB(dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	migDir := t.TempDir()
	sep := "_"

	// Invalid prefix
	if err = os.WriteFile(filepath.Join(migDir, "abc_init.sql"), []byte("..."), 0644); err != nil {
		t.Fatalf("failed to write invalid prefix file: %v", err)
	}
	if err = client.RunMigrationsContext(ctx, migDir, sep); err == nil || !strings.Contains(err.Error(), "migration file name prefix is not a number") {
		t.Errorf("expected invalid prefix error, got %v", err)
	}

	// Missing separator
	migDir2 := t.TempDir()
	if err = os.WriteFile(filepath.Join(migDir2, "20230101000000init.sql"), []byte("..."), 0644); err != nil {
		t.Fatalf("failed to write missing separator file: %v", err)
	}
	if err = client.RunMigrationsContext(ctx, migDir2, sep); err == nil || !strings.Contains(err.Error(), "migration file name separator not found") {
		t.Errorf("expected missing separator error, got %v", err)
	}
}
