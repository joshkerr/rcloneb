package main

import (
	"context"
	"time"

	"rcloneb/queue"
	"rcloneb/rclone"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// AppState represents the current view/state of the application
type AppState int

const (
	StateRemoteSelect AppState = iota
	StateFileBrowser
	StateQueueView
	StateTransferView
)

// BrowserItem extends FileItem with selection state
type BrowserItem struct {
	rclone.FileItem
	Selected bool
}

// Model represents the main application state
type Model struct {
	// Current state/view
	state AppState

	// Remotes
	remotes       []string
	selectedIndex int

	// File browser
	currentRemote string
	currentPath   string
	pathStack     []string // For back navigation
	files         []BrowserItem
	fileIndex     int

	// Filtering
	filterMode  bool
	filterInput textinput.Model
	filterText  string

	// Download queue
	queue *queue.Queue

	// Transfer management
	transferMgr    *rclone.TransferManager
	transferCtx    context.Context
	transferCancel context.CancelFunc
	progressBar    progress.Model

	// UI state
	width   int
	height  int
	loading bool
	spinner spinner.Model
	err     error

	// Keybindings
	keys KeyMap
}

// NewModel creates a new application model
func NewModel() Model {
	ti := textinput.New()
	ti.Placeholder = "Type to filter..."
	ti.Prompt = "/ "

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle

	prog := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
	)

	return Model{
		state:         StateRemoteSelect,
		queue:         queue.New(),
		filterInput:   ti,
		spinner:       s,
		progressBar:   prog,
		keys:          DefaultKeyMap(),
		selectedIndex: 0,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadRemotes(),
		m.spinner.Tick,
	)
}

// Messages for async operations

// remotesLoadedMsg is sent when remotes are loaded
type remotesLoadedMsg struct {
	remotes []string
	err     error
}

// filesLoadedMsg is sent when files are loaded
type filesLoadedMsg struct {
	files []rclone.FileItem
	err   error
}

// tickMsg is sent periodically to update the transfer UI
type tickMsg time.Time

// tickCmd returns a command that sends a tick message
func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// loadRemotes returns a command to load remotes
func (m Model) loadRemotes() tea.Cmd {
	return func() tea.Msg {
		remotes, err := rclone.ListRemotes()
		return remotesLoadedMsg{remotes: remotes, err: err}
	}
}

// loadFiles returns a command to load files at the current path
func (m Model) loadFiles() tea.Cmd {
	remote := m.currentRemote
	path := m.currentPath
	return func() tea.Msg {
		files, err := rclone.ListFiles(remote, path)
		return filesLoadedMsg{files: files, err: err}
	}
}

// filteredFiles returns files matching the current filter
func (m Model) filteredFiles() []BrowserItem {
	if m.filterText == "" {
		return m.files
	}

	var filtered []BrowserItem
	for _, f := range m.files {
		if containsIgnoreCase(f.Name, m.filterText) {
			filtered = append(filtered, f)
		}
	}
	return filtered
}

// containsIgnoreCase checks if s contains substr (case-insensitive)
func containsIgnoreCase(s, substr string) bool {
	return len(substr) == 0 ||
		(len(s) >= len(substr) && containsFold(s, substr))
}

func containsFold(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if equalFold(s[i:i+len(substr)], substr) {
			return true
		}
	}
	return false
}

func equalFold(s, t string) bool {
	if len(s) != len(t) {
		return false
	}
	for i := 0; i < len(s); i++ {
		sr := s[i]
		tr := t[i]
		if sr >= 'A' && sr <= 'Z' {
			sr += 'a' - 'A'
		}
		if tr >= 'A' && tr <= 'Z' {
			tr += 'a' - 'A'
		}
		if sr != tr {
			return false
		}
	}
	return true
}

// toggleSelection toggles selection of the current file
func (m *Model) toggleSelection() {
	files := m.filteredFiles()
	if m.fileIndex >= 0 && m.fileIndex < len(files) {
		// Find the actual index in m.files
		for i := range m.files {
			if m.files[i].Path == files[m.fileIndex].Path {
				m.files[i].Selected = !m.files[i].Selected
				break
			}
		}
	}
}

// selectAll toggles all visible files and directories
func (m *Model) selectAll() {
	files := m.filteredFiles()
	// Check if all are selected
	allSelected := true
	for _, f := range files {
		if !f.Selected {
			allSelected = false
			break
		}
	}

	// Toggle based on current state
	for _, f := range files {
		for i := range m.files {
			if m.files[i].Path == f.Path {
				m.files[i].Selected = !allSelected
				break
			}
		}
	}
}

// addSelectedToQueue adds all selected files and directories to the queue
func (m *Model) addSelectedToQueue() {
	for _, f := range m.files {
		if f.Selected {
			m.queue.Add(m.currentRemote, f.FileItem)
		}
	}
	// Clear selections
	for i := range m.files {
		m.files[i].Selected = false
	}
}

// enterDirectory enters a directory
func (m *Model) enterDirectory(dir string) {
	m.pathStack = append(m.pathStack, m.currentPath)
	if m.currentPath == "" {
		m.currentPath = dir
	} else {
		m.currentPath = m.currentPath + "/" + dir
	}
	m.fileIndex = 0
	m.filterText = ""
	m.filterInput.SetValue("")
}

// goBack navigates to the parent directory
func (m *Model) goBack() bool {
	if len(m.pathStack) > 0 {
		m.currentPath = m.pathStack[len(m.pathStack)-1]
		m.pathStack = m.pathStack[:len(m.pathStack)-1]
		m.fileIndex = 0
		m.filterText = ""
		m.filterInput.SetValue("")
		return true
	}
	return false
}
