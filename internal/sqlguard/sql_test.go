package sqlguard

import "testing"

func TestCheckReadOnly(t *testing.T) {
	tests := []struct {
		sql     string
		allowed bool
	}{
		{"SELECT * FROM t WHERE id = 1", true},
		{"  -- note\nselect count(*) from `p.d.t`;", true},
		{"WITH a AS (SELECT 1) SELECT * FROM a", true},
		{"SELECT REPLACE(name, 'a', 'b') FROM t", true},
		{"SELECT 'DELETE FROM t; DROP TABLE t' AS s", true},
		{"SELECT $$ UPDATE t $$, $1", true},
		{"(SELECT 1) UNION ALL (SELECT 2)", true},
		{"EXPLAIN SELECT * FROM t", true},
		{"SHOW TABLES", true},
		{"DELETE FROM t", false},
		{"insert into t values (1)", false},
		{"UPDATE t SET a = 1", false},
		{"MERGE INTO t USING s ON true WHEN MATCHED THEN DELETE", false},
		{"CREATE TABLE t AS SELECT 1", false},
		{"WITH d AS (DELETE FROM t RETURNING *) SELECT * FROM d", false},
		{"SELECT * INTO new_t FROM t", false},
		{"SELECT * FROM t FOR UPDATE", false},
		{"SELECT * FROM t FOR SHARE", false},
		{"SELECT * FROM t LOCK IN SHARE MODE", false},
		{"EXPLAIN ANALYZE DELETE FROM t", false},
		{"SELECT 1; DROP TABLE t", false},
		{"SELECT 1 /* unclosed", false},
		{"SELECT 'it\\'s'; DELETE FROM t", false},
		{"CALL proc()", false},
		{"EXPORT DATA OPTIONS(uri='gs://b/*') AS SELECT 1", false},
		{"", false},
	}
	for _, tt := range tests {
		if err := CheckReadOnly(tt.sql); (err == nil) != tt.allowed {
			t.Errorf("CheckReadOnly(%q) = %v, want allowed %v", tt.sql, err, tt.allowed)
		}
	}
}
