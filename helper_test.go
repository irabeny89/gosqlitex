package gosqlitex

import (
	"testing"
)

func Test_validateFilename(t *testing.T) {
	tests := []struct {
		title string // description of this test case
		// Named input parameters for target function.
		name    string
		sep     string
		wantErr bool
	}{
		{
			title:   "valid filename",
			name:    "11111111_create_table.sql",
			sep:     "_",
			wantErr: false,
		},
		{
			title:   "invalid filename",
			name:    "test.sql",
			sep:     "_",
			wantErr: true,
		},
		{
			title:   "valid filename, incorrect separator",
			name:    "11111111_create_table",
			sep:     "-",
			wantErr: true,
		},
		{
			title:   "invalid filename, correct separator",
			name:    "create_table.sql",
			sep:     "-",
			wantErr: true,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := validateFilename(tt.name, tt.sep)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("validateFilename() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("validateFilename() succeeded unexpectedly")
			}
		})
	}
}

func Test_CreateDSN(t *testing.T) {
	diskPath, memPath := "test.db", "file:memdb"
	rQuery := "?_pragma=journal_mode%28WAL%29&_pragma=busy_timeout%285000%29&_pragma=foreign_keys%28ON%29&_pragma=cache_size%28-64000%29&_pragma=temp_store%28MEMORY%29&_pragma=mmap_size%28268435456%29&_pragma=synchronous%28NORMAL%29&mode=ro"
	wQuery := "?_pragma=journal_mode%28WAL%29&_pragma=busy_timeout%285000%29&_pragma=foreign_keys%28ON%29&_pragma=cache_size%28-64000%29&_pragma=temp_store%28MEMORY%29&_pragma=mmap_size%28268435456%29&_pragma=synchronous%28NORMAL%29&mode=rwc"
	rMemQuery := "?_pragma=journal_mode%28WAL%29&_pragma=busy_timeout%285000%29&_pragma=foreign_keys%28ON%29&_pragma=cache_size%28-64000%29&_pragma=temp_store%28MEMORY%29&_pragma=mmap_size%28268435456%29&_pragma=synchronous%28NORMAL%29&_pragma=query_only%28ON%29&cache=shared&mode=memory"
	wMemQuery := "?_pragma=journal_mode%28WAL%29&_pragma=busy_timeout%285000%29&_pragma=foreign_keys%28ON%29&_pragma=cache_size%28-64000%29&_pragma=temp_store%28MEMORY%29&_pragma=mmap_size%28268435456%29&_pragma=synchronous%28NORMAL%29&cache=shared&mode=memory"
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		path     string
		isRead   bool
		isMemory bool
		want     string
	}{
		{
			name:     "disk read dsn",
			path:     diskPath,
			isRead:   true,
			isMemory: false,
			want:     diskPath + rQuery,
		},
		{
			name:     "disk write dsn",
			path:     diskPath,
			isRead:   false,
			isMemory: false,
			want:     diskPath + wQuery,
		},
		{
			name:     "memory read dsn",
			path:     memPath,
			isRead:   true,
			isMemory: true,
			want:     memPath + rMemQuery,
		},
		{
			name:     "memory write dsn",
			path:     memPath,
			isRead:   false,
			isMemory: true,
			want:     memPath + wMemQuery,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CreateDSN(tt.path, tt.isRead, tt.isMemory)
			if got != tt.want {
				t.Errorf("CreateDSN() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_setupPools(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		rDSN    string
		wDSN    string
		wantErr bool
	}{
		{
			name:    "setupPools success for disk",
			rDSN:    "test.db",
			wDSN:    "test.db",
			wantErr: false,
		},
		{
			name:    "setupPools success for memory",
			rDSN:    "file:memdb?mode=memory",
			wDSN:    "file:memdb?mode=memory",
			wantErr: false,
		},

		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := setupPools(tt.rDSN, tt.wDSN)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("setupPools() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("setupPools() succeeded unexpectedly")
			}
			wErr, rErr := got.WritePool.Ping(), got.ReadPool.Ping()
			if wErr != nil {
				t.Errorf("expect write pool to ping, got %v", wErr)
			}
			if rErr != nil {
				t.Errorf("expect read pool to ping, got %v", rErr)
			}
		})
	}
}
