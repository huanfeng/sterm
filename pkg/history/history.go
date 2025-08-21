// Package history provides communication history management functionality
package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Direction represents the direction of data flow
type Direction int

const (
	DirectionInput Direction = iota
	DirectionOutput
)

// String returns the string representation of Direction
func (d Direction) String() string {
	switch d {
	case DirectionInput:
		return "input"
	case DirectionOutput:
		return "output"
	default:
		return "unknown"
	}
}

// FileFormat represents different file export formats
type FileFormat int

const (
	FormatPlainText FileFormat = iota
	FormatTimestamped
	FormatJSON
)

// String returns the string representation of FileFormat
func (f FileFormat) String() string {
	switch f {
	case FormatPlainText:
		return "plain_text"
	case FormatTimestamped:
		return "timestamped"
	case FormatJSON:
		return "json"
	default:
		return "unknown"
	}
}

// HistoryManager interface defines the contract for history operations
type HistoryManager interface {
	Write(data []byte, direction Direction) error
	Read(offset, length int) ([]byte, error)
	GetSize() int
	GetEntryCount() int
	SaveToFile(filename string, format FileFormat) error
	Clear() error
	SetMaxSize(size int) error
	GetMaxSize() int
	GetEntries(start, count int) ([]HistoryEntry, error)
}

// HistoryEntry represents a single entry in the communication history
type HistoryEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Direction Direction `json:"direction"`
	Data      []byte    `json:"data"`
	Length    int       `json:"length"`
}

// Validate checks if the history entry is valid
func (h HistoryEntry) Validate() error {
	if h.Timestamp.IsZero() {
		return fmt.Errorf("timestamp cannot be zero")
	}
	
	if h.Direction != DirectionInput && h.Direction != DirectionOutput {
		return fmt.Errorf("invalid direction: %d", h.Direction)
	}
	
	if h.Data == nil {
		return fmt.Errorf("data cannot be nil")
	}
	
	if h.Length != len(h.Data) {
		return fmt.Errorf("length mismatch: expected %d, got %d", len(h.Data), h.Length)
	}
	
	return nil
}

// NewHistoryEntry creates a new history entry with current timestamp
func NewHistoryEntry(data []byte, direction Direction) HistoryEntry {
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)
	return HistoryEntry{
		Timestamp: time.Now(),
		Direction: direction,
		Data:      dataCopy,
		Length:    len(data),
	}
}

// HistoryStats provides statistics about the history buffer
type HistoryStats struct {
	TotalEntries   int           `json:"total_entries"`
	TotalBytes     int           `json:"total_bytes"`
	InputEntries   int           `json:"input_entries"`
	OutputEntries  int           `json:"output_entries"`
	InputBytes     int           `json:"input_bytes"`
	OutputBytes    int           `json:"output_bytes"`
	MaxSize        int           `json:"max_size"`
	CurrentSize    int           `json:"current_size"`
	OldestEntry    *time.Time    `json:"oldest_entry,omitempty"`
	NewestEntry    *time.Time    `json:"newest_entry,omitempty"`
}

// RingBufferHistoryManager implements HistoryManager using a ring buffer
type RingBufferHistoryManager struct {
	buffer      []byte
	maxSize     int
	writePos    int
	readPos     int
	size        int
	entries     []HistoryEntry
	maxEntries  int
	entryCount  int
	entryStart  int
}

// NewRingBufferHistoryManager creates a new ring buffer history manager
func NewRingBufferHistoryManager(maxSize int) *RingBufferHistoryManager {
	if maxSize <= 0 {
		maxSize = 10 * 1024 * 1024 // Default 10MB
	}
	
	maxEntries := maxSize / 10 // Estimate 10 bytes per entry on average
	if maxEntries < 1000 {
		maxEntries = 1000 // Minimum 1000 entries
	}
	
	return &RingBufferHistoryManager{
		buffer:     make([]byte, maxSize),
		maxSize:    maxSize,
		maxEntries: maxEntries,
		entries:    make([]HistoryEntry, maxEntries),
		writePos:   0,
		readPos:    0,
		size:       0,
		entryCount: 0,
		entryStart: 0,
	}
}

