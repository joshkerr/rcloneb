package queue

import (
	"rcloneb/rclone"
	"sync"
)

// ItemStatus represents the status of a queue item
type ItemStatus int

const (
	StatusPending ItemStatus = iota
	StatusDownloading
	StatusCompleted
	StatusError
)

// Item represents a file or directory in the download queue
type Item struct {
	Remote   string
	Path     string
	Name     string
	Size     int64
	IsDir    bool
	Status   ItemStatus
	Progress float64
	Speed    string
	Error    error
}

// Queue manages the download queue
type Queue struct {
	items []Item
	mu    sync.Mutex
}

// New creates a new download queue
func New() *Queue {
	return &Queue{
		items: make([]Item, 0),
	}
}

// Add adds a file or directory to the queue
func (q *Queue) Add(remote string, file rclone.FileItem) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Check if already in queue
	for _, item := range q.items {
		if item.Remote == remote && item.Path == file.Path {
			return
		}
	}

	q.items = append(q.items, Item{
		Remote: remote,
		Path:   file.Path,
		Name:   file.Name,
		Size:   file.Size,
		IsDir:  file.IsDir,
		Status: StatusPending,
	})
}

// Remove removes an item from the queue by index
func (q *Queue) Remove(index int) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if index >= 0 && index < len(q.items) {
		q.items = append(q.items[:index], q.items[index+1:]...)
	}
}

// Items returns a copy of the queue items
func (q *Queue) Items() []Item {
	q.mu.Lock()
	defer q.mu.Unlock()

	result := make([]Item, len(q.items))
	copy(result, q.items)
	return result
}

// Len returns the number of items in the queue
func (q *Queue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items)
}

// Clear removes all items from the queue
func (q *Queue) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.items = make([]Item, 0)
}

// UpdateProgress updates the progress of an item
func (q *Queue) UpdateProgress(path string, progress float64, speed string) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for i := range q.items {
		if q.items[i].Path == path {
			q.items[i].Progress = progress
			q.items[i].Speed = speed
			q.items[i].Status = StatusDownloading
			break
		}
	}
}

// SetStatus sets the status of an item
func (q *Queue) SetStatus(path string, status ItemStatus, err error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for i := range q.items {
		if q.items[i].Path == path {
			q.items[i].Status = status
			q.items[i].Error = err
			if status == StatusCompleted {
				q.items[i].Progress = 100
			}
			break
		}
	}
}

// GetNextPending returns the next pending item, or nil if none
func (q *Queue) GetNextPending() *Item {
	q.mu.Lock()
	defer q.mu.Unlock()

	for i := range q.items {
		if q.items[i].Status == StatusPending {
			return &q.items[i]
		}
	}
	return nil
}

// HasPending returns true if there are pending items
func (q *Queue) HasPending() bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	for _, item := range q.items {
		if item.Status == StatusPending {
			return true
		}
	}
	return false
}

// TotalSize returns the total size of all items in the queue
func (q *Queue) TotalSize() int64 {
	q.mu.Lock()
	defer q.mu.Unlock()

	var total int64
	for _, item := range q.items {
		total += item.Size
	}
	return total
}

// Contains checks if a path is already in the queue
func (q *Queue) Contains(remote, path string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	for _, item := range q.items {
		if item.Remote == remote && item.Path == path {
			return true
		}
	}
	return false
}
