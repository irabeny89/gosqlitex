package gosqlitex_test

import(
	"database/sql"
	"github.com/irabeny89/gosqlitex"
	"testing"
)

func TestDBPool(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		Dsn     string
		maxConn int
		want    *sql.DB
		wantErr bool
	}{
		{
			name:   "Successfully creates a DBPool",
			Dsn:    "test.db",
			maxConn: 8,
			want:    nil,
			wantErr: false,
		},
		{
			name:   "Fail to create a DBPool if Dsn is empty string",
			Dsn:    "",
			maxConn: 8,
			want:    nil,
			wantErr: true,
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := gosqlitex.DBPool(tt.Dsn, tt.maxConn)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("DBPool() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("DBPool() succeeded unexpectedly")
			}
			// TODO: update the condition below to compare got with tt.want.
			if got == nil {
				t.Errorf("DBPool() = %v, want %v", got, tt.want)
			}
		})
	}
}