// Write adds data to the history buffer
func (rbhm *RingBufferHistoryManager) Write(data []byte, direction Direction) error {
	if data == nil {
		return fmt.Errorf("data cannot be nil")
	}
	
	if direction != DirectionInput && direction != DirectionOutput {
		return fmt.Errorf("invalid direction: %d", direction)
	}
	
	// Create history entry
	entry := NewHistoryEntry(data, direction)
	
	// Add entry to entries ring buffer
	rbhm.entries[rbhm.entryStart] = entry
	rbhm.entryStart = (rbhm.entryStart + 1) % rbhm.maxEntries
	
	if rbhm.entryCount < rbhm.maxEntries {
		rbhm.entryCount++
	}
	
	// Add data to byte buffer
	dataLen := len(data)
	
	// Check if we need to wrap around or make space
	for rbhm.size+dataLen > rbhm.maxSize {
		// Remove oldest data to make space
		if rbhm.size == 0 {
			break // Buffer is empty, can't remove more
		}
		
		// Move read position forward to free space
		oldSize := rbhm.size
		rbhm.readPos = (rbhm.readPos + 1) % rbhm.maxSize
		rbhm.size--
		
		// Prevent infinite loop
		if rbhm.size == oldSize {
			break
		}
	}
	
	// Write data to buffer
	for _, b := range data {
		if rbhm.size < rbhm.maxSize {
			rbhm.buffer[rbhm.writePos] = b
			rbhm.writePos = (rbhm.writePos + 1) % rbhm.maxSize
			rbhm.size++
		} else {
			// Buffer is full, overwrite oldest data
			rbhm.buffer[rbhm.writePos] = b
			rbhm.writePos = (rbhm.writePos + 1) % rbhm.maxSize
			rbhm.readPos = (rbhm.readPos + 1) % rbhm.maxSize
		}
	}
	
	return nil
}

// Read reads data from the history buffer
func (rbhm *RingBufferHistoryManager) Read(offset, length int) ([]byte, error) {
	if offset < 0 {
		return nil, fmt.Errorf("offset cannot be negative")
	}
	
	if length < 0 {
		return nil, fmt.Errorf("length cannot be negative")
	}
	
	if offset >= rbhm.size {
		return []byte{}, nil // Return empty slice if offset is beyond data
	}
	
	// Adjust length if it would read beyond available data
	if offset+length > rbhm.size {
		length = rbhm.size - offset
	}
	
	result := make([]byte, length)
	
	for i := 0; i < length; i++ {
		pos := (rbhm.readPos + offset + i) % rbhm.maxSize
		result[i] = rbhm.buffer[pos]
	}
	
	return result, nil
}

// GetSize returns the current size of data in the buffer
func (rbhm *RingBufferHistoryManager) GetSize() int {
	return rbhm.size
}

// GetEntryCount returns the number of entries in the history
func (rbhm *RingBufferHistoryManager) GetEntryCount() int {
	return rbhm.entryCount
}

// SaveToFile saves the history to a file in the specified format
func (rbhm *RingBufferHistoryManager) SaveToFile(filename string, format FileFormat) error {
	if filename == "" {
		return fmt.Errorf("filename cannot be empty")
	}
	
	entries, err := rbhm.GetEntries(0, rbhm.entryCount)
	if err != nil {
		return fmt.Errorf("failed to get entries: %w", err)
	}
	
	return saveEntriesToFile(entries, filename, format)
}

// Clear clears all data from the history buffer
func (rbhm *RingBufferHistoryManager) Clear() error {
	rbhm.writePos = 0
	rbhm.readPos = 0
	rbhm.size = 0
	rbhm.entryCount = 0
	rbhm.entryStart = 0
	
	// Clear the buffers
	for i := range rbhm.buffer {
		rbhm.buffer[i] = 0
	}
	
	for i := range rbhm.entries {
		rbhm.entries[i] = HistoryEntry{}
	}
	
	return nil
}

