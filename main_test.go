package eltest_test

import (
	"log"
	"os"
	"testing"

	"github.com/ttab/eltest"
)

// testValue is the payload stored and read back in the round-trip tests.
const testValue = "world"

func TestMain(m *testing.M) {
	code := m.Run()

	err := eltest.PurgeBackingServices()
	if err != nil {
		log.Printf("failed to purge backing services: %v", err)
	}

	os.Exit(code)
}
