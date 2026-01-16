package main

import (
	"fmt"
	"strings"
	"time"

	"rcloneb/rclone"
)

// View renders the current view
func (m Model) View() string {
	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error: %v\n\nPress any key to continue...", m.err))
	}

	switch m.state {
	case StateRemoteSelect:
		return m.remoteSelectView()
	case StateFileBrowser:
		return m.fileBrowserView()
	case StateQueueView:
		return m.queueView()
	case StateTransferView:
		return m.transferView()
	default:
		return "Unknown state"
	}
}

// remoteSelectView renders the remote selection view
func (m Model) remoteSelectView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("rcloneb - Select Remote"))
	b.WriteString("\n\n")

	if m.loading {
		b.WriteString(m.spinner.View())
		b.WriteString(" Loading remotes...")
		return b.String()
	}

	if len(m.remotes) == 0 {
		b.WriteString("No remotes configured. Run 'rclone config' to add one.")
		return b.String()
	}

	for i, remote := range m.remotes {
		isSelected := i == m.selectedIndex

		// Build line content with padding for bar effect
		lineContent := " " + remote
		lineWidth := m.width - 2
		if lineWidth < 40 {
			lineWidth = 40
		}
		if len(lineContent) < lineWidth {
			lineContent += strings.Repeat(" ", lineWidth-len(lineContent))
		}

		if isSelected {
			b.WriteString(selectedStyle.Render(lineContent))
		} else {
			b.WriteString(normalStyle.Render(lineContent))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k: navigate • enter: select • q: quit"))

	return b.String()
}

// fileBrowserView renders the file browser view
func (m Model) fileBrowserView() string {
	var b strings.Builder

	// Header with path
	path := m.currentRemote + ":"
	if m.currentPath != "" {
		path += m.currentPath
	}
	b.WriteString(titleStyle.Render(path))
	b.WriteString("\n")

	// Queue indicator
	if m.queue.Len() > 0 {
		b.WriteString(checkedStyle.Render(fmt.Sprintf("[%d files in queue]", m.queue.Len())))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	if m.loading {
		b.WriteString(m.spinner.View())
		b.WriteString(" Loading...")
		return b.String()
	}

	// Filter input
	if m.filterMode {
		b.WriteString(filterPromptStyle.Render("/ "))
		b.WriteString(filterTextStyle.Render(m.filterInput.View()))
		b.WriteString("\n\n")
	} else if m.filterText != "" {
		b.WriteString(filterPromptStyle.Render(fmt.Sprintf("Filter: %s", m.filterText)))
		b.WriteString("\n\n")
	}

	files := m.filteredFiles()
	if len(files) == 0 {
		if m.filterText != "" {
			b.WriteString("No matching files")
		} else {
			b.WriteString("Empty directory")
		}
		b.WriteString("\n")
	} else {
		// Calculate visible range for scrolling
		visibleLines := m.height - 10 // Account for header/footer
		if visibleLines < 5 {
			visibleLines = 10
		}

		startIdx := 0
		if m.fileIndex >= visibleLines {
			startIdx = m.fileIndex - visibleLines + 1
		}
		endIdx := startIdx + visibleLines
		if endIdx > len(files) {
			endIdx = len(files)
		}

		for i := startIdx; i < endIdx; i++ {
			f := files[i]
			isSelected := i == m.fileIndex

			// Selection checkbox
			checkbox := "[ ] "
			if f.Selected {
				checkbox = "[x] "
			}

			// File/dir name
			name := f.Name
			if f.IsDir {
				name = name + "/"
			}

			// Size
			size := ""
			if !f.IsDir {
				size = "  " + rclone.FormatSize(f.Size)
			}

			// Build the full line content
			lineContent := fmt.Sprintf(" %s%s%s", checkbox, name, size)

			// Pad line to consistent width for full bar effect
			lineWidth := m.width - 2
			if lineWidth < 40 {
				lineWidth = 40
			}
			if len(lineContent) < lineWidth {
				lineContent += strings.Repeat(" ", lineWidth-len(lineContent))
			}

			// Apply styling based on selection
			if isSelected {
				b.WriteString(selectedStyle.Render(lineContent))
			} else if f.IsDir {
				b.WriteString(dirStyle.Render(lineContent))
			} else {
				b.WriteString(fileStyle.Render(lineContent))
			}
			b.WriteString("\n")
		}

		// Scroll indicator
		if len(files) > visibleLines {
			b.WriteString(fmt.Sprintf("\n%d/%d", m.fileIndex+1, len(files)))
		}
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k: navigate • space: select • a: all • l/enter: open • h: back • q: queue • /: filter • r: refresh"))

	return b.String()
}

// queueView renders the queue view
func (m Model) queueView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Download Queue"))
	b.WriteString("\n\n")

	items := m.queue.Items()
	if len(items) == 0 {
		b.WriteString("Queue is empty\n")
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("esc: go back"))
		return b.String()
	}

	// Calculate visible range
	visibleLines := m.height - 8
	if visibleLines < 5 {
		visibleLines = 10
	}

	startIdx := 0
	if m.selectedIndex >= visibleLines {
		startIdx = m.selectedIndex - visibleLines + 1
	}
	endIdx := startIdx + visibleLines
	if endIdx > len(items) {
		endIdx = len(items)
	}

	for i := startIdx; i < endIdx; i++ {
		item := items[i]
		isSelected := i == m.selectedIndex

		name := item.Name
		if item.IsDir {
			name = name + "/"
		}

		var sizeStr string
		if item.IsDir {
			sizeStr = "[folder]"
		} else {
			sizeStr = rclone.FormatSize(item.Size)
		}

		// Build line content
		lineContent := fmt.Sprintf(" %s  %s  (%s)", name, sizeStr, item.Remote)

		// Pad line for bar effect
		lineWidth := m.width - 2
		if lineWidth < 40 {
			lineWidth = 40
		}
		if len(lineContent) < lineWidth {
			lineContent += strings.Repeat(" ", lineWidth-len(lineContent))
		}

		if isSelected {
			b.WriteString(selectedStyle.Render(lineContent))
		} else {
			b.WriteString(normalStyle.Render(lineContent))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Total: %d files, %s\n", len(items), rclone.FormatSize(m.queue.TotalSize())))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k: navigate • d/x: remove • s: start download • esc: go back"))

	return b.String()
}

// transferView renders the transfer progress view
func (m Model) transferView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Downloading..."))
	b.WriteString("\n\n")

	if m.transferMgr == nil {
		b.WriteString("Initializing transfers...\n")
		return b.String()
	}

	// Get stats
	pending, inProgress, completed, failed := m.transferMgr.Stats()
	statsLine := fmt.Sprintf("Pending: %d | Active: %d | Done: %d | Failed: %d",
		pending, inProgress, completed, failed)
	b.WriteString(statsLine)
	b.WriteString("\n\n")

	transfers := m.transferMgr.GetAll()
	if len(transfers) == 0 {
		b.WriteString("No transfers in queue\n")
		return b.String()
	}

	// Show in-progress transfers first
	for _, t := range transfers {
		if t.Status == rclone.StatusInProgress {
			b.WriteString(m.renderTransfer(t))
		}
	}

	// Then pending
	for _, t := range transfers {
		if t.Status == rclone.StatusPending {
			b.WriteString(m.renderTransfer(t))
		}
	}

	// Then completed
	for _, t := range transfers {
		if t.Status == rclone.StatusCompleted {
			b.WriteString(m.renderTransfer(t))
		}
	}

	// Then failed
	for _, t := range transfers {
		if t.Status == rclone.StatusFailed {
			b.WriteString(m.renderTransfer(t))
		}
	}

	b.WriteString("\n")

	// Check if all done
	allDone := pending == 0 && inProgress == 0

	if allDone {
		if failed == 0 {
			b.WriteString(successStyle.Render("All downloads complete!"))
		} else {
			b.WriteString(errorStyle.Render(fmt.Sprintf("Downloads complete with %d error(s)", failed)))
		}
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("enter: continue browsing • q: quit"))
	} else {
		b.WriteString(helpStyle.Render("Downloads in progress... ctrl+c: cancel"))
	}

	return b.String()
}