// SetMaxSize sets the maximum size of the buffer
func (rbhm *RingBufferHistoryManager) SetMaxSize(size int) error {
	if size <= 0 {
		return fmt.Errorf("size must be positive")
	}
	
	if size == rbhm.maxSize {
		return nil // No change needed
	}
	
	// Create new buffer
	newBuffer := make([]byte, size)
	newMaxEntries := size / 10
	if newMaxEntries < 1000 {
		newMaxEntries = 1000
	}
	newEntries := make([]HistoryEntry, newMaxEntries)
	
	// Copy existing data if new buffer is larger
	if size > rbhm.maxSize {
		// Copy all existing data
		for i := 0; i < rbhm.size; i++ {
			pos := (rbhm.readPos + i) % rbhm.maxSize
			newBuffer[i] = rbhm.buffer[pos]
		}
		
		// Copy existing entries
		copyCount := rbhm.entryCount
		if copyCount > newMaxEntries {
			copyCount = newMaxEntries
		}
		
		for i := 0; i < copyCount; i++ {
			entryPos := (rbhm.entryStart - rbhm.entryCount + i + rbhm.maxEntries) % rbhm.maxEntries
			newEntries[i] = rbhm.entries[entryPos]
		}
		
		rbhm.readPos = 0
		rbhm.writePos = rbhm.size
		rbhm.entryStart = copyCount
		rbhm.entryCount = copyCount
	} else {
		// New buffer is smaller, copy what fits
		copySize := size
		if copySize > rbhm.size {
			copySize = rbhm.size
		}
		
		// Copy most recent data
		startOffset := rbhm.size - copySize
		for i := 0; i < copySize; i++ {
			pos := (rbhm.readPos + startOffset + i) % rbhm.maxSize
			newBuffer[i] = rbhm.buffer[pos]
		}
		
		// Copy most recent entries
		copyCount := newMaxEntries
		if copyCount > rbhm.entryCount {
			copyCount = rbhm.entryCount
		}
		
		startEntry := rbhm.entryCount - copyCount
		for i := 0; i < copyCount; i++ {
			entryPos := (rbhm.entryStart - rbhm.entryCount + startEntry + i + rbhm.maxEntries) % rbhm.maxEntries
			newEntries[i] = rbhm.entries[entryPos]
		}
		
		rbhm.readPos = 0
		rbhm.writePos = copySize
		rbhm.size = copySize
		rbhm.entryStart = copyCount
		rbhm.entryCount = copyCount
	}
	
	rbhm.buffer = newBuffer
	rbhm.maxSize = size
	rbhm.entries = newEntries
	rbhm.maxEntries = newMaxEntries
	
	return nil
}

// GetMaxSize returns the maximum size of the buffer
func (rbhm *RingBufferHistoryManager) GetMaxSize() int {
	return rbhm.maxSize
}

// GetEntries returns a slice of history entries
func (rbhm *RingBufferHistoryManager) GetEntries(start, count int) ([]HistoryEntry, error) {
	if start < 0 {
		return nil, fmt.Errorf("start cannot be negative")
	}
	
	if count < 0 {
		return nil, fmt.Errorf("count cannot be negative")
	}
	
	if start >= rbhm.entryCount {
		return []HistoryEntry{}, nil
	}
	
	// Adjust count if it would read beyond available entries
	if start+count > rbhm.entryCount {
		count = rbhm.entryCount - start
	}
	
	result := make([]HistoryEntry, count)
	
	for i := 0; i < count; i++ {
		entryPos := (rbhm.entryStart - rbhm.entryCount + start + i + rbhm.maxEntries) % rbhm.maxEntries
		result[i] = rbhm.entries[entryPos]
	}
	
	return result, nil
}

