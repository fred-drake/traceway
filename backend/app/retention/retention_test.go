package retention

import "testing"

func TestTelemetryRetentionConfig(t *testing.T) {
	cases := []struct {
		name       string
		isDuckDB   bool
		sqliteDays string
		duckdbDays string
		wantDays   int
		wantSource string
	}{
		{"sqlite default", false, "", "", 30, "SQLITE_RETENTION_DAYS"},
		{"sqlite explicit", false, "7", "", 7, "SQLITE_RETENTION_DAYS"},
		{"sqlite ignores duckdb var", false, "7", "90", 7, "SQLITE_RETENTION_DAYS"},
		{"duckdb default", true, "", "", 30, "SQLITE_RETENTION_DAYS"},
		{"duckdb explicit", true, "", "90", 90, "DUCKDB_RETENTION_DAYS"},
		{"duckdb wins over sqlite var", true, "7", "90", 90, "DUCKDB_RETENTION_DAYS"},
		{"duckdb falls back to sqlite var", true, "7", "", 7, "SQLITE_RETENTION_DAYS"},
		{"duckdb disabled", true, "7", "0", 0, "DUCKDB_RETENTION_DAYS"},
		{"duckdb invalid uses default", true, "", "abc", 30, "DUCKDB_RETENTION_DAYS"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			days, source := telemetryRetentionConfig(tc.isDuckDB, tc.sqliteDays, tc.duckdbDays)
			if days != tc.wantDays || source != tc.wantSource {
				t.Errorf("got (%d, %s), want (%d, %s)", days, source, tc.wantDays, tc.wantSource)
			}
		})
	}
}
