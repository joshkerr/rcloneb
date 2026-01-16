package rclone

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// FileItem represents a file or directory from rclone
type FileItem struct {
	Name    string `json:"Name"`
	Path    string `json:"Path"`
	Size    int64  `json:"Size"`
	IsDir   bool   `json:"IsDir"`
	ModTime string `json:"ModTime"`
}

// TransferStatus represents the status of a transfer
type TransferStatus int

const (
	StatusPending TransferStatus = iota
	StatusInProgress
	StatusCompleted
	StatusFailed
)

// Transfer represents an active file transfer
type Transfer struct {
	ID          string
	Source      string
	Destination string
	Status      TransferStatus
	Progress    float64
	BytesCopied int64
	BytesTotal  int64
	Speed       string
	StartTime   time.Time
	EndTime     time.Time
	Error       error
	mu          sync.Mutex
}

// TransferManager manages multiple file transfers
type TransferManager struct {
	transfers map[string]*Transfer
	mu        sync.RWMutex
}

// NewTransferManager creates a new transfer manager
func NewTransferManager() *TransferManager {
	return &TransferManager{
		transfers: make(map[string]*Transfer),
	}
}

// Add adds a new transfer to the manager
func (m *TransferManager) Add(id, source, destination string, totalBytes int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.transfers[id] = &Transfer{
		ID:          id,
		Source:      source,
		Destination: destination,
		Status:      StatusPending,
		BytesTotal:  totalBytes,
	}
}

// Start marks a transfer as in progress
func (m *TransferManager) Start(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if t, exists := m.transfers[id]; exists {
		t.mu.Lock()
		t.Status = StatusInProgress
		t.StartTime = time.Now()
		t.mu.Unlock()
	}
}

// UpdateProgress updates the progress of a transfer
func (m *TransferManager) UpdateProgress(id string, progress float64, bytesCopied, bytesTotal int64, speed string) {
	m.mu.RLock()
	t, exists := m.transfers[id]
	m.mu.RUnlock()

	if exists {
		t.mu.Lock()
		t.Progress = progress
		t.BytesCopied = bytesCopied
		if bytesTotal > 0 {
			t.BytesTotal = bytesTotal
		}
		t.Speed = speed
		t.mu.Unlock()
	}
}

// Complete marks a transfer as completed
func (m *TransferManager) Complete(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if t, exists := m.transfers[id]; exists {
		t.mu.Lock()
		t.Status = StatusCompleted
		t.Progress = 100
		t.EndTime = time.Now()
		t.mu.Unlock()
	}
}

// Fail marks a transfer as failed
func (m *TransferManager) Fail(id string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if t, exists := m.transfers[id]; exists {
		t.mu.Lock()
		t.Status = StatusFailed
		t.Error = err
		t.EndTime = time.Now()
		t.mu.Unlock()
	}
}

// Get returns a transfer by ID
func (m *TransferManager) Get(id string) *Transfer {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.transfers[id]
}

// GetAll returns all transfers
func (m *TransferManager) GetAll() []*Transfer {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Transfer, 0, len(m.transfers))
	for _, t := range m.transfers {
		result = append(result, t)
	}
	return result
}

// Stats returns pending, in-progress, completed, and failed counts
func (m *TransferManager) Stats() (pending, inProgress, completed, failed int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, t := range m.transfers {
		t.mu.Lock()
		switch t.Status {
		case StatusPending:
			pending++
		case StatusInProgress:
			inProgress++
		case StatusCompleted:
			completed++
		case StatusFailed:
			failed++
		}
		t.mu.Unlock()
	}
	return
}

// ListRemotes returns a list of configured rclone remotes
func ListRemotes() ([]string, error) {
	cmd := exec.Command("rclone", "listremotes")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list remotes: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	remotes := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			// Remove trailing colon if present
			remotes = append(remotes, strings.TrimSuffix(line, ":"))
		}
	}
	return remotes, nil
}

// ListFiles returns the files and directories at the given remote path
func ListFiles(remote, path string) ([]FileItem, error) {
	remotePath := remote + ":" + path
	cmd := exec.Command("rclone", "lsjson", remotePath)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list files at %s: %w", remotePath, err)
	}

	var items []FileItem
	if err := json.Unmarshal(output, &items); err != nil {
		return nil, fmt.Errorf("failed to parse file list: %w", err)
	}

	// Update paths to be full paths
	for i := range items {
		if path == "" {
			items[i].Path = items[i].Name
		} else {
			items[i].Path = path + "/" + items[i].Name
		}
	}

	return items, nil
}