// GetStats returns statistics about the history buffer
func (rbhm *RingBufferHistoryManager) GetStats() HistoryStats {
	stats := HistoryStats{
		TotalEntries: rbhm.entryCount,
		TotalBytes:   rbhm.size,
		MaxSize:      rbhm.maxSize,
		CurrentSize:  rbhm.size,
	}
	
	// Calculate input/output statistics
	for i := 0; i < rbhm.entryCount; i++ {
		entryPos := (rbhm.entryStart - rbhm.entryCount + i + rbhm.maxEntries) % rbhm.maxEntries
		entry := rbhm.entries[entryPos]
		
		if entry.Direction == DirectionInput {
			stats.InputEntries++
			stats.InputBytes += entry.Length
		} else if entry.Direction == DirectionOutput {
			stats.OutputEntries++
			stats.OutputBytes += entry.Length
		}
		
		// Track oldest and newest entries
		if i == 0 || (stats.OldestEntry != nil && entry.Timestamp.Before(*stats.OldestEntry)) {
			stats.OldestEntry = &entry.Timestamp
		}
		
		if i == 0 || (stats.NewestEntry != nil && entry.Timestamp.After(*stats.NewestEntry)) {
			stats.NewestEntry = &entry.Timestamp
		}
	}
	
	return stats
}

// saveEntriesToFile saves history entries to a file in the specified format
func saveEntriesToFile(entries []HistoryEntry, filename string, format FileFormat) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()
	
	switch format {
	case FormatPlainText:
		return saveAsPlainText(file, entries)
	case FormatTimestamped:
		return saveAsTimestamped(file, entries)
	case FormatJSON:
		return saveAsJSON(file, entries)
	default:
		return fmt.Errorf("unsupported format: %v", format)
	}
}

// saveAsPlainText saves entries as plain text
func saveAsPlainText(file *os.File, entries []HistoryEntry) error {
	for _, entry := range entries {
		if _, err := file.Write(entry.Data); err != nil {
			return fmt.Errorf("failed to write data: %w", err)
		}
	}
	return nil
}

// saveAsTimestamped saves entries with timestamps
func saveAsTimestamped(file *os.File, entries []HistoryEntry) error {
	for _, entry := range entries {
		direction := "<<"
		if entry.Direction == DirectionOutput {
			direction = ">>"
		}
		
		line := fmt.Sprintf("[%s] %s %s\n", 
			entry.Timestamp.Format("2006-01-02 15:04:05.000"), 
			direction, 
			strings.ReplaceAll(string(entry.Data), "\n", "\\n"))
		
		if _, err := file.WriteString(line); err != nil {
			return fmt.Errorf("failed to write timestamped data: %w", err)
		}
	}
	return nil
}

// saveAsJSON saves entries as JSON
func saveAsJSON(file *os.File, entries []HistoryEntry) error {
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	
	data := struct {
		Entries []HistoryEntry `json:"entries"`
		Count   int            `json:"count"`
	}{
		Entries: entries,
		Count:   len(entries),
	}
	
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}
	
	return nil
}

// MemoryHistoryManager implements HistoryManager using simple in-memory storage
// This is a simpler alternative to RingBufferHistoryManager for smaller datasets
type MemoryHistoryManager struct {
	entries []HistoryEntry
	maxSize int
	maxEntries int
}

// NewMemoryHistoryManager creates a new memory-based history manager
func NewMemoryHistoryManager(maxSize int) *MemoryHistoryManager {
	if maxSize <= 0 {
		maxSize = 10 * 1024 * 1024 // Default 10MB
	}
	
	return &MemoryHistoryManager{
		entries:    make([]HistoryEntry, 0),
		maxSize:    maxSize,
		maxEntries: maxSize / 10, // Estimate 10 bytes per entry
	}
}

