package main

import (
	"context"
	"fmt"
	"os"

	"rcloneb/rclone"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.progressBar.Width = msg.Width - 20
		return m, nil

	case tea.KeyMsg:
		// Handle quit globally
		if key.Matches(msg, m.keys.Quit) {
			// Cancel any running transfers
			if m.transferCancel != nil {
				m.transferCancel()
			}
			return m, tea.Quit
		}

		// Clear error on any key press
		if m.err != nil {
			m.err = nil
			return m, nil
		}

		// Handle based on current state
		switch m.state {
		case StateRemoteSelect:
			return m.updateRemoteSelect(msg)
		case StateFileBrowser:
			return m.updateFileBrowser(msg)
		case StateQueueView:
			return m.updateQueueView(msg)
		case StateTransferView:
			return m.updateTransferView(msg)
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case progress.FrameMsg:
		progressModel, cmd := m.progressBar.Update(msg)
		m.progressBar = progressModel.(progress.Model)
		return m, cmd

	case remotesLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.remotes = msg.remotes
		return m, nil

	case filesLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.files = make([]BrowserItem, len(msg.files))
		for i, f := range msg.files {
			m.files[i] = BrowserItem{FileItem: f}
		}
		return m, nil

	case tickMsg:
		// Only tick while in transfer view
		if m.state != StateTransferView || m.transferMgr == nil {
			return m, nil
		}

		// Always continue ticking while in transfer view
		// This ensures the UI updates even during long transfers
		return m, tickCmd()

	}

	return m, tea.Batch(cmds...)
}

// updateRemoteSelect handles input in remote selection view
func (m Model) updateRemoteSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.selectedIndex > 0 {
			m.selectedIndex--
		}
	case key.Matches(msg, m.keys.Down):
		if m.selectedIndex < len(m.remotes)-1 {
			m.selectedIndex++
		}
	case key.Matches(msg, m.keys.Enter), key.Matches(msg, m.keys.Right):
		if len(m.remotes) > 0 {
			m.currentRemote = m.remotes[m.selectedIndex]
			m.currentPath = ""
			m.pathStack = nil
			m.state = StateFileBrowser
			m.loading = true
			m.fileIndex = 0
			return m, tea.Batch(m.loadFiles(), m.spinner.Tick)
		}
	case msg.String() == "q":
		return m, tea.Quit
	}
	return m, nil
}

// updateFileBrowser handles input in file browser view
func (m Model) updateFileBrowser(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle filter mode
	if m.filterMode {
		switch {
		case key.Matches(msg, m.keys.Escape):
			m.filterMode = false
			m.filterText = ""
			m.filterInput.SetValue("")
			m.fileIndex = 0
			return m, nil
		case msg.String() == "enter":
			m.filterMode = false
			m.filterText = m.filterInput.Value()
			m.fileIndex = 0
			return m, nil
		default:
			var cmd tea.Cmd
			m.filterInput, cmd = m.filterInput.Update(msg)
			m.filterText = m.filterInput.Value()
			m.fileIndex = 0
			return m, cmd
		}
	}

	files := m.filteredFiles()

	switch {
	case key.Matches(msg, m.keys.Up):
		if m.fileIndex > 0 {
			m.fileIndex--
		}
	case key.Matches(msg, m.keys.Down):
		if m.fileIndex < len(files)-1 {
			m.fileIndex++
		}
	case key.Matches(msg, m.keys.Enter), key.Matches(msg, m.keys.Right):
		if m.fileIndex >= 0 && m.fileIndex < len(files) {
			f := files[m.fileIndex]
			if f.IsDir {
				m.enterDirectory(f.Name)
				m.loading = true
				return m, tea.Batch(m.loadFiles(), m.spinner.Tick)
			} else {
				// Add single file to queue
				m.queue.Add(m.currentRemote, f.FileItem)
				return m, nil
			}
		}
	case key.Matches(msg, m.keys.Left), key.Matches(msg, m.keys.Back):
		if m.goBack() {
			m.loading = true
			return m, tea.Batch(m.loadFiles(), m.spinner.Tick)
		} else {
			// Go back to remote selection
			m.state = StateRemoteSelect
			m.selectedIndex = 0
		}
	case key.Matches(msg, m.keys.Select):
		if m.fileIndex >= 0 && m.fileIndex < len(files) {
			m.toggleSelection()
		}
		return m, nil
	case key.Matches(msg, m.keys.SelectAll):
		m.selectAll()
		return m, nil
	case key.Matches(msg, m.keys.Filter):
		m.filterMode = true
		m.filterInput.Focus()
		return m, nil
	case key.Matches(msg, m.keys.Escape):
		if m.filterText != "" {
			m.filterText = ""
			m.filterInput.SetValue("")
			m.fileIndex = 0
		}
	case key.Matches(msg, m.keys.Refresh):
		m.loading = true
		return m, tea.Batch(m.loadFiles(), m.spinner.Tick)
	case msg.String() == "q":
		// Add selected files to queue and go to queue view
		m.addSelectedToQueue()
		if m.queue.Len() > 0 {
			m.state = StateQueueView
			m.selectedIndex = 0
		}
		return m, nil
	}

	return m, nil
}

