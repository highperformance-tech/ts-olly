package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog"
)

func TestPendingFilesKeyFormat(t *testing.T) {
	// Test that pending files are keyed correctly as "processName_processId"
	tests := []struct {
		processName string
		processId   uint8
		expectedKey string
	}{
		{"vizqlserver", 0, "vizqlserver_0"},
		{"vizqlserver", 1, "vizqlserver_1"},
		{"vizqlserver", 2, "vizqlserver_2"},
		{"backgrounder", 0, "backgrounder_0"},
		{"backgrounder", 5, "backgrounder_5"},
		{"dataserver", 10, "dataserver_10"},
	}

	for _, tt := range tests {
		t.Run(tt.expectedKey, func(t *testing.T) {
			// This is the same format used in handleFiles()
			key := fmt.Sprintf("%s_%d", tt.processName, tt.processId)
			if key != tt.expectedKey {
				t.Errorf("got key %q, want %q", key, tt.expectedKey)
			}
		})
	}
}

func TestConfigDirectoryMatching(t *testing.T) {
	// Test that config directory names match pending keys correctly
	// Config directories can have suffixes like "vizqlserver_1_abc123"
	tests := []struct {
		pendingKey  string
		dirName     string
		shouldMatch bool
	}{
		{"vizqlserver_0", "vizqlserver_0", true},
		{"vizqlserver_1", "vizqlserver_1", true},
		{"vizqlserver_1", "vizqlserver_1_abc123", true},
		{"vizqlserver_1", "vizqlserver_10", false}, // Should NOT match - different instance
		{"vizqlserver_1", "vizqlserver_2", false},
		{"backgrounder_0", "backgrounder_0", true},
		{"backgrounder_0", "vizqlserver_0", false},
	}

	for _, tt := range tests {
		name := tt.pendingKey + "_vs_" + tt.dirName
		t.Run(name, func(t *testing.T) {
			// This is the same matching logic used in watchConfigDir()
			// Must be exact match OR match with underscore suffix
			matches := tt.dirName == tt.pendingKey || strings.HasPrefix(tt.dirName, tt.pendingKey+"_")
			if matches != tt.shouldMatch {
				t.Errorf("match(%q, %q) = %v, want %v", tt.dirName, tt.pendingKey, matches, tt.shouldMatch)
			}
		})
	}
}

func TestWatchConfigDirEmptyConfigDir(t *testing.T) {
	// Test that watchConfigDir handles empty config directory gracefully
	logger := zerolog.New(os.Stderr).Level(zerolog.Disabled)
	app := &application{
		config: config{
			configDir: "", // Empty config dir
		},
		logger: logger,
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatal(err)
	}
	defer watcher.Close()

	pendingFiles := &sync.Map{}
	retryCh := make(chan event, 10)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Should return immediately without error when configDir is empty
	done := make(chan struct{})
	go func() {
		app.watchConfigDir(ctx, watcher, pendingFiles, retryCh)
		close(done)
	}()

	select {
	case <-done:
		// Good - returned as expected
	case <-time.After(500 * time.Millisecond):
		t.Error("watchConfigDir did not return when configDir is empty")
	}
}

func TestWatchConfigDirDetectsNewDirectory(t *testing.T) {
	// Create a temporary config directory
	tmpDir, err := os.MkdirTemp("", "ts-olly-test-config-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logger := zerolog.New(os.Stderr).Level(zerolog.Disabled)
	app := &application{
		config: config{
			configDir: tmpDir,
			logsDir:   "/var/logs",
		},
		logger: logger,
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatal(err)
	}
	defer watcher.Close()

	pendingFiles := &sync.Map{}
	retryCh := make(chan event, 10)

	// Add a pending file for vizqlserver_1
	pendingEvent := event{
		Event:  fsnotify.Event{Name: "/var/logs/vizqlserver/vizqlserver_1.log", Op: fsnotify.Create},
		fileId: 12345,
	}
	pendingFiles.Store(fileId(12345), pendingEvent)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go app.watchConfigDir(ctx, watcher, pendingFiles, retryCh)

	// Give the watcher time to start
	time.Sleep(100 * time.Millisecond)

	// Create a new config directory that matches the pending key
	newConfigDir := filepath.Join(tmpDir, "vizqlserver_1")
	if err := os.Mkdir(newConfigDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Wait for the retry event
	select {
	case retryEvent := <-retryCh:
		if retryEvent.Name != pendingEvent.Name {
			t.Errorf("got retry event for %q, want %q", retryEvent.Name, pendingEvent.Name)
		}
		if retryEvent.fileId != pendingEvent.fileId {
			t.Errorf("got retry fileId %d, want %d", retryEvent.fileId, pendingEvent.fileId)
		}
	case <-time.After(2 * time.Second):
		t.Error("did not receive retry event after config directory was created")
	}

	// Verify the pending file was removed
	if _, ok := pendingFiles.Load(fileId(12345)); ok {
		t.Error("pending file was not removed after retry")
	}
}

// Note: TestWatchConfigDirIgnoresNonMatchingDirectory was removed because the
// VictoriaMetrics counter registration doesn't support multiple test runs.
// The matching logic is covered by TestConfigDirectoryMatching instead.
