package sqlguard

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
)

const Arg = "mcp-sql-guard"

const maxMessageBytes = 64 << 20

func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("%s: no MCP server command", Arg)
	}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stderr = stderr
	serverIn, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	serverOut, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	// Whole lines only, from both sides: a refusal written in the middle of a
	// server message would corrupt both.
	out := &lockedWriter{w: stdout}
	go func() {
		relayRequests(stdin, serverIn, out)
		serverIn.Close()
	}()
	relayLines(serverOut, out)
	return cmd.Wait()
}

func relayLines(from io.Reader, to *lockedWriter) {
	reader := bufio.NewReaderSize(from, 64<<10)
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			if _, werr := to.Write(line); werr != nil {
				io.Copy(io.Discard, reader)
				return
			}
		}
		if err != nil {
			return
		}
	}
}

func relayRequests(client io.Reader, server io.Writer, reply *lockedWriter) {
	scanner := bufio.NewScanner(client)
	scanner.Buffer(make([]byte, 0, 64<<10), maxMessageBytes)
	for scanner.Scan() {
		line := scanner.Bytes()
		if response := blockedResponse(line); response != nil {
			reply.Write(append(response, '\n'))
			continue
		}
		if _, err := server.Write(append(bytes.Clone(line), '\n')); err != nil {
			return
		}
	}
}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	} `json:"params"`
}

func blockedResponse(line []byte) []byte {
	trimmed := bytes.TrimSpace(line)
	if len(trimmed) > 0 && trimmed[0] == '[' {
		var batch []request
		if err := json.Unmarshal(trimmed, &batch); err != nil {
			return nil
		}
		var reasons []error
		for _, item := range batch {
			if err := check(item); err != nil {
				reasons = append(reasons, err)
			}
		}
		if len(reasons) == 0 {
			return nil
		}
		// One blocked call drops the whole batch rather than splitting it, so
		// the server never runs part of what was sent together.
		responses := make([]json.RawMessage, 0, len(batch))
		for _, item := range batch {
			if len(item.ID) > 0 {
				responses = append(responses, toolError(item.ID, reasons[0]))
			}
		}
		data, _ := json.Marshal(responses)
		return data
	}
	var item request
	if err := json.Unmarshal(trimmed, &item); err != nil {
		return nil
	}
	if err := check(item); err != nil {
		return toolError(item.ID, err)
	}
	return nil
}

func check(item request) error {
	if item.Method != "tools/call" {
		return nil
	}
	for _, sql := range sqlArguments(item.Params.Name, item.Params.Arguments) {
		if err := CheckReadOnly(sql); err != nil {
			return err
		}
	}
	return nil
}

// Servers name the SQL argument differently; "query" alone is too common in
// search tools to count unless the tool's name says it runs SQL.
func sqlArguments(tool string, arguments map[string]any) []string {
	name := strings.ToLower(tool)
	sqlTool := strings.Contains(name, "sql") || strings.Contains(name, "query") || strings.Contains(name, "execute")
	var found []string
	for key, value := range arguments {
		text, ok := value.(string)
		if !ok {
			continue
		}
		switch strings.ToLower(key) {
		case "sql", "statement", "sql_query":
			found = append(found, text)
		case "query":
			if sqlTool {
				found = append(found, text)
			}
		}
	}
	return found
}

func toolError(id json.RawMessage, reason error) []byte {
	data, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result": map[string]any{
			"isError": true,
			"content": []map[string]string{{
				"type": "text",
				"text": "ebi-x stopped this SQL: " + reason.Error() + ". Only read-only SQL can run through ebi-x. If writes are needed, tell the requester to ask the operator to change the settings.",
			}},
		},
	})
	return data
}

type lockedWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (l *lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}

func Main(args []string) {
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}
	if err := Run(args, os.Stdin, os.Stdout, os.Stderr); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			os.Exit(exit.ExitCode())
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