// updateQueueView handles input in queue view
func (m Model) updateQueueView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	items := m.queue.Items()

	switch {
	case key.Matches(msg, m.keys.Up):
		if m.selectedIndex > 0 {
			m.selectedIndex--
		}
	case key.Matches(msg, m.keys.Down):
		if m.selectedIndex < len(items)-1 {
			m.selectedIndex++
		}
	case key.Matches(msg, m.keys.Remove):
		if len(items) > 0 {
			m.queue.Remove(m.selectedIndex)
			if m.selectedIndex >= m.queue.Len() && m.selectedIndex > 0 {
				m.selectedIndex--
			}
		}
	case key.Matches(msg, m.keys.Escape):
		m.state = StateFileBrowser
		m.selectedIndex = 0
	case key.Matches(msg, m.keys.Start), msg.String() == "s":
		if m.queue.Len() > 0 {
			m.state = StateTransferView
			return m, m.startDownloads()
		}
	}

	return m, nil
}

// updateTransferView handles input in transfer view
func (m Model) updateTransferView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.transferMgr == nil {
		return m, nil
	}

	// Check if all done
	pending, inProgress, _, _ := m.transferMgr.Stats()
	allDone := pending == 0 && inProgress == 0

	if allDone {
		switch {
		case key.Matches(msg, m.keys.Enter):
			m.queue.Clear()
			m.transferMgr = nil
			m.state = StateFileBrowser
			return m, nil
		case msg.String() == "q":
			return m, tea.Quit
		}
	}

	return m, nil
}

// startDownloads initializes the transfer manager and starts downloads
func (m *Model) startDownloads() tea.Cmd {
	// Create context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	m.transferCtx = ctx
	m.transferCancel = cancel

	// Create transfer manager
	m.transferMgr = rclone.NewTransferManager()

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	// Add all queue items to transfer manager
	items := m.queue.Items()
	for i, item := range items {
		transferID := fmt.Sprintf("transfer_%d", i)
		source := item.Remote + ":" + item.Path
		m.transferMgr.Add(transferID, source, cwd, item.Size)
	}

	// Start all transfers in background goroutines
	// Each transfer runs sequentially but doesn't block the UI
	go m.runTransfers(ctx, cwd)

	// Start ticking to update the UI
	return tickCmd()
}

// runTransfers runs all transfers sequentially in a background goroutine
func (m *Model) runTransfers(ctx context.Context, cwd string) {
	items := m.queue.Items()

	for i, item := range items {
		// Check if cancelled
		select {
		case <-ctx.Done():
			return
		default:
		}

		transferID := fmt.Sprintf("transfer_%d", i)
		_ = rclone.CopyFile(ctx, m.transferMgr, transferID, item.Remote, item.Path, cwd)
	}
}
