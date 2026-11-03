package unit_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"ydb-installer/internal/models"
	"ydb-installer/internal/services/config"
)

// TestInventoryWriter_ConcurrentWriteWithPatch verifies that overlapping
// read-modify-write cycles are serialized by the writer mutex: the file always
// stays a complete, parseable inventory owned by a single write — never a torn
// mix of hosts from different writes.
func TestInventoryWriter_ConcurrentWriteWithPatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "50-inventory.yaml")
	writer := config.NewInventoryWriter()

	seed := &models.InventoryData{
		Hosts: []string{"seed.example.com"},
		DC1:   []string{"seed.example.com"},
	}
	if err := writer.Write(path, seed); err != nil {
		t.Fatalf("seed write: %v", err)
	}

	const writers = 8
	var wg sync.WaitGroup
	errs := make([]error, writers)
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			host := fmt.Sprintf("host-%d.example.com", n)
			data := &models.InventoryData{
				Hosts: []string{host},
				DC1:   []string{host},
			}
			errs[n] = writer.WriteWithPatch(path, data)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("concurrent write %d failed: %v", i, err)
		}
	}

	invSvc := config.NewInventoryService()
	parsed, err := invSvc.Parse(path)
	if err != nil {
		t.Fatalf("parse after concurrent writes: %v", err)
	}

	// Each write carries a single host — serialized writes must leave exactly
	// one writer's host set, never a corrupted or mixed inventory.
	writerHosts := 0
	for _, h := range parsed.Hosts {
		if strings.HasPrefix(h, "host-") {
			writerHosts++
		}
	}
	if writerHosts != 1 {
		t.Fatalf("expected exactly one writer's hosts in the final file, got %d: %v", writerHosts, parsed.Hosts)
	}
}
