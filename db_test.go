package gosqlitex

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpen(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	cnf := &Config{
		DbPath: dbPath,
	}

	client, err := Open(cnf)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer client.Close()

	if err := client.Ping(); err != nil {
		t.Errorf("failed to ping database: %v", err)
	}
}

func TestExecAndQuery(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_exec.db")
	client, err := Open(&Config{DbPath: dbPath})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer client.Close()

	// Test Exec
	_, err = client.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	_, err = client.Exec("INSERT INTO users (name) VALUES (?)", "Alice")
	if err != nil {
		t.Fatalf("failed to insert data: %v", err)
	}

	// Test QueryRow
	var name string
	err = client.QueryRow("SELECT name FROM users WHERE id = ?", 1).Scan(&name)
	if err != nil {
		t.Fatalf("failed to query row: %v", err)
	}
	if name != "Alice" {
		t.Errorf("expected Alice, got %s", name)
	}

	// Test Query
	rows, err := client.Query("SELECT name FROM users")
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
}

func TestOpenWithDSN(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_dsn.db")
	// Manual DSN configuration
	dsn := "file:" + dbPath + "?mode=rwc&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"

	cnf := &Config{
		Driver: "sqlite",
		RDsn:   dsn,
		WDsn:   dsn,
	}

	client, err := Open(cnf)
	if err != nil {
		t.Fatalf("failed to open database with DSN: %v", err)
	}
	defer client.Close()

	if err := client.Ping(); err != nil {
		t.Errorf("failed to ping database: %v", err)
	}
}

func TestConcurrency(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_concurrency.db")
	client, err := Open(&Config{DbPath: dbPath})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer client.Close()

	_, err = client.Exec("CREATE TABLE counters (id INTEGER PRIMARY KEY, val INTEGER)")
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	_, err = client.Exec("INSERT INTO counters (val) VALUES (0)")
	if err != nil {
		t.Fatalf("failed to insert data: %v", err)
	}

	// Test concurrent reads while writing
	done := make(chan bool)
	go func() {
		for i := 0; i < 100; i++ {
			var val int
			err := client.QueryRow("SELECT val FROM counters WHERE id = 1").Scan(&val)
			if err != nil {
				t.Errorf("concurrent read failed: %v", err)
			}
		}
		done <- true
	}()

	for i := 0; i < 10; i++ {
		_, err = client.Exec("UPDATE counters SET val = val + 1 WHERE id = 1")
		if err != nil {
			t.Fatalf("write failed: %v", err)
		}
	}

	<-done
}