// Regex to match "Transferred:" lines
// Example: "Transferred:   1.234 GiB / 5.678 GiB, 22%, 10 MiB/s, ETA 1m30s"
var statsRegex = regexp.MustCompile(`Transferred:\s+([0-9.]+)\s*([kKMGTP]i?[Bb]?)\s*/\s*([0-9.]+)\s*([kKMGTP]i?[Bb]?),\s*([0-9]+)%`)

// parseSize converts size string to bytes (e.g., "1.234" with unit "GiB")
func parseSize(value, unit string) int64 {
	val, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}

	// Normalize unit - handle both "MiB" and "MB" formats
	unit = strings.ToUpper(strings.TrimSpace(unit))
	unit = strings.TrimSuffix(unit, "B")
	unit = strings.TrimSuffix(unit, "I") // Handle MiB vs MB

	multiplier := int64(1)
	switch unit {
	case "K":
		multiplier = 1024
	case "M":
		multiplier = 1024 * 1024
	case "G":
		multiplier = 1024 * 1024 * 1024
	case "T":
		multiplier = 1024 * 1024 * 1024 * 1024
	case "P":
		multiplier = 1024 * 1024 * 1024 * 1024 * 1024
	}

	return int64(val * float64(multiplier))
}

// CopyFile copies a file from remote to local directory with progress updates via TransferManager
func CopyFile(ctx context.Context, manager *TransferManager, transferID, remote, remotePath, localDir string) error {
	src := remote + ":" + remotePath

	// Use -v (verbose) flag - this outputs "Transferred:" lines to stderr
	// Use --stats to control update frequency
	cmd := exec.CommandContext(ctx, "rclone", "copy", "-v", "--stats", "500ms", src, localDir)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	manager.Start(transferID)

	if err := cmd.Start(); err != nil {
		manager.Fail(transferID, err)
		return fmt.Errorf("failed to start rclone: %w", err)
	}

	// Parse progress output in a goroutine
	done := make(chan struct{})
	go func() {
		defer close(done)
		parseRcloneOutput(bufio.NewReader(stderr), transferID, manager)
	}()

	// Wait for command to complete
	err = cmd.Wait()

	// Wait for parsing to finish
	<-done

	if err != nil {
		manager.Fail(transferID, err)
		return err
	}

	manager.Complete(transferID)
	return nil
}

// parseRcloneOutput parses rclone stderr output to extract progress information
func parseRcloneOutput(reader *bufio.Reader, transferID string, mgr *TransferManager) {
	scanner := bufio.NewScanner(reader)

	// Increase buffer size for long lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	// Custom split function to handle both \r and \n
	// This is critical because rclone uses \r to update progress lines in place
	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}

		// Look for \r or \n
		if i := strings.IndexAny(string(data), "\r\n"); i >= 0 {
			// Return the token before the delimiter
			token = data[0:i]

			// Skip the delimiter(s) - handle both \r\n and standalone \r or \n
			advance = i + 1
			if advance < len(data) && data[i] == '\r' && data[advance] == '\n' {
				advance++ // Skip the \n after \r
			}

			return advance, token, nil
		}

		// Request more data
		if atEOF {
			return len(data), data, nil
		}

		return 0, nil, nil
	})

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Try to match progress line
		matches := statsRegex.FindStringSubmatch(line)
		if len(matches) >= 6 {
			// Parse percentage
			percentage, err := strconv.ParseFloat(matches[5], 64)
			if err == nil {
				// Parse bytes with proper unit handling
				copied := parseSize(matches[1], matches[2])
				total := parseSize(matches[3], matches[4])
				mgr.UpdateProgress(transferID, percentage, copied, total, "")
			}
		}
	}
}

// FormatSize formats a file size in human-readable format
func FormatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FormatSpeed formats a transfer speed
func FormatSpeed(bytesPerSec float64) string {
	if bytesPerSec == 0 {
		return "0 B/s"
	}
	return FormatSize(int64(bytesPerSec)) + "/s"
}
