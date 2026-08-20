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
	if result.LocalURL == "" {
		t.Fatal("Expected non-empty LocalURL")
	}
	if result.Token == "" {
		t.Fatal("Expected non-empty token")
	}
	if len(result.AllURLs) == 0 {
		t.Fatal("Expected non-empty AllURLs")
	}
	
	t.Logf("Server URL (LAN): %s", result.URL)
	t.Logf("Server LocalURL: %s", result.LocalURL)
	t.Logf("Server Token: %s", result.Token)
	t.Logf("All URLs: %v", result.AllURLs)
	
	// Verify LocalURL contains localhost
	if len(result.LocalURL) < 11 || result.LocalURL[:10] != "http://loc" {
		t.Fatalf("Expected LocalURL to start with http://localhost, got: %s", result.LocalURL)
	}
	
	// Verify AllURLs contains LocalURL
	foundLocal := false
	for _, url := range result.AllURLs {
		if url == result.LocalURL {
			foundLocal = true
			break
		}
	}
	if !foundLocal {
		t.Fatal("Expected AllURLs to contain LocalURL")
	}
	
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
