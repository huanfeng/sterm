package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDirection_String(t *testing.T) {
	tests := []struct {
		direction Direction
		expected  string
	}{
		{DirectionInput, "input"},
		{DirectionOutput, "output"},
		{Direction(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.direction.String(); got != tt.expected {
				t.Errorf("Direction.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFileFormat_String(t *testing.T) {
	tests := []struct {
		format   FileFormat
		expected string
	}{
		{FormatPlainText, "plain_text"},
		{FormatTimestamped, "timestamped"},
		{FormatJSON, "json"},
		{FileFormat(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.format.String(); got != tt.expected {
				t.Errorf("FileFormat.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestHistoryEntry_Validate(t *testing.T) {
	now := time.Now()
	testData := []byte("test data")

	tests := []struct {
		name    string
		entry   HistoryEntry
		wantErr bool
	}{
		{
			name: "valid entry",
			entry: HistoryEntry{
				Timestamp: now,
				Direction: DirectionInput,
				Data:      testData,
				Length:    len(testData),
			},
			wantErr: false,
		},
		{
			name: "zero timestamp",
			entry: HistoryEntry{
				Timestamp: time.Time{},
				Direction: DirectionInput,
				Data:      testData,
				Length:    len(testData),
			},
			wantErr: true,
		},
		{
			name: "invalid direction",
			entry: HistoryEntry{
				Timestamp: now,
				Direction: Direction(999),
				Data:      testData,
				Length:    len(testData),
			},
			wantErr: true,
		},
		{
			name: "nil data",
			entry: HistoryEntry{
				Timestamp: now,
				Direction: DirectionInput,
				Data:      nil,
				Length:    0,
			},
			wantErr: true,
		},
		{
			name: "length mismatch",
			entry: HistoryEntry{
				Timestamp: now,
				Direction: DirectionInput,
				Data:      testData,
				Length:    len(testData) + 1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.entry.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("HistoryEntry.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewHistoryEntry(t *testing.T) {
	testData := []byte("test data")
	direction := DirectionOutput

	entry := NewHistoryEntry(testData, direction)

	if entry.Direction != direction {
		t.Errorf("NewHistoryEntry() Direction = %v, want %v", entry.Direction, direction)
	}

	if entry.Length != len(testData) {
		t.Errorf("NewHistoryEntry() Length = %d, want %d", entry.Length, len(testData))
	}

	if len(entry.Data) != len(testData) {
		t.Errorf("NewHistoryEntry() Data length = %d, want %d", len(entry.Data), len(testData))
	}

	if entry.Timestamp.IsZero() {
		t.Error("NewHistoryEntry() should set a non-zero timestamp")
	}

	// Verify data is copied, not referenced
	testData[0] = 'X'
	if entry.Data[0] == 'X' {
		t.Error("NewHistoryEntry() should copy data, not reference it")
	}

	if err := entry.Validate(); err != nil {
		t.Errorf("NewHistoryEntry() should create valid entry: %v", err)
	}
}

func TestNewRingBufferHistoryManager(t *testing.T) {
	maxSize := 1024
	manager := NewRingBufferHistoryManager(maxSize)

	if manager == nil {
		t.Error("NewRingBufferHistoryManager() returned nil")
	}

	if manager.maxSize != maxSize {
		t.Errorf("NewRingBufferHistoryManager() maxSize = %d, want %d", manager.maxSize, maxSize)
	}

	if len(manager.buffer) != maxSize {
		t.Errorf("NewRingBufferHistoryManager() buffer length = %d, want %d", len(manager.buffer), maxSize)
	}

	// The implementation uses minimum 1000 entries, so for small buffers it will be 1000
	expectedMaxEntries := 1000 // Minimum entries as per implementation
	if manager.maxEntries != expectedMaxEntries {
		t.Errorf("NewRingBufferHistoryManager() maxEntries = %d, want %d", manager.maxEntries, expectedMaxEntries)
	}

	if len(manager.entries) != expectedMaxEntries {
		t.Errorf("NewRingBufferHistoryManager() entries length = %d, want %d", len(manager.entries), expectedMaxEntries)
	}

	if manager.writePos != 0 {
		t.Errorf("NewRingBufferHistoryManager() writePos = %d, want 0", manager.writePos)
	}

	if manager.readPos != 0 {
		t.Errorf("NewRingBufferHistoryManager() readPos = %d, want 0", manager.readPos)
	}

	if manager.size != 0 {
		t.Errorf("NewRingBufferHistoryManager() size = %d, want 0", manager.size)
	}
}

func TestHistoryStats_Structure(t *testing.T) {
	now := time.Now()
	stats := HistoryStats{
		TotalEntries:  100,
		TotalBytes:    1024,
		InputEntries:  60,
		OutputEntries: 40,
		InputBytes:    600,
		OutputBytes:   424,
		MaxSize:       2048,
		CurrentSize:   1024,
		OldestEntry:   &now,
		NewestEntry:   &now,
	}

	if stats.TotalEntries != 100 {
		t.Errorf("HistoryStats.TotalEntries = %d, want 100", stats.TotalEntries)
	}

	if stats.InputEntries+stats.OutputEntries != stats.TotalEntries {
		t.Errorf("Input + Output entries (%d + %d) should equal total entries (%d)",
			stats.InputEntries, stats.OutputEntries, stats.TotalEntries)
	}

	if stats.InputBytes+stats.OutputBytes != stats.TotalBytes {
		t.Errorf("Input + Output bytes (%d + %d) should equal total bytes (%d)",
			stats.InputBytes, stats.OutputBytes, stats.TotalBytes)
	}

	if stats.CurrentSize > stats.MaxSize {
		t.Errorf("Current size (%d) should not exceed max size (%d)",
			stats.CurrentSize, stats.MaxSize)
	}

	if stats.OldestEntry == nil {
		t.Error("OldestEntry should not be nil in this test")
	}

	if stats.NewestEntry == nil {
		t.Error("NewestEntry should not be nil in this test")
	}
}

func TestRingBufferHistoryManager_Write(t *testing.T) {
	manager := NewRingBufferHistoryManager(1024)

	testData := []byte("Hello, World!")
	err := manager.Write(testData, DirectionInput)
	if err != nil {
		t.Errorf("Write() failed: %v", err)
	}

	if manager.GetSize() != len(testData) {
		t.Errorf("GetSize() = %d, want %d", manager.GetSize(), len(testData))
	}

	if manager.GetEntryCount() != 1 {
		t.Errorf("GetEntryCount() = %d, want 1", manager.GetEntryCount())
	}
}

func TestRingBufferHistoryManager_WriteNilData(t *testing.T) {
	manager := NewRingBufferHistoryManager(1024)

	err := manager.Write(nil, DirectionInput)
	if err == nil {
		t.Error("Write() with nil data should return error")
	}
}

func TestRingBufferHistoryManager_WriteInvalidDirection(t *testing.T) {
	manager := NewRingBufferHistoryManager(1024)

	testData := []byte("test")
	err := manager.Write(testData, Direction(999))
	if err == nil {
		t.Error("Write() with invalid direction should return error")
	}
}

func TestRingBufferHistoryManager_Read(t *testing.T) {
	manager := NewRingBufferHistoryManager(1024)

	testData := []byte("Hello, World!")
	manager.Write(testData, DirectionInput)

	readData, err := manager.Read(0, len(testData))
	if err != nil {
		t.Errorf("Read() failed: %v", err)
	}

	if string(readData) != string(testData) {
		t.Errorf("Read() = %s, want %s", string(readData), string(testData))
	}
}

func TestRingBufferHistoryManager_ReadNegativeOffset(t *testing.T) {
	manager := NewRingBufferHistoryManager(1024)

	_, err := manager.Read(-1, 10)
	if err == nil {
		t.Error("Read() with negative offset should return error")
	}
}

func TestRingBufferHistoryManager_ReadNegativeLength(t *testing.T) {
	manager := NewRingBufferHistoryManager(1024)

	_, err := manager.Read(0, -1)
	if err == nil {
		t.Error("Read() with negative length should return error")
	}
}

func TestRingBufferHistoryManager_ReadBeyondData(t *testing.T) {
	manager := NewRingBufferHistoryManager(1024)

	testData := []byte("Hello")
	manager.Write(testData, DirectionInput)

	// Read beyond available data
	readData, err := manager.Read(10, 5)
	if err != nil {
		t.Errorf("Read() beyond data failed: %v", err)
	}

	if len(readData) != 0 {
		t.Errorf("Read() beyond data should return empty slice, got %d bytes", len(readData))
	}
}

func TestRingBufferHistoryManager_Clear(t *testing.T) {
	manager := NewRingBufferHistoryManager(1024)

	testData := []byte("Hello, World!")
	manager.Write(testData, DirectionInput)

	err := manager.Clear()
	if err != nil {
		t.Errorf("Clear() failed: %v", err)
	}

	if manager.GetSize() != 0 {
		t.Errorf("GetSize() after Clear() = %d, want 0", manager.GetSize())
	}

	if manager.GetEntryCount() != 0 {
		t.Errorf("GetEntryCount() after Clear() = %d, want 0", manager.GetEntryCount())
	}
}

func TestRingBufferHistoryManager_SetMaxSize(t *testing.T) {
	manager := NewRingBufferHistoryManager(1024)

	// Test setting larger size
	err := manager.SetMaxSize(2048)
	if err != nil {
		t.Errorf("SetMaxSize() failed: %v", err)
	}

	if manager.GetMaxSize() != 2048 {
		t.Errorf("GetMaxSize() = %d, want 2048", manager.GetMaxSize())
	}

	// Test setting invalid size
	err = manager.SetMaxSize(0)
	if err == nil {
		t.Error("SetMaxSize() with zero size should return error")
	}

	err = manager.SetMaxSize(-1)
	if err == nil {
		t.Error("SetMaxSize() with negative size should return error")
	}
}

func TestRingBufferHistoryManager_GetEntries(t *testing.T) {
	manager := NewRingBufferHistoryManager(1024)

	// Add some test data
	testData1 := []byte("First message")
	testData2 := []byte("Second message")

	manager.Write(testData1, DirectionInput)
	manager.Write(testData2, DirectionOutput)

	entries, err := manager.GetEntries(0, 2)
	if err != nil {
		t.Errorf("GetEntries() failed: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("GetEntries() returned %d entries, want 2", len(entries))
	}

	if string(entries[0].Data) != string(testData1) {
		t.Errorf("First entry data = %s, want %s", string(entries[0].Data), string(testData1))
	}

	if entries[0].Direction != DirectionInput {
		t.Errorf("First entry direction = %v, want %v", entries[0].Direction, DirectionInput)
	}

	if string(entries[1].Data) != string(testData2) {
		t.Errorf("Second entry data = %s, want %s", string(entries[1].Data), string(testData2))
	}

	if entries[1].Direction != DirectionOutput {
		t.Errorf("Second entry direction = %v, want %v", entries[1].Direction, DirectionOutput)
	}
}

func TestRingBufferHistoryManager_GetEntriesInvalidParams(t *testing.T) {
	manager := NewRingBufferHistoryManager(1024)

	// Test negative start
	_, err := manager.GetEntries(-1, 5)
	if err == nil {
		t.Error("GetEntries() with negative start should return error")
	}

	// Test negative count
	_, err = manager.GetEntries(0, -1)
	if err == nil {
		t.Error("GetEntries() with negative count should return error")
	}
}

func TestRingBufferHistoryManager_GetStats(t *testing.T) {
	manager := NewRingBufferHistoryManager(1024)

	// Add some test data
	inputData := []byte("Input message")
	outputData := []byte("Output message")

	manager.Write(inputData, DirectionInput)
	manager.Write(outputData, DirectionOutput)

	stats := manager.GetStats()

	if stats.TotalEntries != 2 {
		t.Errorf("Stats.TotalEntries = %d, want 2", stats.TotalEntries)
	}

	if stats.InputEntries != 1 {
		t.Errorf("Stats.InputEntries = %d, want 1", stats.InputEntries)
	}

	if stats.OutputEntries != 1 {
		t.Errorf("Stats.OutputEntries = %d, want 1", stats.OutputEntries)
	}

	if stats.InputBytes != len(inputData) {
		t.Errorf("Stats.InputBytes = %d, want %d", stats.InputBytes, len(inputData))
	}

	if stats.OutputBytes != len(outputData) {
		t.Errorf("Stats.OutputBytes = %d, want %d", stats.OutputBytes, len(outputData))
	}

	if stats.MaxSize != 1024 {
		t.Errorf("Stats.MaxSize = %d, want 1024", stats.MaxSize)
	}
}

func TestRingBufferHistoryManager_BufferOverflow(t *testing.T) {
	// Create small buffer to test overflow
	manager := NewRingBufferHistoryManager(50)

	// Write data that exceeds buffer size
	for i := 0; i < 10; i++ {
		data := []byte(fmt.Sprintf("Message %d with some content", i))
		err := manager.Write(data, DirectionInput)
		if err != nil {
			t.Errorf("Write() failed on iteration %d: %v", i, err)
		}
	}

	// Buffer should not exceed max size
	if manager.GetSize() > manager.GetMaxSize() {
		t.Errorf("Buffer size %d exceeds max size %d", manager.GetSize(), manager.GetMaxSize())
	}

	// Should still have some entries
	if manager.GetEntryCount() == 0 {
		t.Error("Should have some entries after overflow")
	}
}

func TestMemoryHistoryManager_Basic(t *testing.T) {
	manager := NewMemoryHistoryManager(1024)

	testData := []byte("Hello, Memory!")
	err := manager.Write(testData, DirectionInput)
	if err != nil {
		t.Errorf("Write() failed: %v", err)
	}

	if manager.GetSize() != len(testData) {
		t.Errorf("GetSize() = %d, want %d", manager.GetSize(), len(testData))
	}

	if manager.GetEntryCount() != 1 {
		t.Errorf("GetEntryCount() = %d, want 1", manager.GetEntryCount())
	}

	entries, err := manager.GetEntries(0, 1)
	if err != nil {
		t.Errorf("GetEntries() failed: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("GetEntries() returned %d entries, want 1", len(entries))
	}

	if string(entries[0].Data) != string(testData) {
		t.Errorf("Entry data = %s, want %s", string(entries[0].Data), string(testData))
	}
}

func TestMemoryHistoryManager_Clear(t *testing.T) {
	manager := NewMemoryHistoryManager(1024)

	testData := []byte("Test data")
	manager.Write(testData, DirectionInput)

	err := manager.Clear()
	if err != nil {
		t.Errorf("Clear() failed: %v", err)
	}

	if manager.GetSize() != 0 {
		t.Errorf("GetSize() after Clear() = %d, want 0", manager.GetSize())
	}

	if manager.GetEntryCount() != 0 {
		t.Errorf("GetEntryCount() after Clear() = %d, want 0", manager.GetEntryCount())
	}
}

func TestRingBufferHistoryManager_SaveToFile(t *testing.T) {
	manager := NewRingBufferHistoryManager(1024)

	// Add some test data
	testData1 := []byte("First message")
	testData2 := []byte("Second message")

	manager.Write(testData1, DirectionInput)
	manager.Write(testData2, DirectionOutput)

	// Test different formats
	tempDir := t.TempDir()

	// Test plain text format
	plainFile := tempDir + "/plain.txt"
	err := manager.SaveToFile(plainFile, FormatPlainText)
	if err != nil {
		t.Errorf("SaveToFile(PlainText) failed: %v", err)
	}

	// Test timestamped format
	timestampFile := tempDir + "/timestamp.txt"
	err = manager.SaveToFile(timestampFile, FormatTimestamped)
	if err != nil {
		t.Errorf("SaveToFile(Timestamped) failed: %v", err)
	}

	// Test JSON format
	jsonFile := tempDir + "/data.json"
	err = manager.SaveToFile(jsonFile, FormatJSON)
	if err != nil {
		t.Errorf("SaveToFile(JSON) failed: %v", err)
	}

	// Verify files exist
	files := []string{plainFile, timestampFile, jsonFile}
	for _, file := range files {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("File %s should exist after SaveToFile", file)
		}
	}
}

func TestRingBufferHistoryManager_SaveToFileEmptyFilename(t *testing.T) {
	manager := NewRingBufferHistoryManager(1024)

	err := manager.SaveToFile("", FormatPlainText)
	if err == nil {
		t.Error("SaveToFile() with empty filename should return error")
	}
}

func TestMemoryHistoryManager_SaveToFile(t *testing.T) {
	manager := NewMemoryHistoryManager(1024)

	testData := []byte("Memory test data")
	manager.Write(testData, DirectionInput)

	tempDir := t.TempDir()
	filename := tempDir + "/memory_test.json"

	err := manager.SaveToFile(filename, FormatJSON)
	if err != nil {
		t.Errorf("SaveToFile() failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Error("File should exist after SaveToFile")
	}
}

func TestSaveEntriesToFile_UnsupportedFormat(t *testing.T) {
	entries := []HistoryEntry{
		NewHistoryEntry([]byte("test"), DirectionInput),
	}

	tempDir := t.TempDir()
	filename := tempDir + "/test.txt"

	err := saveEntriesToFile(entries, filename, FileFormat(999))
	if err == nil {
		t.Error("saveEntriesToFile() with unsupported format should return error")
	}
}

func TestMemoryHistoryManager_Read(t *testing.T) {
	manager := NewMemoryHistoryManager(1024)

	// Add multiple messages
	msg1 := []byte("Hello")
	msg2 := []byte(" ")
	msg3 := []byte("World")

	manager.Write(msg1, DirectionInput)
	manager.Write(msg2, DirectionOutput)
	manager.Write(msg3, DirectionInput)

	// Read all data
	allData, err := manager.Read(0, 100)
	if err != nil {
		t.Errorf("Read() failed: %v", err)
	}

	expected := "Hello World"
	if string(allData) != expected {
		t.Errorf("Read() = %s, want %s", string(allData), expected)
	}

	// Read partial data
	partialData, err := manager.Read(0, 5)
	if err != nil {
		t.Errorf("Read() partial failed: %v", err)
	}

	if string(partialData) != "Hello" {
		t.Errorf("Read() partial = %s, want Hello", string(partialData))
	}
}

func TestMemoryHistoryManager_SetMaxSize(t *testing.T) {
	manager := NewMemoryHistoryManager(100)

	// Fill with data
	for i := 0; i < 10; i++ {
		data := []byte(fmt.Sprintf("Message %d", i))
		manager.Write(data, DirectionInput)
	}

	originalCount := manager.GetEntryCount()

	// Reduce max size
	err := manager.SetMaxSize(50)
	if err != nil {
		t.Errorf("SetMaxSize() failed: %v", err)
	}

	if manager.GetMaxSize() != 50 {
		t.Errorf("GetMaxSize() = %d, want 50", manager.GetMaxSize())
	}

	// Should have fewer entries now
	newCount := manager.GetEntryCount()
	if newCount >= originalCount {
		t.Errorf("Entry count should be reduced after SetMaxSize, was %d, now %d", originalCount, newCount)
	}

	// Test invalid size
	err = manager.SetMaxSize(0)
	if err == nil {
		t.Error("SetMaxSize() with zero size should return error")
	}
}

func TestMemoryHistoryManager_GetEntriesInvalidParams(t *testing.T) {
	manager := NewMemoryHistoryManager(1024)

	// Test negative start
	_, err := manager.GetEntries(-1, 5)
	if err == nil {
		t.Error("GetEntries() with negative start should return error")
	}

	// Test negative count
	_, err = manager.GetEntries(0, -1)
	if err == nil {
		t.Error("GetEntries() with negative count should return error")
	}

	// Test start beyond entries
	entries, err := manager.GetEntries(100, 5)
	if err != nil {
		t.Errorf("GetEntries() beyond entries failed: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("GetEntries() beyond entries should return empty slice, got %d entries", len(entries))
	}
}

func TestNewMemoryHistoryManager_DefaultSize(t *testing.T) {
	// Test with zero size (should use default)
	manager := NewMemoryHistoryManager(0)

	if manager.GetMaxSize() != 10*1024*1024 {
		t.Errorf("NewMemoryHistoryManager(0) should use default size 10MB, got %d", manager.GetMaxSize())
	}

	// Test with negative size (should use default)
	manager = NewMemoryHistoryManager(-100)

	if manager.GetMaxSize() != 10*1024*1024 {
		t.Errorf("NewMemoryHistoryManager(-100) should use default size 10MB, got %d", manager.GetMaxSize())
	}
}

func TestRingBufferHistoryManager_DefaultSize(t *testing.T) {
	// Test with zero size (should use default)
	manager := NewRingBufferHistoryManager(0)

	if manager.GetMaxSize() != 10*1024*1024 {
		t.Errorf("NewRingBufferHistoryManager(0) should use default size 10MB, got %d", manager.GetMaxSize())
	}

	// Test with negative size (should use default)
	manager = NewRingBufferHistoryManager(-100)

	if manager.GetMaxSize() != 10*1024*1024 {
		t.Errorf("NewRingBufferHistoryManager(-100) should use default size 10MB, got %d", manager.GetMaxSize())
	}
}

func TestPersistentHistoryManager_Basic(t *testing.T) {
	baseManager := NewMemoryHistoryManager(1024)
	tempDir := t.TempDir()

	persistentManager := NewPersistentHistoryManager(baseManager, tempDir)

	if persistentManager == nil {
		t.Error("NewPersistentHistoryManager() returned nil")
	}

	if persistentManager.backupDir != tempDir {
		t.Errorf("backupDir = %s, want %s", persistentManager.backupDir, tempDir)
	}

	if !persistentManager.autoBackup {
		t.Error("autoBackup should be enabled by default")
	}
}

func TestPersistentHistoryManager_SetAutoBackup(t *testing.T) {
	baseManager := NewMemoryHistoryManager(1024)
	tempDir := t.TempDir()

	persistentManager := NewPersistentHistoryManager(baseManager, tempDir)

	// Disable auto backup
	persistentManager.SetAutoBackup(false, time.Minute, 10)

	if persistentManager.autoBackup {
		t.Error("autoBackup should be disabled")
	}

	if persistentManager.backupInterval != time.Minute {
		t.Errorf("backupInterval = %v, want %v", persistentManager.backupInterval, time.Minute)
	}

	if persistentManager.maxBackupFiles != 10 {
		t.Errorf("maxBackupFiles = %d, want 10", persistentManager.maxBackupFiles)
	}
}

func TestPersistentHistoryManager_Write(t *testing.T) {
	baseManager := NewMemoryHistoryManager(1024)
	tempDir := t.TempDir()

	persistentManager := NewPersistentHistoryManager(baseManager, tempDir)

	testData := []byte("Test data for persistent manager")
	err := persistentManager.Write(testData, DirectionInput)
	if err != nil {
		t.Errorf("Write() failed: %v", err)
	}

	if persistentManager.GetSize() != len(testData) {
		t.Errorf("GetSize() = %d, want %d", persistentManager.GetSize(), len(testData))
	}
}

func TestPersistentHistoryManager_SaveToTempFile(t *testing.T) {
	baseManager := NewMemoryHistoryManager(1024)
	tempDir := t.TempDir()

	persistentManager := NewPersistentHistoryManager(baseManager, tempDir)

	// Add some test data
	testData := []byte("Temp file test data")
	persistentManager.Write(testData, DirectionInput)

	// Save to temp file
	tempPath, err := persistentManager.SaveToTempFile(FormatJSON)
	if err != nil {
		t.Errorf("SaveToTempFile() failed: %v", err)
	}

	// Verify temp file exists
	if _, err := os.Stat(tempPath); os.IsNotExist(err) {
		t.Error("Temp file should exist")
	}

	// Clean up
	os.Remove(tempPath)
}

func TestPersistentHistoryManager_LoadFromFile(t *testing.T) {
	baseManager := NewMemoryHistoryManager(1024)
	tempDir := t.TempDir()

	persistentManager := NewPersistentHistoryManager(baseManager, tempDir)

	// Create test data file
	testEntries := []HistoryEntry{
		NewHistoryEntry([]byte("First message"), DirectionInput),
		NewHistoryEntry([]byte("Second message"), DirectionOutput),
	}

	historyData := struct {
		Entries []HistoryEntry `json:"entries"`
	}{
		Entries: testEntries,
	}

	// Save test data to file
	testFile := tempDir + "/test_history.json"
	file, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	encoder := json.NewEncoder(file)
	encoder.Encode(historyData)
	file.Close()

	// Load from file
	err = persistentManager.LoadFromFile(testFile)
	if err != nil {
		t.Errorf("LoadFromFile() failed: %v", err)
	}

	// Verify data was loaded
	if persistentManager.GetEntryCount() != 2 {
		t.Errorf("GetEntryCount() = %d, want 2", persistentManager.GetEntryCount())
	}

	entries, err := persistentManager.GetEntries(0, 2)
	if err != nil {
		t.Errorf("GetEntries() failed: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("GetEntries() returned %d entries, want 2", len(entries))
	}
}

func TestPersistentHistoryManager_GetMemoryUsage(t *testing.T) {
	baseManager := NewMemoryHistoryManager(1024)
	tempDir := t.TempDir()

	persistentManager := NewPersistentHistoryManager(baseManager, tempDir)

	// Add some test data
	testData := []byte("Memory usage test data")
	persistentManager.Write(testData, DirectionInput)

	usage := persistentManager.GetMemoryUsage()

	if usage.CurrentSize != len(testData) {
		t.Errorf("MemoryUsage.CurrentSize = %d, want %d", usage.CurrentSize, len(testData))
	}

	if usage.MaxSize != 1024 {
		t.Errorf("MemoryUsage.MaxSize = %d, want 1024", usage.MaxSize)
	}

	if usage.EntryCount != 1 {
		t.Errorf("MemoryUsage.EntryCount = %d, want 1", usage.EntryCount)
	}

	if !usage.AutoBackup {
		t.Error("MemoryUsage.AutoBackup should be true")
	}

	expectedPercent := float64(len(testData)) / 1024 * 100
	if usage.UsagePercent != expectedPercent {
		t.Errorf("MemoryUsage.UsagePercent = %f, want %f", usage.UsagePercent, expectedPercent)
	}
}

func TestPersistentHistoryManager_CompactHistory(t *testing.T) {
	baseManager := NewMemoryHistoryManager(1000)
	tempDir := t.TempDir()

	persistentManager := NewPersistentHistoryManager(baseManager, tempDir)

	// Fill with data to exceed 50% of buffer
	for i := 0; i < 20; i++ {
		data := []byte(fmt.Sprintf("Message %d with lots of content to fill the buffer beyond 50 percent", i))
		persistentManager.Write(data, DirectionInput)
	}

	originalSize := persistentManager.GetSize()
	maxSize := persistentManager.GetMaxSize()

	// Only test compaction if current size is > 50% of max size
	if originalSize > maxSize/2 {
		// Compact to 30% of max size (smaller than current)
		err := persistentManager.CompactHistory(30.0)
		if err != nil {
			t.Errorf("CompactHistory() failed: %v", err)
		}

		newSize := persistentManager.GetSize()

		// Size should be reduced
		if newSize >= originalSize {
			t.Errorf("CompactHistory() should reduce size, was %d, now %d", originalSize, newSize)
		}
	} else {
		// If buffer is not full enough, test that compaction doesn't fail
		err := persistentManager.CompactHistory(50.0)
		if err != nil {
			t.Errorf("CompactHistory() should not fail even when no compaction needed: %v", err)
		}

		t.Logf("Buffer not full enough for compaction test (size: %d, max: %d)", originalSize, maxSize)
	}
}

func TestPersistentHistoryManager_CompactHistoryInvalidPercent(t *testing.T) {
	baseManager := NewMemoryHistoryManager(1024)
	tempDir := t.TempDir()

	persistentManager := NewPersistentHistoryManager(baseManager, tempDir)

	// Test invalid percentages
	err := persistentManager.CompactHistory(0)
	if err == nil {
		t.Error("CompactHistory(0) should return error")
	}

	err = persistentManager.CompactHistory(100)
	if err == nil {
		t.Error("CompactHistory(100) should return error")
	}

	err = persistentManager.CompactHistory(-10)
	if err == nil {
		t.Error("CompactHistory(-10) should return error")
	}
}

func TestPersistentHistoryManager_CreateTempFile(t *testing.T) {
	baseManager := NewMemoryHistoryManager(1024)
	tempDir := t.TempDir()

	persistentManager := NewPersistentHistoryManager(baseManager, tempDir)

	tempFile, err := persistentManager.CreateTempFile()
	if err != nil {
		t.Errorf("CreateTempFile() failed: %v", err)
	}
	defer tempFile.Close()
	defer os.Remove(tempFile.Name())

	// Verify file was created
	if _, err := os.Stat(tempFile.Name()); os.IsNotExist(err) {
		t.Error("Temp file should exist")
	}

	// Verify file name has correct prefix
	filename := filepath.Base(tempFile.Name())
	if !strings.HasPrefix(filename, "history_temp_") {
		t.Errorf("Temp file name should have prefix 'history_temp_', got: %s", filename)
	}
}
