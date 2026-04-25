package main

import (
	"bufio"
	"net"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// Helper to start the server and return a cleanup function
func startServer(t *testing.T) func() {
	cmd := exec.Command("go", "run", "main.go")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start main.go: %v", err)
	}
	// Wait for server to start
	ready := false
	for i := 0; i < 20; i++ {
		conn, err := net.DialTimeout("tcp", "127.0.0.1:7379", 200*time.Millisecond)
		if err == nil {
			conn.Close()
			ready = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !ready {
		cmd.Process.Kill()
		t.Fatalf("Server did not start in time")
	}
	return func() { cmd.Process.Kill() }
}

func TestServerPingPong(t *testing.T) {
	cleanup := startServer(t)
	defer cleanup()
	conn, err := net.Dial("tcp", "127.0.0.1:7379")
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()
	// Send PING
	_, err = conn.Write([]byte("*1\r\n$4\r\nPING\r\n"))
	if err != nil {
		t.Fatalf("Failed to write to server: %v", err)
	}
	resp, _ := bufio.NewReader(conn).ReadString('\n')
	if !strings.Contains(resp, "+PONG") {
		t.Errorf("Expected +PONG, got: %q", resp)
	}
}

func TestSetGetDel(t *testing.T) {
	cleanup := startServer(t)
	defer cleanup()
	conn, err := net.Dial("tcp", "127.0.0.1:7379")
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()
	reader := bufio.NewReader(conn)
	// SET
	_, err = conn.Write([]byte("*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\nbar\r\n"))
	if err != nil {
		t.Fatalf("Failed to write SET: %v", err)
	}
	resp, _ := reader.ReadString('\n')
	if !strings.Contains(resp, "+OK") {
		t.Errorf("Expected +OK for SET, got: %q", resp)
	}
	// GET
	_, err = conn.Write([]byte("*2\r\n$3\r\nGET\r\n$3\r\nfoo\r\n"))
	if err != nil {
		t.Fatalf("Failed to write GET: %v", err)
	}
	resp, _ = reader.ReadString('\n')
	if strings.HasPrefix(resp, "$") {
		// It's a bulk string, so read the next line too
		resp2, _ := reader.ReadString('\n')
		resp += resp2
	}
	if !strings.Contains(resp, "$3") && !strings.Contains(resp, "bar") {
		t.Errorf("Expected $3...bar for GET, got: %q", resp)
	}
	// DEL
	_, err = conn.Write([]byte("*2\r\n$3\r\nDEL\r\n$3\r\nfoo\r\n"))
	if err != nil {
		t.Fatalf("Failed to write DEL: %v", err)
	}
	resp, _ = reader.ReadString('\n')
	if !strings.Contains(resp, ":1") {
		t.Errorf("Expected :1 for DEL, got: %q", resp)
	}
}

func TestExpireAndTTL(t *testing.T) {
	cleanup := startServer(t)
	defer cleanup()
	conn, err := net.Dial("tcp", "127.0.0.1:7379")
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()
	reader := bufio.NewReader(conn)
	// SET with EX 1 (1 second)
	_, err = conn.Write([]byte("*5\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\nbar\r\n$2\r\nEX\r\n$1\r\n1\r\n"))
	if err != nil {
		t.Fatalf("Failed to write SET with EX: %v", err)
	}
	reader.ReadString('\n') // +OK
	// TTL should be 1 or 0
	_, err = conn.Write([]byte("*2\r\n$3\r\nTTL\r\n$3\r\nfoo\r\n"))
	if err != nil {
		t.Fatalf("Failed to write TTL: %v", err)
	}
	resp, _ := reader.ReadString('\n')
	if !strings.HasPrefix(resp, ":") {
		t.Errorf("Expected integer for TTL, got: %q", resp)
	}
	// Wait for key to expire
	time.Sleep(1200 * time.Millisecond)
	// GET should return nil
	_, err = conn.Write([]byte("*2\r\n$3\r\nGET\r\n$3\r\nfoo\r\n"))
	if err != nil {
		t.Fatalf("Failed to write GET after expire: %v", err)
	}
	resp, _ = reader.ReadString('\n')
	if !strings.Contains(resp, "$-1") {
		t.Errorf("Expected $-1 for expired key, got: %q", resp)
	}
}