func TestTransactionsAndContext(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_tx_ctx.db")
	client, err := Open(&Config{DbPath: dbPath})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Test ExecContext
	_, err = client.ExecContext(ctx, "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		t.Fatalf("ExecContext failed: %v", err)
	}

	// Test Transaction (Begin)
	tx, err := client.Begin()
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}
	_, err = tx.Exec("INSERT INTO users (name) VALUES (?)", "Bob")
	if err != nil {
		tx.Rollback()
		t.Fatalf("transaction insert failed: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	// Test QueryRowContext
	var name string
	err = client.QueryRowContext(ctx, "SELECT name FROM users WHERE name = ?", "Bob").Scan(&name)
	if err != nil {
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
	_, err = tx.ExecContext(ctx, "INSERT INTO users (name) VALUES (?)", "Charlie")
	if err != nil {
		tx.Rollback()
		t.Fatalf("transaction insert with context failed: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit (BeginTx) failed: %v", err)
	}

	// Test QueryContext
	rows, err := client.QueryContext(ctx, "SELECT name FROM users ORDER BY name")
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
}

func TestPrepare(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_prepare.db")
	client, err := Open(&Config{DbPath: dbPath})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer client.Close()

	_, err = client.Exec("CREATE TABLE items (id INTEGER PRIMARY KEY, val TEXT)")
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	// Test Prepare (Write)
	stmt, err := client.Prepare("INSERT INTO items (val) VALUES (?)")
	if err != nil {
		t.Fatalf("Prepare (Write) failed: %v", err)
	}
	_, err = stmt.Exec("item1")
	if err != nil {
		t.Fatalf("stmt.Exec failed: %v", err)
	}
	stmt.Close()

	// Test Prepare (Read)
	stmt, err = client.Prepare("SELECT val FROM items WHERE id = ?")
	if err != nil {
		t.Fatalf("Prepare (Read) failed: %v", err)
	}
	var val string
	err = stmt.QueryRow(1).Scan(&val)
	if err != nil {
		t.Fatalf("stmt.QueryRow failed: %v", err)
	}
	if val != "item1" {
		t.Errorf("expected item1, got %s", val)
	}
	stmt.Close()

	// Test Prepare (Read with lowercase and whitespace)
	stmt, err = client.Prepare("  select val from items where id = ?")
	if err != nil {
		t.Fatalf("Prepare (Read lowercase) failed: %v", err)
	}
	err = stmt.QueryRow(1).Scan(&val)
	if err != nil {
		t.Fatalf("stmt.QueryRow (lowercase) failed: %v", err)
	}
	if val != "item1" {
		t.Errorf("expected item1, got %s", val)
	}
	stmt.Close()

	// Test PrepareContext
	ctx := context.Background()
	stmt, err = client.PrepareContext(ctx, "SELECT val FROM items WHERE id = ?")
	if err != nil {
		t.Fatalf("PrepareContext failed: %v", err)
	}
	err = stmt.QueryRowContext(ctx, 1).Scan(&val)
	if err != nil {
		t.Fatalf("stmt.QueryRowContext failed: %v", err)
	}
	if val != "item1" {
		t.Errorf("expected item1, got %s", val)
	}
	stmt.Close()
}

func TestClose(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_close.db")
	client, err := Open(&Config{DbPath: dbPath})
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
}

func TestMigrations(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_mig.db")
	client, err := Open(&Config{DbPath: dbPath})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	migDir := t.TempDir()
	sep := "_"

	// 1. Create a valid migration file
	mig1 := "20230101000000" + sep + "init.sql"
	sql1 := "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT);"
	err = os.WriteFile(filepath.Join(migDir, mig1), []byte(sql1), 0644)
	if err != nil {
		t.Fatalf("failed to write mig1: %v", err)
	}

	// 2. Run migrations
	err = client.RunMigrationsContext(ctx, migDir, sep)
	if err != nil {
		t.Fatalf("RunMigrationsContext failed: %v", err)
	}

	// Verify table exists
	_, err = client.Exec("INSERT INTO users (name) VALUES (?)", "Alice")
	if err != nil {
		t.Errorf("table users does not exist or insert failed: %v", err)
	}

	// 3. List migrations
	migs, err := client.ListMigrationsContext(ctx)
	if err != nil {
		t.Fatalf("ListMigrationsContext failed: %v", err)
	}
	if len(migs) != 1 || migs[0] != mig1 {
		t.Errorf("unexpected migrations list: %v", migs)
	}

	// 4. Run again (should skip)
	err = client.RunMigrationsContext(ctx, migDir, sep)
	if err != nil {
		t.Fatalf("RunMigrationsContext (second run) failed: %v", err)
	}

	// 5. Add second migration
	mig2 := "20230101000001" + sep + "add_posts.sql"
	sql2 := "CREATE TABLE posts (id INTEGER PRIMARY KEY, title TEXT);"
	err = os.WriteFile(filepath.Join(migDir, mig2), []byte(sql2), 0644)
	if err != nil {
		t.Fatalf("failed to write mig2: %v", err)
	}

	err = client.RunMigrationsContext(ctx, migDir, sep)
	if err != nil {
		t.Fatalf("RunMigrationsContext (third run) failed: %v", err)
	}

	migs, err = client.ListMigrationsContext(ctx)
	if err != nil {
		t.Fatalf("ListMigrationsContext failed: %v", err)
	}
	if len(migs) != 2 {
		t.Errorf("expected 2 migrations, got %d", len(migs))
	}

	// 6. Test content change error
	err = os.WriteFile(filepath.Join(migDir, mig1), []byte("CHANGED"), 0644)
	if err != nil {
		t.Fatalf("failed to change mig1: %v", err)
	}
	err = client.RunMigrationsContext(ctx, migDir, sep)
	if err == nil || !strings.Contains(err.Error(), "migration content changed") {
		t.Errorf("expected content changed error, got %v", err)
	}
}

func TestInvalidMigrations(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_invalid_mig.db")
	client, err := Open(&Config{DbPath: dbPath})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	migDir := t.TempDir()
	sep := "_"

	// Invalid prefix
	err = os.WriteFile(filepath.Join(migDir, "abc_init.sql"), []byte("..."), 0644)
	if err != nil {
		t.Fatalf("failed to write invalid prefix file: %v", err)
	}
	err = client.RunMigrationsContext(ctx, migDir, sep)
	if err == nil || !strings.Contains(err.Error(), "migration file name prefix is not a number") {
		t.Errorf("expected invalid prefix error, got %v", err)
	}

	// Missing separator
	migDir2 := t.TempDir()
	err = os.WriteFile(filepath.Join(migDir2, "20230101000000init.sql"), []byte("..."), 0644)
	if err != nil {
		t.Fatalf("failed to write missing separator file: %v", err)
	}
	err = client.RunMigrationsContext(ctx, migDir2, sep)
	if err == nil || !strings.Contains(err.Error(), "migration file name separator not found") {
		t.Errorf("expected missing separator error, got %v", err)
	}
}

