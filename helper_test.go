package gosqlitex

import (
	"fmt"
	"testing"
)

func Test_parseDSN(t *testing.T) {
	dsns := []string{
		"./test.db",
		":memory:",
	}
	modes := []string{
		"rwc",
		"ro",
	}
	
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		Dsn     string
		mode    string
		pragma  []string
		want    string
		wantErr bool
	}{
		{
			name:   "Disk storage with relative path and no pragma",
			Dsn:    dsns[0],
			mode:   modes[0],
			pragma: nil,
			want:   fmt.Sprintf("file://%s?mode=%s", dsns[0], modes[0]),
			wantErr: false,
		},
		{
			name:   "Memory storage with relative path and no pragma",
			Dsn:    dsns[1],
			mode:   modes[0],
			pragma: nil,
			want:   fmt.Sprintf("file:%s?cache=shared&mode=%s", memDBName, modes[0]),
			wantErr: false,
		},
		{
			name:   "Set default rwc mode on empty mode string value",
			Dsn:    dsns[0],
			mode:   "",
			pragma: nil,
			want:   fmt.Sprintf("file://%s?mode=%s", dsns[0], modes[0]),
			wantErr: false,
		},
		{
			name:   "Error on empty Dsn",
			Dsn:    "",
			mode:   "",
			pragma: nil,
			want:   "",
			wantErr: true,
		},
		{
			name:   "Error if Dsn is prefixed with file://",
			Dsn:    "file://",
			mode:   "",
			pragma: nil,
			want:   "",
			wantErr: true,
		},
		{
			name:   "Error if Dsn is suffixed with ?",
			Dsn:    "test.db?mode=rwc",
			mode:   "",
			pragma: nil,
			want:   "",
			wantErr: true,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := parseDSN(tt.Dsn, tt.mode, tt.pragma)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("parseDSN() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("parseDSN() succeeded unexpectedly")
			}
			// TODO: update the condition below to compare got with tt.want.
			if got != tt.want {
				t.Errorf("parseDSN() = %v, want %v", got, tt.want)
			}
		})
	}
}