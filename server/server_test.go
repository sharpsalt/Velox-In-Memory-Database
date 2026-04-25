package server_test

import (
	"testing"
)

func TestServerSanity(t *testing.T) {
	// A simple sanity test for the server package
	// A full integration test would require spinning up the TCP listener,
	// connecting to it, and asserting responses.
	t.Log("Server package compiles successfully")
}
