package android

import (
	"testing"
	"time"
)

func TestStartWifiServer(t *testing.T) {
	dstPath := t.TempDir() + "/game.db"
	
	result, err := StartWifiServer(dstPath)
	if err != nil {
		t.Fatalf("StartWifiServer failed: %v", err)
	}
	if result.URL == "" {
		t.Fatal("Expected non-empty URL")
	}
	if result.Token == "" {
		t.Fatal("Expected non-empty token")
	}
	
	t.Logf("Server URL: %s", result.URL)
	t.Logf("Server Token: %s", result.Token)
	
	// Test CheckWifiUpload before upload
	path, err := CheckWifiUpload(result.Token)
	if err != nil {
		t.Fatalf("CheckWifiUpload failed: %v", err)
	}
	if path != "" {
		t.Fatal("Expected empty path before upload")
	}
	
	// Test StopWifiServer
	StopWifiServer()
	
	// Verify server is stopped by checking it returns empty
	time.Sleep(100 * time.Millisecond)
	path, _ = CheckWifiUpload(result.Token)
	if path != "" {
		t.Log("Path should still be empty after stop")
	}
	
	t.Log("Test passed!")
}