// Write adds data to the memory history
func (mhm *MemoryHistoryManager) Write(data []byte, direction Direction) error {
	if data == nil {
		return fmt.Errorf("data cannot be nil")
	}
	
	if direction != DirectionInput && direction != DirectionOutput {
		return fmt.Errorf("invalid direction: %d", direction)
	}
	
	entry := NewHistoryEntry(data, direction)
	
	// Check if we need to remove old entries
	currentSize := mhm.calculateTotalSize()
	for currentSize + len(data) > mhm.maxSize && len(mhm.entries) > 0 {
		// Remove oldest entry
		removed := mhm.entries[0]
		mhm.entries = mhm.entries[1:]
		currentSize -= len(removed.Data)
	}
	
	// Check entry count limit
	if len(mhm.entries) >= mhm.maxEntries {
		// Remove oldest entries to make room
		removeCount := len(mhm.entries) - mhm.maxEntries + 1
		mhm.entries = mhm.entries[removeCount:]
	}
	
	mhm.entries = append(mhm.entries, entry)
	return nil
}

// Read reads data from the memory history
func (mhm *MemoryHistoryManager) Read(offset, length int) ([]byte, error) {
	if offset < 0 {
		return nil, fmt.Errorf("offset cannot be negative")
	}
	
	if length < 0 {
		return nil, fmt.Errorf("length cannot be negative")
	}
	
	// Concatenate all data
	var allData []byte
	for _, entry := range mhm.entries {
		allData = append(allData, entry.Data...)
	}
	
	if offset >= len(allData) {
		return []byte{}, nil
	}
	
	end := offset + length
	if end > len(allData) {
		end = len(allData)
	}
	
	return allData[offset:end], nil
}

// GetSize returns the total size of data in memory
func (mhm *MemoryHistoryManager) GetSize() int {
	return mhm.calculateTotalSize()
}

// GetEntryCount returns the number of entries
func (mhm *MemoryHistoryManager) GetEntryCount() int {
	return len(mhm.entries)
}

// SaveToFile saves the history to a file
func (mhm *MemoryHistoryManager) SaveToFile(filename string, format FileFormat) error {
	if filename == "" {
		return fmt.Errorf("filename cannot be empty")
	}
	
	return saveEntriesToFile(mhm.entries, filename, format)
}

// Clear clears all entries
func (mhm *MemoryHistoryManager) Clear() error {
	mhm.entries = mhm.entries[:0]
	return nil
}

// SetMaxSize sets the maximum size
func (mhm *MemoryHistoryManager) SetMaxSize(size int) error {
	if size <= 0 {
		return fmt.Errorf("size must be positive")
	}
	
	mhm.maxSize = size
	mhm.maxEntries = size / 10
	
	// Remove entries if current size exceeds new limit
	currentSize := mhm.calculateTotalSize()
	for currentSize > size && len(mhm.entries) > 0 {
		removed := mhm.entries[0]
		mhm.entries = mhm.entries[1:]
		currentSize -= len(removed.Data)
	}
	
	return nil
}

// GetMaxSize returns the maximum size
func (mhm *MemoryHistoryManager) GetMaxSize() int {
	return mhm.maxSize
}

// GetEntries returns a slice of entries
func (mhm *MemoryHistoryManager) GetEntries(start, count int) ([]HistoryEntry, error) {
	if start < 0 {
		return nil, fmt.Errorf("start cannot be negative")
	}
	
	if count < 0 {
		return nil, fmt.Errorf("count cannot be negative")
	}
	
	if start >= len(mhm.entries) {
		return []HistoryEntry{}, nil
	}
	
	end := start + count
	if end > len(mhm.entries) {
		end = len(mhm.entries)
	}
	
	// Return a copy of the entries
	result := make([]HistoryEntry, end-start)
	copy(result, mhm.entries[start:end])
	
	return result, nil
}

// calculateTotalSize calculates the total size of all data
func (mhm *MemoryHistoryManager) calculateTotalSize() int {
	total := 0
	for _, entry := range mhm.entries {
		total += len(entry.Data)
	}
	return total
}

