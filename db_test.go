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
