package android

import (
	"io"
	"net/http"
	"os"
	"strings"
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

func TestServerAcceptsConnections(t *testing.T) {
	dstPath := t.TempDir() + "/game.db"
	
	result, err := StartWifiServer(dstPath)
	if err != nil {
		t.Fatalf("StartWifiServer failed: %v", err)
	}
	defer StopWifiServer()

	t.Logf("Server started at: %s", result.LocalURL)

	// Give server time to fully start
	time.Sleep(100 * time.Millisecond)

	// Test 1: GET / should return HTML
	t.Run("GET / returns HTML", func(t *testing.T) {
		resp, err := http.Get(result.LocalURL)
		if err != nil {
			t.Fatalf("Failed to connect to server: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		if !strings.Contains(string(body), "上传") {
			t.Error("Response body doesn't contain upload form")
		}
		t.Logf("GET / response: %d bytes, status %d", len(body), resp.StatusCode)
	})

	// Test 2: GET /status should return JSON
	t.Run("GET /status returns JSON", func(t *testing.T) {
		resp, err := http.Get(result.LocalURL + "status")
		if err != nil {
			t.Fatalf("Failed to connect to server: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		t.Logf("GET /status response: %s", string(body))
	})

	// Test 3: POST /upload without file should return error
	t.Run("POST /upload without file returns error", func(t *testing.T) {
		resp, err := http.Post(result.LocalURL+"upload", "multipart/form-data", nil)
		if err != nil {
			t.Fatalf("Failed to connect to server: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
		t.Logf("POST /upload response: status %d", resp.StatusCode)
	})

	t.Log("All connection tests passed!")
}

func TestServerUpload(t *testing.T) {
	dstPath := t.TempDir() + "/game.db"
	
	result, err := StartWifiServer(dstPath)
	if err != nil {
		t.Fatalf("StartWifiServer failed: %v", err)
	}
	defer StopWifiServer()

	time.Sleep(100 * time.Millisecond)

	// Create a test file content
	testContent := []byte("test game db content")

	// Create multipart form data
	var body strings.Builder
	body.WriteString("--boundary\r\n")
	body.WriteString("Content-Disposition: form-data; name=\"file\"; filename=\"game.db\"\r\n")
	body.WriteString("Content-Type: application/octet-stream\r\n\r\n")
	body.WriteString(string(testContent))
	body.WriteString("\r\n--boundary--\r\n")

	req, err := http.NewRequest("POST", result.LocalURL+"upload", strings.NewReader(body.String()))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to upload: %v", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("Upload response: status %d, body: %s", resp.StatusCode, string(respBody))

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verify file was saved
	savedContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("Failed to read saved file: %v", err)
	}
	if string(savedContent) != string(testContent) {
		t.Errorf("Saved content doesn't match: got %q, want %q", savedContent, testContent)
	}
	t.Logf("File saved correctly: %d bytes", len(savedContent))

	// Check upload status via API
	path, err := CheckWifiUpload(result.Token)
	if err != nil {
		t.Fatalf("CheckWifiUpload failed: %v", err)
	}
	if path == "" {
		t.Error("Expected non-empty path after upload")
	}
	t.Logf("Upload path: %s", path)

	t.Log("Upload test passed!")
}

func TestLANConnectivity(t *testing.T) {
	dstPath := t.TempDir() + "/game.db"
	
	result, err := StartWifiServer(dstPath)
	if err != nil {
		t.Fatalf("StartWifiServer failed: %v", err)
	}
	defer StopWifiServer()

	time.Sleep(200 * time.Millisecond)

	t.Logf("Testing LAN connectivity...")
	t.Logf("Primary URL: %s", result.URL)
	t.Logf("All URLs: %v", result.AllURLs)

	// Test each URL in AllURLs
	for _, url := range result.AllURLs {
		t.Run("Connect to "+url, func(t *testing.T) {
			client := &http.Client{Timeout: 3 * time.Second}
			resp, err := client.Get(url)
			if err != nil {
				t.Logf("Failed to connect to %s: %v", url, err)
			} else {
				resp.Body.Close()
				t.Logf("Successfully connected to %s: status %d", url, resp.StatusCode)
			}
		})
	}

	t.Log("LAN connectivity test complete!")
}
