package gosqlitex

import (
	"path/filepath"
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
	defer client.readPool.Close()
	defer client.writePool.Close()

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
	defer client.readPool.Close()
	defer client.writePool.Close()

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
	defer client.readPool.Close()
	defer client.writePool.Close()

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
	defer client.readPool.Close()
	defer client.writePool.Close()

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
