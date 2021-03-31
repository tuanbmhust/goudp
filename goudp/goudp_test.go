package main

import (
	"testing"
)

func TestAppend(t *testing.T) {
	expectAppend(t, "localhost", ":8080", "localhost:8080", 1)
	expectAppend(t, "localhost:8080", ":8888", "localhost:8080", 2)

	expectAppend(t, "127.0.0.1", ":1337", "127.0.0.1:1337", 3)
	expectAppend(t, "127.0.0.1:1337", ":1333", "127.0.0.1:1337", 4)

	expectAppend(t, "localhost", ":1337", "localhost:1337", 5)
	expectAppend(t, "localhost:1337", "1337", "localhost:1337", 6)

	expectAppend(t, "[::1]", ":80", "[::1]:80", 7)
	expectAppend(t, "[::1]:8080", ":80", "[::1]:8080", 8)
}

func expectAppend(t *testing.T, host, port, wanted string, id int) {
	result := appendPortIfMissing(host, port)
	if result != wanted {
		t.Errorf("expectAppend %d: host=%s, port=%s, result=%s, wanted = %s", id, host, port, result, wanted)
	}
}