// PersistentHistoryManager extends HistoryManager with automatic persistence features
type PersistentHistoryManager struct {
	HistoryManager
	backupDir       string
	autoBackup      bool
	backupInterval  time.Duration
	maxBackupFiles  int
	tempFilePrefix  string
	lastBackupTime  time.Time
}

// NewPersistentHistoryManager creates a new persistent history manager
func NewPersistentHistoryManager(baseManager HistoryManager, backupDir string) *PersistentHistoryManager {
	return &PersistentHistoryManager{
		HistoryManager:  baseManager,
		backupDir:       backupDir,
		autoBackup:      true,
		backupInterval:  time.Hour, // Default backup every hour
		maxBackupFiles:  24,        // Keep 24 backup files (24 hours)
		tempFilePrefix:  "history_temp_",
		lastBackupTime:  time.Now(),
	}
}

// SetAutoBackup enables or disables automatic backup
func (phm *PersistentHistoryManager) SetAutoBackup(enabled bool, interval time.Duration, maxFiles int) {
	phm.autoBackup = enabled
	phm.backupInterval = interval
	phm.maxBackupFiles = maxFiles
}

// Write wraps the base Write method and triggers backup if needed
func (phm *PersistentHistoryManager) Write(data []byte, direction Direction) error {
	err := phm.HistoryManager.Write(data, direction)
	if err != nil {
		return err
	}
	
	// Check if backup is needed
	if phm.autoBackup && time.Since(phm.lastBackupTime) >= phm.backupInterval {
		go phm.performAutoBackup() // Async backup to avoid blocking
	}
	
	return nil
}

// performAutoBackup performs automatic backup in background
func (phm *PersistentHistoryManager) performAutoBackup() {
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("history_backup_%s.json", timestamp)
	backupPath := filepath.Join(phm.backupDir, filename)
	
	// Create backup directory if it doesn't exist
	if err := os.MkdirAll(phm.backupDir, 0755); err != nil {
		return // Silently fail for auto backup
	}
	
	// Save current history
	if err := phm.HistoryManager.SaveToFile(backupPath, FormatJSON); err != nil {
		return // Silently fail for auto backup
	}
	
	phm.lastBackupTime = time.Now()
	
	// Clean up old backup files
	phm.cleanupOldBackups()
}

// cleanupOldBackups removes old backup files to maintain the limit
func (phm *PersistentHistoryManager) cleanupOldBackups() {
	files, err := os.ReadDir(phm.backupDir)
	if err != nil {
		return
	}
	
	// Filter backup files and sort by modification time
	var backupFiles []os.FileInfo
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "history_backup_") && strings.HasSuffix(file.Name(), ".json") {
			info, err := file.Info()
			if err == nil {
				backupFiles = append(backupFiles, info)
			}
		}
	}
	
	// Remove excess files (keep only maxBackupFiles)
	if len(backupFiles) > phm.maxBackupFiles {
		// Sort by modification time (oldest first)
		for i := 0; i < len(backupFiles)-1; i++ {
			for j := i + 1; j < len(backupFiles); j++ {
				if backupFiles[i].ModTime().After(backupFiles[j].ModTime()) {
					backupFiles[i], backupFiles[j] = backupFiles[j], backupFiles[i]
				}
			}
		}
		
		// Remove oldest files
		filesToRemove := len(backupFiles) - phm.maxBackupFiles
		for i := 0; i < filesToRemove; i++ {
			filePath := filepath.Join(phm.backupDir, backupFiles[i].Name())
			os.Remove(filePath)
		}
	}
}

// CreateTempFile creates a temporary file for large data operations
func (phm *PersistentHistoryManager) CreateTempFile() (*os.File, error) {
	tempDir := os.TempDir()
	tempFile, err := os.CreateTemp(tempDir, phm.tempFilePrefix)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	
	return tempFile, nil
}

