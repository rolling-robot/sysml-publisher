package servicetest_test

import (
	"testing"

	"github.com/rolling-robot/sysml-publisher/internal/servicetest"
)

func TestAPISpawnsAService(t *testing.T) {
	addr := servicetest.API(t)
	t.Log("service at", addr)
	if addr == "" {
		t.Fatal("empty address")
	}
	if again := servicetest.API(t); again != addr {
		t.Errorf("API(t) = %s again, want the same service", again)
	}
}
