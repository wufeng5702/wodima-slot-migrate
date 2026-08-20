package android

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// WifiServer manages a temporary HTTP server for receiving game.db via Wi-Fi.
type WifiServer struct {
	mu       sync.Mutex
	server   *http.Server
	listener net.Listener
	token    string
	result   string // path to uploaded file
	done     chan struct{}
	dstPath  string // where to save uploaded file
}

// StartWifiServer starts an HTTP server on the local network and returns
// the URL and token for the client to upload game.db from their phone.
func StartWifiServer(dstPath string) (*WifiResult, error) {
	// Get all candidate IPs
	primaryIP, allIPs := getAllIPs()

	// Bind to all interfaces for reliability
	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		return nil, fmt.Errorf("bind port: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port

	// Generate token for this session
	token := fmt.Sprintf("%d", time.Now().UnixNano())

	// Create server
	s := &WifiServer{
		listener: listener,
		token:    token,
		done:     make(chan struct{}),
		dstPath:  dstPath,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/upload", s.handleUpload)
	mux.HandleFunc("/status", s.handleStatus)

	s.server = &http.Server{Handler: mux}

	// Serve in background
	go func() {
		if err := s.server.Serve(listener); err != http.ErrServerClosed {
			_ = err
		}
	}()

	// Build URLs
	url := fmt.Sprintf("http://%s:%d/", primaryIP, port)
	localURL := fmt.Sprintf("http://localhost:%d/", port)

	// Build all access URLs
	var allURLs []string
	for _, ip := range allIPs {
		allURLs = append(allURLs, fmt.Sprintf("http://%s:%d/", ip, port))
	}
	allURLs = append(allURLs, localURL)

	// Get debug info
	debugInfo := fmt.Sprintf("Auto-detected primary IP: %s\nAll candidate IPs: %v\nNetwork interfaces:\n%s",
		primaryIP, allIPs, listNetworkInterfaces())

	// Store server reference
	mu.Lock()
	activeServer = s
	mu.Unlock()

	return &WifiResult{
		URL:       url,
		LocalURL:  localURL,
		AllURLs:   allURLs,
		Token:     token,
		DebugInfo: debugInfo,
	}, nil
}

// StopWifiServer stops the currently running Wi-Fi server if any.
func StopWifiServer() {
	mu.Lock()
	s := activeServer
	activeServer = nil
	mu.Unlock()
	if s != nil {
		if s.server != nil {
			_ = s.server.Close()
		}
		if s.listener != nil {
			_ = s.listener.Close()
		}
	}
}

// CheckWifiUpload polls the current Wi-Fi server for upload completion.
func CheckWifiUpload(token string) (string, error) {
	mu.Lock()
	s := activeServer
	mu.Unlock()
	if s == nil || s.token != token {
		return "", nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.result, nil
}

// Global server reference for management
var (
	mu           sync.Mutex
	activeServer *WifiServer
)

// WifiResult holds the connection info for the client.
type WifiResult struct {
	URL       string   `json:"url"`      // Best guess URL for phone access
	LocalURL  string   `json:"localUrl"` // localhost URL for local testing
	AllURLs   []string `json:"allUrls"`  // All available IP-based URLs
	Token     string   `json:"token"`
	DebugInfo string   `json:"debugInfo"` // Debug info for troubleshooting
}

func (s *WifiServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>上传 game.db</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; max-width: 500px; margin: 40px auto; padding: 20px; }
    h1 { color: #333; }
    .upload-area { border: 2px dashed #ccc; border-radius: 12px; padding: 40px; text-align: center; cursor: pointer; transition: border-color 0.3s; }
    .upload-area:hover, .upload-area.dragover { border-color: #667eea; }
    .upload-area p { color: #666; margin: 10px 0; }
    #file-input { display: none; }
    .hint { background: #f5f5f5; padding: 16px; border-radius: 8px; margin-top: 16px; font-size: 14px; color: #555; }
    .hint code { background: #ddd; padding: 2px 6px; border-radius: 4px; }
    .status { margin-top: 16px; padding: 12px; border-radius: 8px; text-align: center; display: none; }
    .status.success { background: #d4edda; color: #155724; display: block; }
    .status.error { background: #f8d7da; color: #721c24; display: block; }
    .progress { width: 100%; height: 20px; background: #eee; border-radius: 10px; overflow: hidden; margin-top: 16px; display: none; }
    .progress-bar { height: 100%; background: #667eea; width: 0%; transition: width 0.3s; }
  </style>
</head>
<body>
  <h1>📤 上传游戏存档</h1>
  <p>请将 <code>game.db</code> 文件上传到电脑</p>

  <div class="hint">
    <strong>📱 操作指引：</strong><br>
    1. 打开手机的「文件管理」App<br>
    2. 进入路径：<code>Android/data/com.itaotuo.wodima/files/</code><br>
    3. 复制 <code>game.db</code> 到「Download」文件夹<br>
    4. 点击下方按钮选择文件
  </div>

  <div class="upload-area" id="drop-zone">
    <p>📁 点击选择 game.db 文件</p>
    <p style="font-size: 14px; color: #999;">或将文件拖到此处</p>
  </div>
  <input type="file" id="file-input" accept=".db,.sqlite">

  <div class="progress" id="progress-container">
    <div class="progress-bar" id="progress-bar"></div>
  </div>

  <div class="status" id="status"></div>

  <script>
    const dropZone = document.getElementById('drop-zone');
    const fileInput = document.getElementById('file-input');
    const progressContainer = document.getElementById('progress-container');
    const progressBar = document.getElementById('progress-bar');
    const status = document.getElementById('status');

    dropZone.addEventListener('click', () => fileInput.click());

    fileInput.addEventListener('change', (e) => {
      if (e.target.files[0]) uploadFile(e.target.files[0]);
    });

    dropZone.addEventListener('dragover', (e) => {
      e.preventDefault();
      dropZone.classList.add('dragover');
    });

    dropZone.addEventListener('dragleave', () => {
      dropZone.classList.remove('dragover');
    });

    dropZone.addEventListener('drop', (e) => {
      e.preventDefault();
      dropZone.classList.remove('dragover');
      if (e.dataTransfer.files[0]) uploadFile(e.dataTransfer.files[0]);
    });

    function uploadFile(file) {
      const formData = new FormData();
      formData.append('file', file);

      const xhr = new XMLHttpRequest();
      xhr.open('POST', '/upload');

      progressContainer.style.display = 'block';
      status.className = 'status';
      status.textContent = '';

      xhr.upload.onprogress = (e) => {
        if (e.lengthComputable) {
          const percent = (e.loaded / e.total) * 100;
          progressBar.style.width = percent + '%';
        }
      };

      xhr.onload = () => {
        progressContainer.style.display = 'none';
        if (xhr.status === 200) {
          status.className = 'status success';
          status.textContent = '✅ 上传成功！请返回电脑查看。';
        } else {
          status.className = 'status error';
          status.textContent = '❌ 上传失败：' + xhr.responseText;
        }
      };

      xhr.onerror = () => {
        progressContainer.style.display = 'none';
        status.className = 'status error';
        status.textContent = '❌ 网络错误，请重试';
      };

      xhr.send(formData);
    }
  </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, html)
}

func (s *WifiServer) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form (max 64MB)
	r.Body = http.MaxBytesReader(w, r.Body, 64<<20)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "No file uploaded", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Save to configured destination path
	dstPath := s.dstPath

	dst, err := os.Create(dstPath)
	if err != nil {
		http.Error(w, "Failed to create file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	// Mark as done
	s.mu.Lock()
	s.result = dstPath
	s.mu.Unlock()

	// Notify success
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *WifiServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	result := s.result
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"path": result})
}

// getAllIPs returns the best guess primary IP and all candidate IPs.
// It first tries to determine the primary IP by dialing a known address,
// then falls back to listing all active non-loopback IPv4 addresses.
func getAllIPs() (primaryIP string, allIPs []string) {
	// First, try to determine the primary IP by connecting to a known address
	primaryIP = primaryOutboundIP()

	// Collect all candidate IPs (excluding loopback, link-local, CGNAT)
	allIPs = collectCandidateIPs()

	// If primary IP is not in candidates, add it
	found := false
	for _, ip := range allIPs {
		if ip == primaryIP {
			found = true
			break
		}
	}
	if !found && primaryIP != "" && primaryIP != "127.0.0.1" {
		allIPs = append([]string{primaryIP}, allIPs...)
	}

	// If no IPs found at all, use localhost
	if len(allIPs) == 0 {
		primaryIP = "127.0.0.1"
		allIPs = []string{"127.0.0.1"}
	}

	// If primary IP is empty, use first candidate
	if primaryIP == "" {
		primaryIP = allIPs[0]
	}

	return primaryIP, allIPs
}

// primaryOutboundIP determines the primary outbound IP by dialing a known address.
// Returns empty string if it cannot be determined (e.g., no internet access).
func primaryOutboundIP() string {
	// Try multiple well-known addresses for reliability
	candidates := []string{
		"8.8.8.8:80",         // Google DNS
		"1.1.1.1:80",         // Cloudflare DNS
		"114.114.114.114:80", // 114 DNS (China)
		"223.5.5.5:80",       // Ali DNS (China)
	}

	for _, addr := range candidates {
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err != nil {
			continue
		}
		defer conn.Close()

		localAddr := conn.LocalAddr().(*net.TCPAddr)
		if ip4 := localAddr.IP.To4(); ip4 != nil {
			return ip4.String()
		}
	}

	return ""
}

// collectCandidateIPs returns all non-loopback, non-link-local, non-CGNAT IPv4 addresses.
func collectCandidateIPs() []string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}

	var candidates []string
	for _, iface := range ifaces {
		// Skip down interfaces
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		// Skip loopback and point-to-point (typically VPNs)
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagPointToPoint != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				ip := ipnet.IP
				if ip4 := ip.To4(); ip4 != nil {
					// Skip loopback, link-local, and CGNAT (100.64.0.0/10)
					if ip4.IsLoopback() || ip4.IsLinkLocalUnicast() || isCGNAT(ip4) {
						continue
					}
					candidates = append(candidates, ip4.String())
				}
			}
		}
	}

	// Prioritize 192.168.x.x IPs (most common for home/office LANs)
	var primary []string
	var secondary []string
	for _, ip := range candidates {
		if strings.HasPrefix(ip, "192.168.") {
			primary = append(primary, ip)
		} else {
			secondary = append(secondary, ip)
		}
	}

	result := append(primary, secondary...)
	return result
}

// isCGNAT checks if an IP is in the Carrier-Grade NAT range (100.64.0.0/10)
func isCGNAT(ip net.IP) bool {
	_, cgnatNet, _ := net.ParseCIDR("100.64.0.0/10")
	return cgnatNet.Contains(ip)
}

// listNetworkInterfaces returns a summary of all active network interfaces
// for debugging purposes.
func listNetworkInterfaces() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "failed to list interfaces: " + err.Error()
	}

	var result string
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		ifaceDesc := iface.Name
		if iface.Flags&net.FlagPointToPoint != 0 {
			ifaceDesc += " (VPN/PPP)"
		}
		if iface.Flags&net.FlagLoopback != 0 {
			ifaceDesc += " (loopback)"
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				ip4 := ipnet.IP.To4()
				if ip4 != nil {
					result += fmt.Sprintf("  %s: %s\n", ifaceDesc, ip4.String())
				}
			}
		}
	}
	return result
}
