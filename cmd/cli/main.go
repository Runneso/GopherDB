package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"github.com/google/uuid"

	"GopherDB/internal/protocol"
)

func main() {
	host := flag.String("host", "127.0.0.1", "Server DB host")
	port := flag.Int("port", 8081, "Server DB port")
	trace := flag.Bool("trace", false, "Enable explain trace")
	flag.Parse()

	addr := fmt.Sprintf("%s:%d", *host, *port)
	conn, err := net.Dial("tcp", addr)

	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to connect to %s: %v\n", addr, err)
		os.Exit(1)
	}
	defer func() {
		_ = conn.Close()
	}()

	client := newClient(conn)

	fmt.Printf("Connected to %s. End SQL statements with ';'. Type \\help for help, \\q to quit.\n", addr)

	if err := repl(client, *trace); err != nil && err != io.EOF {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

type client struct {
	reader *bufio.Reader
	writer *bufio.Writer
}

func newClient(conn net.Conn) *client {
	return &client{
		reader: bufio.NewReader(conn),
		writer: bufio.NewWriter(conn),
	}
}

func (c *client) sendQuery(sql string, trace bool) (*protocol.Response, error) {
	requestID := uuid.New().String()
	req := protocol.NewRequest(requestID, sql, trace)
	if err := c.writeJSONLine(req); err != nil {
		return nil, err
	}

	respLine, err := c.reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	var resp protocol.Response
	if err := json.Unmarshal(respLine, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *client) writeJSONLine(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if _, err := c.writer.Write(data); err != nil {
		return err
	}
	if err := c.writer.WriteByte('\n'); err != nil {
		return err
	}
	return c.writer.Flush()
}

func repl(client *client, trace bool) error {
	stdin := bufio.NewReader(os.Stdin)
	var buf strings.Builder

	for {
		if buf.Len() == 0 {
			fmt.Print("db> ")
		} else {
			fmt.Print("   > ")
		}

		line, err := stdin.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println()
				return nil
			}
			return err
		}

		trimmed := strings.TrimSpace(line)
		if buf.Len() == 0 {
			switch strings.ToLower(trimmed) {
			case "\\q", "quit", "exit":
				return nil
			case "\\help":
				printHelp()
				continue
			}
		}

		buf.WriteString(line)

		statements, remainder := splitStatementsWithRemainder(buf.String())
		if len(statements) == 0 {
			continue
		}

		for _, sql := range statements {
			if strings.TrimSpace(sql) == "" {
				continue
			}
			resp, err := client.sendQuery(sql, trace)
			if err != nil {
				return err
			}
			printResponse(resp)
		}

		buf.Reset()
		buf.WriteString(remainder)
	}
}

func printHelp() {
	fmt.Println("Commands:")
	fmt.Println("  \\q | quit | exit   - exit")
	fmt.Println("  \\help              - help")
	fmt.Println()
	fmt.Println("End SQL statements with ';'.")
}

func printResponse(resp *protocol.Response) {
	if resp == nil {
		fmt.Println("(null response)")
		return
	}

	if strings.ToLower(resp.Status) != "ok" {
		if resp.Error == nil {
			fmt.Println("ERROR: unknown")
			return
		}
		where := ""
		if resp.Error.Pos != nil {
			where = fmt.Sprintf(" at %d:%d", resp.Error.Pos.Line, resp.Error.Pos.Column)
		}
		fmt.Printf("ERROR[%s]%s: %s\n", resp.Error.Code, where, resp.Error.Message)
		return
	}

	if resp.Explain != "" {
		fmt.Println(resp.Explain)
	}

	if len(resp.Columns) > 0 && len(resp.Rows) > 0 {
		printTable(resp.Columns, resp.Rows)
		return
	}
	if resp.Affected != 0 {
		fmt.Printf("OK (affected=%d)\n", resp.Affected)
		return
	}
	fmt.Println("OK")
}

func printTable(columns []string, rows [][]any) {
	widths := make([]int, len(columns))
	for i, col := range columns {
		if col == "" {
			widths[i] = 4
		} else {
			widths[i] = len(col)
		}
	}
	for _, row := range rows {
		for i := range columns {
			cell := "null"
			if i < len(row) && row[i] != nil {
				cell = fmt.Sprint(row[i])
			}
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	sep := buildSeparator(widths)
	fmt.Println(sep)
	fmt.Println(buildRow(columns, widths))
	fmt.Println(sep)
	for _, row := range rows {
		cells := make([]string, len(columns))
		for i := range columns {
			if i < len(row) && row[i] != nil {
				cells[i] = fmt.Sprint(row[i])
			} else {
				cells[i] = "null"
			}
		}
		fmt.Println(buildRow(cells, widths))
	}
	fmt.Println(sep)
	fmt.Printf("%d row(s)\n", len(rows))
}

func buildSeparator(widths []int) string {
	var sb strings.Builder
	sb.WriteByte('+')
	for _, w := range widths {
		sb.WriteString(strings.Repeat("-", w+2))
		sb.WriteByte('+')
	}
	return sb.String()
}

func buildRow(cells []string, widths []int) string {
	var sb strings.Builder
	sb.WriteByte('|')
	for i, w := range widths {
		cell := "null"
		if i < len(cells) && cells[i] != "" {
			cell = cells[i]
		}
		sb.WriteByte(' ')
		sb.WriteString(padRight(cell, w))
		sb.WriteByte(' ')
		sb.WriteByte('|')
	}
	return sb.String()
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

func splitStatementsWithRemainder(text string) ([]string, string) {
	out := make([]string, 0)
	if text == "" {
		return out, ""
	}

	inString := false
	start := 0
	for i := 0; i < len(text); i++ {
		c := text[i]
		if c == '\'' {
			if inString && i+1 < len(text) && text[i+1] == '\'' {
				i++
				continue
			}
			inString = !inString
			continue
		}
		if c == ';' && !inString {
			stmt := strings.TrimSpace(text[start : i+1])
			if stmt != "" {
				out = append(out, stmt)
			}
			start = i + 1
		}
	}
	return out, text[start:]
}
