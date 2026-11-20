package sqlguard

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunRelaysOnlyReadOnlySQL(t *testing.T) {
	requests := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"execute_sql","arguments":{"sql":"SELECT 1"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"execute_sql","arguments":{"sql":"DELETE FROM t"}}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"search_messages","arguments":{"query":"delete from"}}}`,
		`[{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"query","arguments":{"query":"UPDATE t SET a = 1"}}}]`,
	}, "\n") + "\n"
	var out bytes.Buffer
	// cat stands in for the MCP server: whatever reaches it comes back.
	if err := Run([]string{"cat"}, strings.NewReader(requests), &out, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{`"sql":"SELECT 1"`, `"query":"delete from"`, `{"id":2,`, `[{"id":4,`} {
		if !strings.Contains(got, want) {
			t.Errorf("output lacks %s:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{`DELETE FROM t"}`, `"query":"UPDATE t`} {
		if strings.Contains(got, unwanted) {
			t.Errorf("blocked call reached the server: %s\n%s", unwanted, got)
		}
	}
}
