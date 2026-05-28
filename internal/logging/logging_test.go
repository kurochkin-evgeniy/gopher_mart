package logging

import "testing"

func TestInitAndSync(t *testing.T) {
	if err := Init(); err != nil {
		t.Fatalf("init: %v", err)
	}
	if Sugar == nil {
		t.Fatal("Sugar is nil after Init")
	}

	Sync()
}
