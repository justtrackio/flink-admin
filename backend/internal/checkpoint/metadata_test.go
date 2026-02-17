package checkpoint

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFileSummary(t *testing.T) {
	metadataPath := filepath.Join("..", "..", "_metadata")
	if _, err := os.Stat(metadataPath); err != nil {
		t.Skip("metadata file not found")
	}

	summary, err := ParseFileSummary(metadataPath, ParseOptions{IncludeInlineStrings: true})
	if err != nil {
		t.Fatalf("parse summary: %v", err)
	}

	if summary.Version != 4 {
		t.Fatalf("expected version 4, got %d", summary.Version)
	}
	if summary.CheckpointID != 1151012 {
		t.Fatalf("expected checkpoint id 1151012, got %d", summary.CheckpointID)
	}
	if summary.NumOperators == 0 {
		t.Fatalf("expected operators, got 0")
	}
	if len(summary.StateFilePaths) == 0 {
		t.Fatalf("expected state file paths")
	}
}