// renderTransfer renders a single transfer with progress bar
func (m Model) renderTransfer(t *rclone.Transfer) string {
	var b strings.Builder

	// Extract filename from source path
	parts := strings.Split(t.Source, "/")
	filename := parts[len(parts)-1]
	if len(filename) > 40 {
		filename = filename[:37] + "..."
	}

	// Status indicator
	var statusPrefix string
	var style = normalStyle
	switch t.Status {
	case rclone.StatusPending:
		statusPrefix = "[PENDING] "
		style = normalStyle
	case rclone.StatusInProgress:
		statusPrefix = "[ACTIVE]  "
		style = selectedStyle
	case rclone.StatusCompleted:
		statusPrefix = successStyle.Render("[DONE]    ")
		style = successStyle
	case rclone.StatusFailed:
		statusPrefix = errorStyle.Render("[FAILED]  ")
		style = errorStyle
	}

	// First line: status + filename
	b.WriteString(fmt.Sprintf("%s%s\n", statusPrefix, style.Render(filename)))

	// Progress bar for in-progress transfers
	if t.Status == rclone.StatusInProgress {
		// Calculate progress bar width based on terminal width
		barWidth := m.width - 25
		if barWidth < 20 {
			barWidth = 20
		}
		if barWidth > 50 {
			barWidth = 50
		}

		// Use the bubbles progress bar or simple ASCII
		progress := t.Progress / 100.0
		if progress > 1 {
			progress = 1
		}
		if progress < 0 {
			progress = 0
		}

		filled := int(float64(barWidth) * progress)
		empty := barWidth - filled

		bar := progressBarStyle.Render(strings.Repeat("█", filled)) +
			strings.Repeat("░", empty)

		b.WriteString(fmt.Sprintf("   [%s] %.0f%%\n", bar, t.Progress))

		// Stats line: bytes transferred, speed
		if t.BytesTotal > 0 {
			stats := fmt.Sprintf("   %s / %s",
				rclone.FormatSize(t.BytesCopied),
				rclone.FormatSize(t.BytesTotal))
			if t.Speed != "" {
				stats += fmt.Sprintf(" @ %s", t.Speed)
			}
			b.WriteString(helpStyle.Render(stats))
			b.WriteString("\n")
		} else if t.Speed != "" {
			b.WriteString(helpStyle.Render(fmt.Sprintf("   %s", t.Speed)))
			b.WriteString("\n")
		}
	}

	// Completed: show duration
	if t.Status == rclone.StatusCompleted && !t.EndTime.IsZero() {
		duration := t.EndTime.Sub(t.StartTime).Round(time.Millisecond)
		b.WriteString(helpStyle.Render(fmt.Sprintf("   Completed in %v", duration)))
		b.WriteString("\n")
	}

	// Failed: show error
	if t.Status == rclone.StatusFailed && t.Error != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("   Error: %v", t.Error)))
		b.WriteString("\n")
	}

	return b.String()
}
