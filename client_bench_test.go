package gosqlitex

import (
	"path/filepath"
	"strconv"
	"testing"
)

// BenchmarkParallelReads measures how fast ReadPool scales across CPU cores.
func BenchmarkParallelReads(b *testing.B) {
	b.Run("Reads concurrently", func(b *testing.B) {
		path := filepath.Join(b.TempDir(), d)
		client, err := DiskDB(path)
		if err != nil {
			b.Fatalf("failed to open database: %v", err)
		}
		defer client.Close()

		// Prep seed data before timing begins
		if _, err = client.Exec("CREATE TABLE benchmarks (id INTEGER PRIMARY KEY, val TEXT)"); err != nil {
			b.Fatalf("failed to create table: %v", err)
		}
		if _, err = client.Exec("INSERT INTO benchmarks (id, val) VALUES (1, 'some-cached-data')"); err != nil {
			b.Fatalf("failed to insert data: %v", err)
		}

		b.ResetTimer()

		// Scale reads concurrently across all available processor threads
		b.RunParallel(func(pb *testing.PB) {
			selectQ, err := client.Prepare("SELECT val FROM benchmarks WHERE id = 1")
			if err != nil {
				b.Errorf("prepare failed: %v", err)
			}
			for pb.Next() {
				var val string
				if err := selectQ.QueryRow().Scan(&val); err != nil {
					b.Errorf("read benchmark failed: %v", err)
				}
			}
		})
	})
	b.Run("Reads concurrently on memory", func(b *testing.B) {
		path := filepath.Join(b.TempDir(), d)
		client, err := DiskDB(path)
		if err != nil {
			b.Fatalf("failed to open database: %v", err)
		}
		defer client.Close()

		// Prep seed data before timing begins
		if _, err = client.Exec("CREATE TABLE benchmarks (id INTEGER PRIMARY KEY, val TEXT)"); err != nil {
			b.Fatalf("failed to create table: %v", err)
		}
		if _, err = client.Exec("INSERT INTO benchmarks (id, val) VALUES (1, 'some-cached-data')"); err != nil {
			b.Fatalf("failed to insert data: %v", err)
		}

		b.ResetTimer()

		// Scale reads concurrently across all available processor threads
		b.RunParallel(func(pb *testing.PB) {
			selectQ, err := client.Prepare("SELECT val FROM benchmarks WHERE id = 1")
			if err != nil {
				b.Errorf("prepare failed: %v", err)
			}
			for pb.Next() {
				var val string
				if err := selectQ.QueryRow().Scan(&val); err != nil {
					b.Errorf("read benchmark failed: %v", err)
				}
			}
		})
	})
}

// BenchmarkParallelWrites measures how Write-Ahead Logging handles concurrent writes.
func BenchmarkParallelWrites(b *testing.B) {
	b.Run("Writes concurrently", func(b *testing.B) {
		path := filepath.Join(b.TempDir(), d)
		client, err := DiskDB(path)
		if err != nil {
			b.Fatalf("failed to open database: %v", err)
		}
		defer client.Close()

		if _, err = client.Exec("CREATE TABLE benchmarks (id INTEGER PRIMARY KEY, val TEXT)"); err != nil {
			b.Fatalf("failed to create table: %v", err)
		}

		b.ResetTimer()

		// Run writes concurrently.
		// Because writeDBMaxConn = 1, Go's connection pool handles the queue,
		// while _busy_timeout prevents thread collisions from crashing the run.
		b.RunParallel(func(pb *testing.PB) {
			insertQ, err := client.Prepare("INSERT INTO benchmarks (val) VALUES (?)")
			if err != nil {
				b.Errorf("prepare failed: %v", err)
			}
			i := 0
			for pb.Next() {
				i++
				if _, err := insertQ.Exec("payload-" + strconv.Itoa(i)); err != nil {
					b.Errorf("write benchmark failed: %v", err)
				}
			}
		})
	})
	b.Run("Writes concurrently on memory", func(b *testing.B) {
		path := filepath.Join(b.TempDir(), d)
		client, err := DiskDB(path)
		if err != nil {
			b.Fatalf("failed to open database: %v", err)
		}
		defer client.Close()

		if _, err = client.Exec("CREATE TABLE benchmarks (id INTEGER PRIMARY KEY, val TEXT)"); err != nil {
			b.Fatalf("failed to create table: %v", err)
		}

		b.ResetTimer()

		// Run writes concurrently.
		// Because writeDBMaxConn = 1, Go's connection pool handles the queue,
		// while _busy_timeout prevents thread collisions from crashing the run.
		b.RunParallel(func(pb *testing.PB) {
			insertQ, err := client.Prepare("INSERT INTO benchmarks (val) VALUES (?)")
			if err != nil {
				b.Errorf("prepare failed: %v", err)
			}
			i := 0
			for pb.Next() {
				i++
				if _, err := insertQ.Exec("payload-" + strconv.Itoa(i)); err != nil {
					b.Errorf("write benchmark failed: %v", err)
				}
			}
		})
	})
}
