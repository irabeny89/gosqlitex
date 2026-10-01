package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func Test_requiredArgs(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		args           *ParsedArgs
		requiredFields []string
		want           bool
	}{
		{
			name:           "no args, no required fields",
			args:           nil,
			requiredFields: nil,
			want:           false,
		},
		{
			name:           "args provided, no required fields",
			args:           &ParsedArgs{},
			requiredFields: nil,
			want:           false,
		},
		{
			name: "args provided contains field with zero value, required fields provided",
			args: &ParsedArgs{
				dir:   "",
				db:    "",
				title: "",
				file:  "",
				sep:   "",
				run:   false,
				list:  false,
			},
			requiredFields: []string{"dir"},
			want:           false,
		},
		{
			name: "args provided, required fields provided",
			args: &ParsedArgs{
				dir:   "./",
				db:    "",
				title: "",
				file:  "",
				sep:   "",
				run:   false,
				list:  false,
			},
			requiredFields: []string{"dir"},
			want:           true,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := requiredArgs(tt.args, tt.requiredFields...)
			if got != tt.want {
				t.Errorf("requiredArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_CLI(t *testing.T) {
	dBPath := func(dir string, disk bool) string {
		if !disk {
			// Sanitize the base name
			safeName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
			// Append a Unix timestamp (nanoseconds) to ensure a truly unique name
			// for every individual execution run.
			uniqueID := strconv.FormatInt(time.Now().UnixNano(), 10)
			return ":memory:" + safeName + "_" + uniqueID
		}
		dbPath := filepath.Join(dir, "test.db")
		return filepath.ToSlash(dbPath)
	}

	titles := []string{"test migration file"}
	dir := t.TempDir()
	migDir := filepath.Join(dir, "migrations")
	dsn := dBPath(dir, true)
	errChan := make(chan error, 2)
	var wg sync.WaitGroup
	file := ""

	buildBin := func(dir string, errChan chan error) {
		cmd := exec.Command("go", "build", "-o", dir, "mig8.go")
		if err := cmd.Run(); err != nil {
			errChan <- err
		}
		if err := os.Chmod(dir, 0755); err != nil {
			errChan <- err
		}
	}
	createMig := func(dir string, errChan chan error) {
		createQ := "CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, name TEXT);"
		migDir := filepath.Join(dir, "migrations")
		f, err := generateFile(migDir, "create user", sep)
		if err != nil {
			errChan <- err
			return
		}
		defer f.Close()
		file = f.Name()
		if err := os.WriteFile(f.Name(), []byte(createQ), 0744); err != nil {
			errChan <- err
		}
	}

	wg.Go(func() { buildBin(dir, errChan) })
	wg.Go(func() { createMig(dir, errChan) })
	wg.Wait()
	close(errChan)
	for err := range errChan {
		t.Errorf("error in goroutine: %v", err)
	}

	tests := []struct {
		name    string
		args    []string
		wantErr bool
		want    string
	}{
		{
			name:    "no arguments, print usage",
			args:    []string{""},
			wantErr: false,
			want:    "Wrong usage",
		},
		{
			name:    "generate migration file",
			args:    []string{"-dir", migDir, "-title", titles[0]},
			wantErr: false,
			want:    strings.ReplaceAll(titles[0], " ", sep),
		},
		{
			name:    "run all migrations",
			args:    []string{"-dir", migDir, "-db", dsn},
			wantErr: false,
			want:    "Migrations ran successfully",
		},
		{
			name:    "run a single migration",
			args:    []string{"-dir", migDir, "-db", dsn, "file", file},
			wantErr: false,
			want:    "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			binPath := filepath.Join(dir, "mig8")
			cmd := exec.Command(binPath, tt.args...)
			out, err := cmd.Output()
			if err != nil {
				if tt.wantErr {
					return
				}
				t.Errorf("expect no error: %v", err)
			}
			outStr := string(out)
			if !strings.Contains(outStr, tt.want) {
				t.Errorf("expect: %s, got: %s", tt.want, outStr)
			}
		})
	}
}