// SaveToTempFile saves history to a temporary file and returns the file path
func (phm *PersistentHistoryManager) SaveToTempFile(format FileFormat) (string, error) {
	tempFile, err := phm.CreateTempFile()
	if err != nil {
		return "", err
	}
	defer tempFile.Close()
	
	tempPath := tempFile.Name()
	
	err = phm.HistoryManager.SaveToFile(tempPath, format)
	if err != nil {
		os.Remove(tempPath) // Clean up on error
		return "", fmt.Errorf("failed to save to temp file: %w", err)
	}
	
	return tempPath, nil
}

// LoadFromFile loads history from a file (for restoration)
func (phm *PersistentHistoryManager) LoadFromFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}
	
	// Try to parse as JSON first
	var historyData struct {
		Entries []HistoryEntry `json:"entries"`
	}
	
	if err := json.Unmarshal(data, &historyData); err != nil {
		return fmt.Errorf("failed to parse history file: %w", err)
	}
	
	// Clear current history and load entries
	if err := phm.HistoryManager.Clear(); err != nil {
		return fmt.Errorf("failed to clear current history: %w", err)
	}
	
	for _, entry := range historyData.Entries {
		if err := phm.HistoryManager.Write(entry.Data, entry.Direction); err != nil {
			return fmt.Errorf("failed to restore entry: %w", err)
		}
	}
	
	return nil
}

// GetBackupFiles returns a list of available backup files
func (phm *PersistentHistoryManager) GetBackupFiles() ([]string, error) {
	files, err := os.ReadDir(phm.backupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}
	
	var backupFiles []string
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "history_backup_") && strings.HasSuffix(file.Name(), ".json") {
			backupFiles = append(backupFiles, file.Name())
		}
	}
	
	return backupFiles, nil
}

// RestoreFromBackup restores history from a specific backup file
func (phm *PersistentHistoryManager) RestoreFromBackup(backupFilename string) error {
	backupPath := filepath.Join(phm.backupDir, backupFilename)
	return phm.LoadFromFile(backupPath)
}

// GetMemoryUsage returns current memory usage statistics
func (phm *PersistentHistoryManager) GetMemoryUsage() MemoryUsage {
	return MemoryUsage{
		CurrentSize:    phm.HistoryManager.GetSize(),
		MaxSize:        phm.HistoryManager.GetMaxSize(),
		EntryCount:     phm.HistoryManager.GetEntryCount(),
		UsagePercent:   float64(phm.HistoryManager.GetSize()) / float64(phm.HistoryManager.GetMaxSize()) * 100,
		AutoBackup:     phm.autoBackup,
		LastBackupTime: phm.lastBackupTime,
	}
}

// MemoryUsage provides information about memory usage
type MemoryUsage struct {
	CurrentSize    int       `json:"current_size"`
	MaxSize        int       `json:"max_size"`
	EntryCount     int       `json:"entry_count"`
	UsagePercent   float64   `json:"usage_percent"`
	AutoBackup     bool      `json:"auto_backup"`
	LastBackupTime time.Time `json:"last_backup_time"`
}

// CompactHistory removes old entries to free memory while preserving recent data
func (phm *PersistentHistoryManager) CompactHistory(targetSizePercent float64) error {
	if targetSizePercent <= 0 || targetSizePercent >= 100 {
		return fmt.Errorf("target size percent must be between 0 and 100")
	}
	
	currentSize := phm.HistoryManager.GetSize()
	maxSize := phm.HistoryManager.GetMaxSize()
	targetSize := int(float64(maxSize) * targetSizePercent / 100)
	
	if currentSize <= targetSize {
		return nil // No compaction needed
	}
	
	// Create backup before compaction
	if phm.autoBackup {
		phm.performAutoBackup()
	}
	
	// For ring buffer, we can adjust the size
	if rbhm, ok := phm.HistoryManager.(*RingBufferHistoryManager); ok {
		return rbhm.SetMaxSize(targetSize)
	}
	
	// For memory manager, we need to remove old entries
	if mhm, ok := phm.HistoryManager.(*MemoryHistoryManager); ok {
		return mhm.SetMaxSize(targetSize)
	}
	
	return fmt.Errorf("compaction not supported for this history manager type")
}