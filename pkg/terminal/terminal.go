// Package terminal provides terminal emulation functionality
package terminal

import (
	"fmt"
	"sterm/pkg/history"
	"sterm/pkg/serial"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
)

// Terminal interface defines the contract for terminal operations
type Terminal interface {
	Start() error
	Stop() error
	ProcessInput(input []byte) error
	ProcessOutput(output []byte) error
	Resize(width, height int) error
	EnableMouse(enable bool) error
	GetState() TerminalState
	SetState(state TerminalState) error
	IsRunning() bool
}

// TerminalState represents the current state of the terminal
type TerminalState struct {
	CursorX      int            `json:"cursor_x"`
	CursorY      int            `json:"cursor_y"`
	Width        int            `json:"width"`
	Height       int            `json:"height"`
	Attributes   TextAttributes `json:"attributes"`
	MouseMode    MouseMode      `json:"mouse_mode"`
	ScrollTop    int            `json:"scroll_top"`
	ScrollBottom int            `json:"scroll_bottom"`
	IsRunning    bool           `json:"is_running"`
	LineWrap     bool           `json:"line_wrap"`
}

// Validate checks if the terminal state is valid
func (t TerminalState) Validate() error {
	if t.Width <= 0 {
		return fmt.Errorf("width must be positive, got: %d", t.Width)
	}

	if t.Height <= 0 {
		return fmt.Errorf("height must be positive, got: %d", t.Height)
	}

	if t.CursorX < 0 || t.CursorX >= t.Width {
		return fmt.Errorf("cursor X out of bounds: %d (width: %d)", t.CursorX, t.Width)
	}

	if t.CursorY < 0 || t.CursorY >= t.Height {
		return fmt.Errorf("cursor Y out of bounds: %d (height: %d)", t.CursorY, t.Height)
	}

	if t.ScrollTop < 0 || t.ScrollTop >= t.Height {
		return fmt.Errorf("scroll top out of bounds: %d (height: %d)", t.ScrollTop, t.Height)
	}

	if t.ScrollBottom < 0 || t.ScrollBottom >= t.Height {
		return fmt.Errorf("scroll bottom out of bounds: %d (height: %d)", t.ScrollBottom, t.Height)
	}

	if t.ScrollTop > t.ScrollBottom {
		return fmt.Errorf("scroll top (%d) cannot be greater than scroll bottom (%d)", t.ScrollTop, t.ScrollBottom)
	}

	return nil
}

// DefaultTerminalState returns a default terminal state
func DefaultTerminalState(width, height int) TerminalState {
	return TerminalState{
		CursorX:      0,
		CursorY:      0,
		Width:        width,
		Height:       height,
		Attributes:   DefaultTextAttributes(),
		MouseMode:    MouseModeOff,
		ScrollTop:    0,
		ScrollBottom: height - 1,
		IsRunning:    false,
		LineWrap:     true, // Default to line wrap enabled
	}
}

// TextAttributes defines text formatting attributes
type TextAttributes struct {
	Foreground Color `json:"foreground"`
	Background Color `json:"background"`
	Bold       bool  `json:"bold"`
	Italic     bool  `json:"italic"`
	Underline  bool  `json:"underline"`
	Reverse    bool  `json:"reverse"`
	Blink      bool  `json:"blink"`
}

// DefaultTextAttributes returns default text attributes
func DefaultTextAttributes() TextAttributes {
	return TextAttributes{
		Foreground: ColorDefault, // Use terminal default foreground
		Background: ColorDefault, // Use terminal default background
		Bold:       false,
		Italic:     false,
		Underline:  false,
		Reverse:    false,
		Blink:      false,
	}
}

// Color represents terminal colors
type Color int

const (
	ColorDefault       Color = -1 // Terminal default color
	ColorBlack         Color = 0
	ColorRed           Color = 1
	ColorGreen         Color = 2
	ColorYellow        Color = 3
	ColorBlue          Color = 4
	ColorMagenta       Color = 5
	ColorCyan          Color = 6
	ColorWhite         Color = 7
	ColorBrightBlack   Color = 8
	ColorBrightRed     Color = 9
	ColorBrightGreen   Color = 10
	ColorBrightYellow  Color = 11
	ColorBrightBlue    Color = 12
	ColorBrightMagenta Color = 13
	ColorBrightCyan    Color = 14
	ColorBrightWhite   Color = 15
)

// String returns the string representation of Color
func (c Color) String() string {
	if c == ColorDefault {
		return "default"
	}

	colors := []string{
		"black", "red", "green", "yellow", "blue", "magenta", "cyan", "white",
		"bright_black", "bright_red", "bright_green", "bright_yellow",
		"bright_blue", "bright_magenta", "bright_cyan", "bright_white",
	}

	if int(c) >= 0 && int(c) < len(colors) {
		return colors[c]
	}
	return "unknown"
}

// MouseMode represents different mouse modes
type MouseMode int

const (
	MouseModeOff MouseMode = iota
	MouseModeX10
	MouseModeVT200
	MouseModeVT200Highlight
	MouseModeBtnEvent
	MouseModeAnyEvent
)

// String returns the string representation of MouseMode
func (m MouseMode) String() string {
	modes := []string{
		"off", "x10", "vt200", "vt200_highlight", "btn_event", "any_event",
	}

	if int(m) < len(modes) {
		return modes[m]
	}
	return "unknown"
}

// Logger interface for debug logging
type Logger interface {
	Debugf(format string, args ...interface{})
}

// TerminalEmulator implements the Terminal interface
type TerminalEmulator struct {
	screen         *Screen
	altScreen      *Screen // Alternative screen buffer for full-screen apps
	parser         *VTParser
	serialPort     serial.SerialPort
	historyManager history.HistoryManager
	state          TerminalState
	savedState     *TerminalState // Saved cursor state for DECSC/DECRC
	isRunning      bool
	useAltScreen   bool         // Whether using alternative screen
	tabStops       map[int]bool // Custom tab stops
	utf8Decoder    *UTF8Decoder // UTF-8 decoder for multi-byte characters
	logger         Logger       // Logger for debug output
	mu             sync.RWMutex // Protect concurrent access

	// Scrollback buffer for history
	scrollbackBuffer [][]Cell // History lines
	scrollbackSize   int      // Maximum scrollback lines
	scrollOffset     int      // Current scroll position (0 = bottom/normal)
	scrollPosition   int      // Absolute line position in scroll mode (fixed position)
	isScrolling      bool     // Whether in scroll mode

	// Mouse mode change callback
	onMouseModeChange func(mode MouseMode)
}

// NewTerminalEmulator creates a new terminal emulator
func NewTerminalEmulator(serialPort serial.SerialPort, historyManager history.HistoryManager, width, height int) *TerminalEmulator {
	te := &TerminalEmulator{
		screen:           NewScreen(width, height),
		altScreen:        NewScreen(width, height),
		parser:           NewVTParser(),
		serialPort:       serialPort,
		historyManager:   historyManager,
		state:            DefaultTerminalState(width, height),
		savedState:       nil,
		isRunning:        false,
		useAltScreen:     false,
		tabStops:         make(map[int]bool),
		utf8Decoder:      NewUTF8Decoder(),
		logger:           nil,                       // Will be set with SetLogger if needed
		scrollbackBuffer: make([][]Cell, 0, 100000), // Initial capacity of 100000 lines
		scrollbackSize:   100000,                    // Maximum 100000 lines of history
		scrollOffset:     0,                         // Start at bottom (no scroll)
		scrollPosition:   0,                         // Absolute position in buffer
		isScrolling:      false,
	}
	// Initialize default tab stops every 8 columns
	for i := 8; i < width; i += 8 {
		te.tabStops[i] = true
	}
	return te
}

// SetLogger sets the logger for debug output
func (te *TerminalEmulator) SetLogger(logger Logger) {
	te.logger = logger
	if te.utf8Decoder != nil {
		te.utf8Decoder.logger = logger
	}
}

// SetMouseModeChangeCallback sets a callback for mouse mode changes
func (te *TerminalEmulator) SetMouseModeChangeCallback(callback func(mode MouseMode)) {
	te.onMouseModeChange = callback
}

// Screen represents the terminal screen buffer
type Screen struct {
	Width  int
	Height int
	Buffer [][]Cell
	Dirty  bool

	// Dirty region tracking
	DirtyLines map[int]bool // Track which lines are dirty
	DirtyMinX  int          // Minimum dirty X coordinate
	DirtyMaxX  int          // Maximum dirty X coordinate
	DirtyMinY  int          // Minimum dirty Y coordinate
	DirtyMaxY  int          // Maximum dirty Y coordinate

	// Special flags
	JustCleared bool // Flag to indicate screen was just cleared

	// Mutex for thread safety
	mutex sync.RWMutex
}

// Cell represents a single character cell in the terminal
type Cell struct {
	Char       rune           `json:"char"`
	Attributes TextAttributes `json:"attributes"`
	Dirty      bool           `json:"-"` // Track if this cell is dirty
}

// NewScreen creates a new screen buffer
func NewScreen(width, height int) *Screen {
	buffer := make([][]Cell, height)
	for i := range buffer {
		buffer[i] = make([]Cell, width)
		for j := range buffer[i] {
			buffer[i][j] = Cell{
				Char:       ' ',
				Attributes: DefaultTextAttributes(),
				Dirty:      true,
			}
		}
	}

	return &Screen{
		Width:      width,
		Height:     height,
		Buffer:     buffer,
		Dirty:      true,
		DirtyLines: make(map[int]bool),
		DirtyMinX:  0,
		DirtyMaxX:  width - 1,
		DirtyMinY:  0,
		DirtyMaxY:  height - 1,
	}
}

// MarkDirty marks a region as dirty
func (s *Screen) MarkDirty(x, y int) {
	// Bounds check first - prevent out of bounds access
	if y < 0 || y >= s.Height || x < 0 || x >= s.Width {
		// Ignore out of bounds coordinates
		return
	}

	// Extra safety: also check buffer bounds
	if y >= len(s.Buffer) {
		return
	}
	if len(s.Buffer) > 0 && len(s.Buffer[y]) > 0 && x >= len(s.Buffer[y]) {
		return
	}

	// Lock for concurrent access
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.Dirty = true

	// Initialize map if nil
	if s.DirtyLines == nil {
		s.DirtyLines = make(map[int]bool)
	}

	s.DirtyLines[y] = true

	if len(s.DirtyLines) == 1 {
		// First dirty cell, initialize bounds
		s.DirtyMinX = x
		s.DirtyMaxX = x
		s.DirtyMinY = y
		s.DirtyMaxY = y
	} else {
		// Update bounds
		if x < s.DirtyMinX {
			s.DirtyMinX = x
		}
		if x > s.DirtyMaxX {
			s.DirtyMaxX = x
		}
		if y < s.DirtyMinY {
			s.DirtyMinY = y
		}
		if y > s.DirtyMaxY {
			s.DirtyMaxY = y
		}
	}

	if y >= 0 && y < len(s.Buffer) && x >= 0 && x < len(s.Buffer[y]) {
		s.Buffer[y][x].Dirty = true
	}
}

// MarkLineDirty marks an entire line as dirty
func (s *Screen) MarkLineDirty(y int) {
	// Bounds check
	if y < 0 || y >= s.Height || y >= len(s.Buffer) {
		return
	}

	// Lock for concurrent access
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.Dirty = true

	// Initialize map if nil
	if s.DirtyLines == nil {
		s.DirtyLines = make(map[int]bool)
	}

	s.DirtyLines[y] = true

	// Mark all cells in this line as dirty
	for x := 0; x < s.Width && x < len(s.Buffer[y]); x++ {
		s.Buffer[y][x].Dirty = true
	}

	// Update dirty bounds to include entire line
	if len(s.DirtyLines) == 1 {
		// First dirty line, initialize bounds
		s.DirtyMinX = 0
		s.DirtyMaxX = s.Width - 1
		s.DirtyMinY = y
		s.DirtyMaxY = y
	} else {
		// Expand bounds to include full line width
		if 0 < s.DirtyMinX {
			s.DirtyMinX = 0
		}
		if s.Width-1 > s.DirtyMaxX {
			s.DirtyMaxX = s.Width - 1
		}
		if y < s.DirtyMinY {
			s.DirtyMinY = y
		}
		if y > s.DirtyMaxY {
			s.DirtyMaxY = y
		}
	}
}

// ClearDirty clears all dirty flags
func (s *Screen) ClearDirty() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.Dirty = false
	s.DirtyLines = make(map[int]bool)
	// Initialize to invalid range (min > max) so first dirty cell will reset them
	s.DirtyMinX = s.Width
	s.DirtyMaxX = -1
	s.DirtyMinY = s.Height
	s.DirtyMaxY = -1

	// Note: We intentionally do NOT clear JustCleared flag here
	// It needs to be handled by the display update

	for y := range s.Buffer {
		for x := range s.Buffer[y] {
			s.Buffer[y][x].Dirty = false
		}
	}
}

// IsLineDirty checks if a line is marked as dirty (thread-safe)
func (s *Screen) IsLineDirty(y int) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if s.DirtyLines == nil {
		return false
	}
	return s.DirtyLines[y]
}

// IsJustCleared checks if the screen was just cleared (thread-safe)
func (s *Screen) IsJustCleared() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.JustCleared
}

// ClearJustClearedFlag clears the JustCleared flag (thread-safe)
func (s *Screen) ClearJustClearedFlag() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.JustCleared = false
}

// GetDirtyBounds returns the dirty region bounds (thread-safe)
func (s *Screen) GetDirtyBounds() (minX, maxX, minY, maxY int, hasDirty bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if !s.Dirty || len(s.DirtyLines) == 0 {
		return 0, 0, 0, 0, false
	}
	return s.DirtyMinX, s.DirtyMaxX, s.DirtyMinY, s.DirtyMaxY, true
}

// UTF8Decoder handles UTF-8 character decoding
type UTF8Decoder struct {
	bytes    []byte
	expected int
	logger   Logger
}

// NewUTF8Decoder creates a new UTF-8 decoder
func NewUTF8Decoder() *UTF8Decoder {
	return &UTF8Decoder{
		bytes:    make([]byte, 0, 4),
		expected: 0,
	}
}

// Decode processes a byte and returns a rune if complete, or 0 if incomplete
func (d *UTF8Decoder) Decode(b byte) (rune, bool) {
	// If we're expecting continuation bytes
	if d.expected > 0 {
		// Check if this is a valid continuation byte
		if b >= 0x80 && b < 0xC0 {
			// Valid continuation byte
			d.bytes = append(d.bytes, b)
			d.expected--

			if d.expected == 0 {
				// Complete character
				r, size := decodeUTF8(d.bytes)
				if size > 0 {
					// Successfully decoded
					d.Reset()
					return r, true
				} else {
					// Failed to decode
					d.Reset()
					return '�', true
				}
			}
			// Still need more bytes
			return 0, false
		} else {
			// Invalid continuation byte - either new start or ASCII
			// This byte is NOT a valid continuation
			// Check if it's ASCII - if so, we need to output replacement for incomplete sequence
			if b < 0x80 {
				// ASCII byte interrupting UTF-8 sequence
				d.Reset()
				// Process ASCII byte normally
				return rune(b), true
			}
			// Reset and try to process this byte as new character
			d.Reset()
			// Fall through to process as new character
		}
	}

	// Starting a new character
	if b < 0x80 { // ASCII
		return rune(b), true
	} else if b < 0xC0 { // Orphaned continuation byte
		// This shouldn't happen in valid UTF-8
		// IMPORTANT: Never treat a continuation byte as a character!
		// It should always return replacement character
		return '�', true
	} else if b < 0xE0 { // 2-byte sequence
		d.bytes = append(d.bytes[:0], b)
		d.expected = 1
		return 0, false
	} else if b < 0xF0 { // 3-byte sequence
		d.bytes = append(d.bytes[:0], b)
		d.expected = 2
		return 0, false
	} else if b < 0xF8 { // 4-byte sequence
		d.bytes = append(d.bytes[:0], b)
		d.expected = 3
		return 0, false
	} else { // Invalid UTF-8
		return '�', true
	}
}

// Reset resets the decoder state
func (d *UTF8Decoder) Reset() {
	d.bytes = d.bytes[:0]
	d.expected = 0
}

// decodeUTF8 decodes a complete UTF-8 sequence
func decodeUTF8(bytes []byte) (rune, int) {
	if len(bytes) == 0 {
		return 0, 0
	}

	switch len(bytes) {
	case 1:
		return rune(bytes[0]), 1
	case 2:
		r := rune((bytes[0]&0x1F)<<6 | (bytes[1] & 0x3F))
		return r, 2
	case 3:
		// Debug the calculation
		b0 := uint32(bytes[0]) & 0x0F
		b1 := uint32(bytes[1]) & 0x3F
		b2 := uint32(bytes[2]) & 0x3F
		result := (b0 << 12) | (b1 << 6) | b2
		r := rune(result)
		// For E4 B8 AD: b0=4, b1=38, b2=2D -> 4000 + E00 + 2D = 4E2D (中)
		return r, 3
	case 4:
		r := rune(int32(bytes[0]&0x07)<<18 | int32(bytes[1]&0x3F)<<12 | int32(bytes[2]&0x3F)<<6 | int32(bytes[3]&0x3F))
		return r, 4
	default:
		return 0, 0
	}
}

// VTParser handles VT100/ANSI escape sequence parsing
type VTParser struct {
	State        ParserState
	Buffer       []byte
	Params       []int
	Intermediate []byte
}

// ParserState represents the current state of the VT parser
type ParserState int

const (
	StateGround ParserState = iota
	StateEscape
	StateCSI
	StateOSC
	StateDCS
)

// NewVTParser creates a new VT parser
func NewVTParser() *VTParser {
	return &VTParser{
		State:        StateGround,
		Buffer:       make([]byte, 0, 256),
		Params:       make([]int, 0, 16),
		Intermediate: make([]byte, 0, 16),
	}
}

// Reset resets the parser to initial state
func (vt *VTParser) Reset() {
	vt.State = StateGround
	vt.Buffer = vt.Buffer[:0]
	vt.Params = vt.Params[:0]
	vt.Intermediate = vt.Intermediate[:0]
}

// ParseByte processes a single byte through the VT parser state machine
func (vt *VTParser) ParseByte(b byte, screen *Screen, state *TerminalState, utf8Decoder *UTF8Decoder) []Action {
	var actions []Action

	switch vt.State {
	case StateGround:
		actions = vt.handleGround(b, screen, state, utf8Decoder)
	case StateEscape:
		actions = vt.handleEscape(b, screen, state)
	case StateCSI:
		actions = vt.handleCSI(b, screen, state)
	case StateOSC:
		actions = vt.handleOSC(b, screen, state)
	case StateDCS:
		actions = vt.handleDCS(b, screen, state)
	}

	return actions
}

// Action represents an action to be performed on the terminal
type Action struct {
	Type ActionType
	Data interface{}
}

// ActionType represents different types of terminal actions
type ActionType int

const (
	ActionPrint ActionType = iota
	ActionMoveCursor
	ActionClearScreen
	ActionClearLine
	ActionSetAttribute
	ActionScroll
	ActionSetMode
	ActionBell
	ActionTab
	ActionNewline
	ActionCarriageReturn
	ActionBackspace
	ActionDeleteChar
	ActionInsertChar
	ActionSetScrollRegion
	ActionSaveCursor
	ActionRestoreCursor
	ActionSwitchAltScreen
	ActionSendResponse
	ActionSetTabStop
	ActionClearTabStop
	ActionReset
)

// handleGround processes characters in ground state
func (vt *VTParser) handleGround(b byte, screen *Screen, state *TerminalState, utf8Decoder *UTF8Decoder) []Action {
	switch b {
	case 0x1B: // ESC
		vt.State = StateEscape
		// Don't reset UTF-8 decoder here - let it continue buffering
		// utf8Decoder.Reset()
		return nil
	case 0x07: // BEL
		return []Action{{Type: ActionBell}}
	case 0x08: // BS
		return []Action{{Type: ActionBackspace}}
	case 0x09: // HT
		return []Action{{Type: ActionTab}}
	case 0x0A: // LF
		return []Action{{Type: ActionNewline}}
	case 0x0D: // CR
		return []Action{{Type: ActionCarriageReturn}}
	default:
		if b >= 0x20 && b <= 0x7E { // Printable ASCII
			return []Action{{Type: ActionPrint, Data: rune(b)}}
		}
		// UTF-8 and other bytes are handled in ProcessOutput
		// Ignore control characters below 0x20
		return nil
	}
}

// handleEscape processes escape sequences
func (vt *VTParser) handleEscape(b byte, screen *Screen, state *TerminalState) []Action {
	switch b {
	case '[': // CSI
		vt.State = StateCSI
		vt.Buffer = vt.Buffer[:0]
		vt.Params = vt.Params[:0]
		vt.Intermediate = vt.Intermediate[:0]
		return nil
	case ']': // OSC
		vt.State = StateOSC
		vt.Buffer = vt.Buffer[:0]
		return nil
	case 'P': // DCS
		vt.State = StateDCS
		vt.Buffer = vt.Buffer[:0]
		return nil
	case 'D': // IND - Index
		vt.Reset()
		return []Action{{Type: ActionScroll, Data: "down"}}
	case 'M': // RI - Reverse Index
		vt.Reset()
		return []Action{{Type: ActionScroll, Data: "up"}}
	case 'E': // NEL - Next Line
		vt.Reset()
		return []Action{{Type: ActionNewline}, {Type: ActionCarriageReturn}}
	case 'H': // HTS - Horizontal Tab Set
		vt.Reset()
		return []Action{{Type: ActionSetTabStop}}
	case '7': // DECSC - Save Cursor
		vt.Reset()
		return []Action{{Type: ActionSaveCursor}}
	case '8': // DECRC - Restore Cursor
		vt.Reset()
		return []Action{{Type: ActionRestoreCursor}}
	case '=': // DECKPAM - Keypad Application Mode
		vt.Reset()
		return []Action{{Type: ActionSetMode, Data: "keypad_app"}}
	case '>': // DECKPNM - Keypad Numeric Mode
		vt.Reset()
		return []Action{{Type: ActionSetMode, Data: "keypad_num"}}
	case 'c': // RIS - Reset to Initial State
		vt.Reset()
		return []Action{{Type: ActionReset}}
	default:
		vt.Reset()
		return nil
	}
}

// handleCSI processes Control Sequence Introducer sequences
func (vt *VTParser) handleCSI(b byte, screen *Screen, state *TerminalState) []Action {
	// Special handling for '?' which marks private mode parameters
	if b == '?' && len(vt.Buffer) == 0 && len(vt.Params) == 0 {
		// '?' at the beginning is an intermediate byte for private modes
		vt.Intermediate = append(vt.Intermediate, b)
		return nil
	}

	if b >= 0x30 && b <= 0x3F { // Parameter bytes (0-9, :, ;, <, =, >, ?)
		vt.Buffer = append(vt.Buffer, b)
		return nil
	}

	if b >= 0x20 && b <= 0x2F { // Intermediate bytes
		vt.Intermediate = append(vt.Intermediate, b)
		return nil
	}

	if b >= 0x40 && b <= 0x7E { // Final byte
		actions := vt.executeCSI(b, screen, state)
		vt.Reset()
		return actions
	}

	// Invalid sequence, reset
	vt.Reset()
	return nil
}

// executeCSI executes a complete CSI sequence
func (vt *VTParser) executeCSI(final byte, screen *Screen, state *TerminalState) []Action {
	// Parse parameters
	vt.parseParams()

	switch final {
	case 'A': // CUU - Cursor Up
		count := vt.getParam(0, 1)
		return []Action{{Type: ActionMoveCursor, Data: CursorMove{Direction: "up", Count: count}}}
	case 'B': // CUD - Cursor Down
		count := vt.getParam(0, 1)
		return []Action{{Type: ActionMoveCursor, Data: CursorMove{Direction: "down", Count: count}}}
	case 'C': // CUF - Cursor Forward
		count := vt.getParam(0, 1)
		return []Action{{Type: ActionMoveCursor, Data: CursorMove{Direction: "right", Count: count}}}
	case 'D': // CUB - Cursor Backward
		count := vt.getParam(0, 1)
		return []Action{{Type: ActionMoveCursor, Data: CursorMove{Direction: "left", Count: count}}}
	case 'E': // CNL - Cursor Next Line
		count := vt.getParam(0, 1)
		actions := []Action{}
		for i := 0; i < count; i++ {
			actions = append(actions, Action{Type: ActionNewline})
		}
		actions = append(actions, Action{Type: ActionCarriageReturn})
		return actions
	case 'F': // CPL - Cursor Previous Line
		count := vt.getParam(0, 1)
		return []Action{
			{Type: ActionMoveCursor, Data: CursorMove{Direction: "up", Count: count}},
			{Type: ActionCarriageReturn},
		}
	case 'G': // CHA - Cursor Horizontal Absolute
		col := vt.getParam(0, 1) - 1
		return []Action{{Type: ActionMoveCursor, Data: CursorMove{Direction: "horizontal", Col: col}}}
	case 'H', 'f': // CUP - Cursor Position
		row := vt.getParam(0, 1) - 1
		col := vt.getParam(1, 1) - 1
		return []Action{{Type: ActionMoveCursor, Data: CursorMove{Direction: "absolute", Row: row, Col: col}}}
	case 'J': // ED - Erase in Display
		mode := vt.getParam(0, 0)
		return []Action{{Type: ActionClearScreen, Data: mode}}
	case 'K': // EL - Erase in Line
		mode := vt.getParam(0, 0)
		return []Action{{Type: ActionClearLine, Data: mode}}
	case 'm': // SGR - Select Graphic Rendition
		return vt.handleSGR()
	case 'r': // DECSTBM - Set Top and Bottom Margins
		top := vt.getParam(0, 1) - 1
		bottom := vt.getParam(1, state.Height) - 1
		return []Action{{Type: ActionSetScrollRegion, Data: ScrollRegion{Top: top, Bottom: bottom}}}
	case 's': // SCOSC - Save Cursor Position
		return []Action{{Type: ActionSaveCursor}}
	case 'u': // SCORC - Restore Cursor Position
		return []Action{{Type: ActionRestoreCursor}}
	case 'h': // SM - Set Mode
		return vt.handleSetMode(true)
	case 'l': // RM - Reset Mode
		return vt.handleSetMode(false)
	case 'P': // DCH - Delete Character
		count := vt.getParam(0, 1)
		return []Action{{Type: ActionDeleteChar, Data: count}}
	case '@': // ICH - Insert Character
		count := vt.getParam(0, 1)
		return []Action{{Type: ActionInsertChar, Data: count}}
	case 'g': // TBC - Tab Clear
		mode := vt.getParam(0, 0)
		return []Action{{Type: ActionClearTabStop, Data: mode}}
	case 'n': // DSR - Device Status Report
		mode := vt.getParam(0, 0)
		switch mode {
		case 5: // Status Report
			// Report that terminal is OK
			response := "\x1b[0n"
			return []Action{{Type: ActionSendResponse, Data: response}}
		case 6: // Report cursor position
			// Response: ESC[<row>;<col>R
			row := state.CursorY + 1
			col := state.CursorX + 1
			response := fmt.Sprintf("\x1b[%d;%dR", row, col)
			return []Action{{Type: ActionSendResponse, Data: response}}
		case 15: // Report printer status
			// Report no printer
			response := "\x1b[?13n"
			return []Action{{Type: ActionSendResponse, Data: response}}
		case 25: // Report UDK status
			// Report UDKs are locked
			response := "\x1b[?21n"
			return []Action{{Type: ActionSendResponse, Data: response}}
		case 26: // Report keyboard status
			// Report North American keyboard
			response := "\x1b[?27;1n"
			return []Action{{Type: ActionSendResponse, Data: response}}
		}
		return nil
	case 't': // Window manipulation
		operation := vt.getParam(0, 0)
		switch operation {
		case 8: // Resize text area (from remote)
			// ESC[8;<height>;<width>t
			// We receive this but don't need to process it
			return nil
		case 14: // Report text area size in pixels (not supported)
			// Just return nil to avoid displaying garbage
			return nil
		case 18: // Report text area size in characters
			// Response: ESC[8;<height>;<width>t
			response := fmt.Sprintf("\x1b[8;%d;%dt", state.Height, state.Width)
			return []Action{{Type: ActionSendResponse, Data: response}}
		case 19: // Report screen size in characters
			// Response: ESC[9;<height>;<width>t
			response := fmt.Sprintf("\x1b[9;%d;%dt", state.Height, state.Width)
			return []Action{{Type: ActionSendResponse, Data: response}}
		default:
			// Ignore unknown window manipulation sequences
			// This prevents garbage output when receiving partial sequences
			return nil
		}
	case 'c': // DA - Device Attributes
		// Send appropriate response based on query type
		if len(vt.Intermediate) > 0 && vt.Intermediate[0] == '>' {
			// Secondary DA (ESC[>c)
			// Report as VT220: ESC[>1;10;0c
			response := "\x1b[>1;10;0c"
			return []Action{{Type: ActionSendResponse, Data: response}}
		} else if len(vt.Intermediate) > 0 && vt.Intermediate[0] == '?' {
			// Primary DA with '?' (ESC[?c)
			// Same as without '?'
			response := "\x1b[?62;1;2;6;7;8;9c" // VT220 with various options
			return []Action{{Type: ActionSendResponse, Data: response}}
		} else {
			// Primary DA (ESC[c or ESC[0c)
			// Report as VT220 compatible
			response := "\x1b[?62;1;2;6;7;8;9c"
			return []Action{{Type: ActionSendResponse, Data: response}}
		}
	default:
		return nil
	}
}

// parseParams parses parameter string into integer array
func (vt *VTParser) parseParams() {
	vt.Params = vt.Params[:0]

	if len(vt.Buffer) == 0 {
		return
	}

	paramStr := string(vt.Buffer)
	current := 0
	hasDigit := false

	for _, ch := range paramStr {
		if ch >= '0' && ch <= '9' {
			current = current*10 + int(ch-'0')
			hasDigit = true
		} else if ch == ';' {
			if hasDigit {
				vt.Params = append(vt.Params, current)
			} else {
				vt.Params = append(vt.Params, 0)
			}
			current = 0
			hasDigit = false
		}
	}

	if hasDigit {
		vt.Params = append(vt.Params, current)
	} else if len(paramStr) > 0 && paramStr[len(paramStr)-1] == ';' {
		vt.Params = append(vt.Params, 0)
	}
}

// getParam gets parameter at index with default value
func (vt *VTParser) getParam(index, defaultValue int) int {
	if index < len(vt.Params) {
		return vt.Params[index]
	}
	return defaultValue
}

// handleSGR handles Select Graphic Rendition sequences
func (vt *VTParser) handleSGR() []Action {
	if len(vt.Params) == 0 {
		// Reset all attributes
		return []Action{{Type: ActionSetAttribute, Data: AttributeChange{Reset: true}}}
	}

	var actions []Action
	for _, param := range vt.Params {
		action := vt.sgrParamToAction(param)
		if action != nil {
			actions = append(actions, *action)
		}
	}

	return actions
}

// sgrParamToAction converts SGR parameter to action
func (vt *VTParser) sgrParamToAction(param int) *Action {
	switch param {
	case 0: // Reset
		return &Action{Type: ActionSetAttribute, Data: AttributeChange{Reset: true}}
	case 1: // Bold
		return &Action{Type: ActionSetAttribute, Data: AttributeChange{Bold: &[]bool{true}[0]}}
	case 3: // Italic
		return &Action{Type: ActionSetAttribute, Data: AttributeChange{Italic: &[]bool{true}[0]}}
	case 4: // Underline
		return &Action{Type: ActionSetAttribute, Data: AttributeChange{Underline: &[]bool{true}[0]}}
	case 5: // Blink
		return &Action{Type: ActionSetAttribute, Data: AttributeChange{Blink: &[]bool{true}[0]}}
	case 7: // Reverse
		return &Action{Type: ActionSetAttribute, Data: AttributeChange{Reverse: &[]bool{true}[0]}}
	case 22: // Normal intensity (not bold)
		return &Action{Type: ActionSetAttribute, Data: AttributeChange{Bold: &[]bool{false}[0]}}
	case 23: // Not italic
		return &Action{Type: ActionSetAttribute, Data: AttributeChange{Italic: &[]bool{false}[0]}}
	case 24: // Not underlined
		return &Action{Type: ActionSetAttribute, Data: AttributeChange{Underline: &[]bool{false}[0]}}
	case 25: // Not blinking
		return &Action{Type: ActionSetAttribute, Data: AttributeChange{Blink: &[]bool{false}[0]}}
	case 27: // Not reversed
		return &Action{Type: ActionSetAttribute, Data: AttributeChange{Reverse: &[]bool{false}[0]}}
	default:
		if param >= 30 && param <= 37 { // Foreground colors
			color := Color(param - 30)
			return &Action{Type: ActionSetAttribute, Data: AttributeChange{Foreground: &color}}
		}
		if param >= 40 && param <= 47 { // Background colors
			color := Color(param - 40)
			return &Action{Type: ActionSetAttribute, Data: AttributeChange{Background: &color}}
		}
		if param >= 90 && param <= 97 { // Bright foreground colors
			color := Color(param - 90 + 8)
			return &Action{Type: ActionSetAttribute, Data: AttributeChange{Foreground: &color}}
		}
		if param >= 100 && param <= 107 { // Bright background colors
			color := Color(param - 100 + 8)
			return &Action{Type: ActionSetAttribute, Data: AttributeChange{Background: &color}}
		}
		return nil
	}
}

// handleSetMode handles mode setting sequences
func (vt *VTParser) handleSetMode(set bool) []Action {
	var actions []Action

	// Check if this is a private mode (starts with '?')
	isPrivate := len(vt.Intermediate) > 0 && vt.Intermediate[0] == '?'

	for _, param := range vt.Params {
		var mode string

		if isPrivate {
			// Private modes (DEC modes)
			switch param {
			case 1: // DECCKM - Cursor Keys Mode
				if set {
					mode = "cursor_app"
				} else {
					mode = "cursor_normal"
				}
			case 3: // DECCOLM - 132 Column Mode (not fully supported)
				continue
			case 4: // DECSCLM - Smooth Scrolling (not supported)
				continue
			case 5: // DECSCNM - Reverse Video
				if set {
					mode = "reverse_video"
				} else {
					mode = "normal_video"
				}
			case 6: // DECOM - Origin Mode
				if set {
					mode = "origin_mode"
				} else {
					mode = "absolute_mode"
				}
			case 7: // DECAWM - Auto Wrap Mode
				if set {
					mode = "autowrap_on"
				} else {
					mode = "autowrap_off"
				}
			case 25: // DECTCEM - Text Cursor Enable Mode
				if set {
					mode = "cursor_visible"
				} else {
					mode = "cursor_hidden"
				}
			case 47: // Use Alternate Screen Buffer (old style)
				if set {
					return []Action{{Type: ActionSwitchAltScreen, Data: true}}
				} else {
					return []Action{{Type: ActionSwitchAltScreen, Data: false}}
				}
			case 1000: // Mouse tracking
				if set {
					mode = "mouse_x10"
				} else {
					mode = "mouse_off"
				}
			case 1002: // Cell motion mouse tracking
				if set {
					mode = "mouse_btn_event"
				} else {
					mode = "mouse_off"
				}
			case 1003: // All motion mouse tracking
				if set {
					mode = "mouse_any_event"
				} else {
					mode = "mouse_off"
				}
			case 1047: // Use Alternate Screen Buffer (new style)
				if set {
					return []Action{{Type: ActionSwitchAltScreen, Data: true}}
				} else {
					return []Action{{Type: ActionSwitchAltScreen, Data: false}}
				}
			case 1048: // Save/Restore Cursor
				if set {
					return []Action{{Type: ActionSaveCursor}}
				} else {
					return []Action{{Type: ActionRestoreCursor}}
				}
			case 1049: // Alternative screen buffer + save/restore cursor
				if set {
					// Save cursor, switch to alt screen, clear it
					// Note: saveCursor and restoreCursor are handled as separate actions
					return []Action{
						{Type: ActionSaveCursor},
						{Type: ActionSwitchAltScreen, Data: true},
						{Type: ActionClearScreen, Data: 2},
					}
				} else {
					// Switch back to normal screen, restore cursor
					// The order is important: switch first, then restore cursor
					return []Action{
						{Type: ActionSwitchAltScreen, Data: false},
						{Type: ActionRestoreCursor},
					}
				}
			case 2004: // Bracketed Paste Mode
				if set {
					mode = "bracketed_paste_on"
				} else {
					mode = "bracketed_paste_off"
				}
			default:
				continue
			}
		} else {
			// Standard modes
			switch param {
			case 4: // IRM - Insert/Replace Mode
				if set {
					mode = "insert"
				} else {
					mode = "replace"
				}
			case 20: // LNM - Line Feed/New Line Mode
				if set {
					mode = "newline"
				} else {
					mode = "linefeed"
				}
			default:
				continue
			}
		}

		if mode != "" {
			actions = append(actions, Action{Type: ActionSetMode, Data: mode})
		}
	}

	return actions
}

// handleOSC processes Operating System Command sequences
func (vt *VTParser) handleOSC(b byte, screen *Screen, state *TerminalState) []Action {
	if b == 0x07 || b == 0x1B { // BEL or ESC (end of OSC)
		// TODO: Process OSC command
		vt.Reset()
		return nil
	}

	vt.Buffer = append(vt.Buffer, b)
	return nil
}

// handleDCS processes Device Control String sequences
func (vt *VTParser) handleDCS(b byte, screen *Screen, state *TerminalState) []Action {
	if b == 0x1B { // ESC (end of DCS)
		// TODO: Process DCS command
		vt.Reset()
		return nil
	}

	vt.Buffer = append(vt.Buffer, b)
	return nil
}

// Supporting data structures

// CursorMove represents cursor movement data
type CursorMove struct {
	Direction string
	Count     int
	Row       int
	Col       int
}

// AttributeChange represents attribute change data
type AttributeChange struct {
	Reset      bool
	Bold       *bool
	Italic     *bool
	Underline  *bool
	Blink      *bool
	Reverse    *bool
	Foreground *Color
	Background *Color
}

// ScrollRegion represents scroll region data
type ScrollRegion struct {
	Top    int
	Bottom int
}

// TerminalRenderer handles terminal display and rendering
type TerminalRenderer struct {
	screen   tcell.Screen
	terminal *TerminalEmulator
	mutex    sync.RWMutex
	running  bool
	events   chan tcell.Event
}

// NewTerminalRenderer creates a new terminal renderer
func NewTerminalRenderer(terminal *TerminalEmulator) (*TerminalRenderer, error) {
	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, fmt.Errorf("failed to create screen: %w", err)
	}

	return &TerminalRenderer{
		screen:   screen,
		terminal: terminal,
		events:   make(chan tcell.Event, 100),
	}, nil
}

// Start initializes and starts the terminal renderer
func (tr *TerminalRenderer) Start() error {
	tr.mutex.Lock()
	defer tr.mutex.Unlock()

	if tr.running {
		return fmt.Errorf("renderer is already running")
	}

	if err := tr.screen.Init(); err != nil {
		return fmt.Errorf("failed to initialize screen: %w", err)
	}

	tr.screen.SetStyle(tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite))
	tr.screen.Clear()

	tr.running = true

	// Start event handling goroutine
	go tr.handleEvents()

	return nil
}

// Stop stops the terminal renderer
func (tr *TerminalRenderer) Stop() error {
	tr.mutex.Lock()
	defer tr.mutex.Unlock()

	if !tr.running {
		return nil
	}

	tr.running = false
	tr.screen.Fini()
	close(tr.events)

	return nil
}

// Render renders the terminal screen
func (tr *TerminalRenderer) Render() error {
	tr.mutex.RLock()
	defer tr.mutex.RUnlock()

	if !tr.running {
		return fmt.Errorf("renderer is not running")
	}

	// Get terminal state and screen
	state := tr.terminal.GetState()
	screen := tr.terminal.screen

	// Clear screen if needed
	if screen.Dirty {
		tr.screen.Clear()
	}

	// Render each cell
	for y := 0; y < screen.Height; y++ {
		for x := 0; x < screen.Width; x++ {
			cell := screen.Buffer[y][x]

			// Skip continuation cells (they're part of the previous wide character)
			if cell.Char == 0 && x > 0 {
				// This is a continuation cell for a wide character
				continue
			}

			style := tr.attributesToStyle(cell.Attributes)

			// tcell's SetContent automatically handles wide characters
			// It will occupy two cells for wide characters and handle cursor positioning
			tr.screen.SetContent(x, y, cell.Char, nil, style)
		}
	}

	// Set cursor position
	tr.screen.ShowCursor(state.CursorX, state.CursorY)

	// Update screen
	tr.screen.Show()
	screen.Dirty = false

	return nil
}

// attributesToStyle converts TextAttributes to tcell.Style
func (tr *TerminalRenderer) attributesToStyle(attrs TextAttributes) tcell.Style {
	style := tcell.StyleDefault

	// Set colors
	fg := tr.colorToTcell(attrs.Foreground)
	bg := tr.colorToTcell(attrs.Background)
	style = style.Foreground(fg).Background(bg)

	// Set attributes
	if attrs.Bold {
		style = style.Bold(true)
	}
	if attrs.Italic {
		style = style.Italic(true)
	}
	if attrs.Underline {
		style = style.Underline(true)
	}
	if attrs.Reverse {
		style = style.Reverse(true)
	}
	if attrs.Blink {
		style = style.Blink(true)
	}

	return style
}

// colorToTcell converts Color to tcell.Color
func (tr *TerminalRenderer) colorToTcell(color Color) tcell.Color {
	switch color {
	case ColorBlack:
		return tcell.ColorBlack
	case ColorRed:
		return tcell.ColorMaroon
	case ColorGreen:
		return tcell.ColorGreen
	case ColorYellow:
		return tcell.ColorOlive
	case ColorBlue:
		return tcell.ColorNavy
	case ColorMagenta:
		return tcell.ColorPurple
	case ColorCyan:
		return tcell.ColorTeal
	case ColorWhite:
		return tcell.ColorSilver
	case ColorBrightBlack:
		return tcell.ColorGray
	case ColorBrightRed:
		return tcell.ColorRed
	case ColorBrightGreen:
		return tcell.ColorLime
	case ColorBrightYellow:
		return tcell.ColorYellow
	case ColorBrightBlue:
		return tcell.ColorBlue
	case ColorBrightMagenta:
		return tcell.ColorFuchsia
	case ColorBrightCyan:
		return tcell.ColorAqua
	case ColorBrightWhite:
		return tcell.ColorWhite
	default:
		return tcell.ColorDefault
	}
}

// handleEvents handles terminal events
func (tr *TerminalRenderer) handleEvents() {
	for tr.running {
		event := tr.screen.PollEvent()
		if event == nil {
			continue
		}

		select {
		case tr.events <- event:
		default:
			// Event channel is full, drop event
		}
	}
}

// GetEvent returns the next event from the event queue
func (tr *TerminalRenderer) GetEvent() tcell.Event {
	select {
	case event := <-tr.events:
		return event
	default:
		return nil
	}
}

// Resize resizes the terminal
func (tr *TerminalRenderer) Resize(width, height int) error {
	tr.mutex.Lock()
	defer tr.mutex.Unlock()

	// Resize terminal state
	if err := tr.terminal.Resize(width, height); err != nil {
		return err
	}

	// Resize screen buffer
	tr.terminal.screen = NewScreen(width, height)

	return nil
}

// GetSize returns the current terminal size
func (tr *TerminalRenderer) GetSize() (int, int) {
	return tr.screen.Size()
}

// Now implement the Terminal interface methods for TerminalEmulator

// Start starts the terminal emulator
func (te *TerminalEmulator) Start() error {
	if te.isRunning {
		return fmt.Errorf("terminal is already running")
	}

	te.isRunning = true
	te.state.IsRunning = true

	return nil
}

// Stop stops the terminal emulator
func (te *TerminalEmulator) Stop() error {
	if !te.isRunning {
		return nil
	}

	te.isRunning = false
	te.state.IsRunning = false

	return nil
}

// ProcessInput processes input from the user
func (te *TerminalEmulator) ProcessInput(input []byte) error {
	if !te.isRunning {
		return fmt.Errorf("terminal is not running")
	}

	// Send input to serial port
	if te.serialPort != nil && te.serialPort.IsOpen() {
		_, err := te.serialPort.Write(input)
		if err != nil {
			return fmt.Errorf("failed to write to serial port: %w", err)
		}

		// Log input to history
		if te.historyManager != nil {
			_ = te.historyManager.Write(input, history.DirectionOutput)
		}
	}

	return nil
}

// ProcessOutput processes output from the serial port
func (te *TerminalEmulator) ProcessOutput(output []byte) error {
	// Add panic recovery to prevent crashes
	defer func() {
		if r := recover(); r != nil {
			te.logDebug("PANIC in ProcessOutput: %v", r)
			// Reset parser state on panic
			te.parser.Reset()
			te.utf8Decoder.Reset()
		}
	}()

	// Lock for thread safety
	te.mu.Lock()
	defer te.mu.Unlock()

	if !te.isRunning {
		return fmt.Errorf("terminal is not running")
	}

	// Log output to history
	if te.historyManager != nil {
		_ = te.historyManager.Write(output, history.DirectionInput)
	}

	// Debug log the raw bytes received and decoder state (disabled for performance)
	// Uncomment for debugging UTF-8 issues
	// if len(output) > 0 {
	// 	hexBytes := fmt.Sprintf("%X", output)
	// 	te.logDebug("UTF-8 Raw bytes received (%d bytes): %s", len(output), hexBytes)
	// 	te.logDebug("Decoder state at start: buffered=%X, expected=%d, decoder_ptr=%p",
	// 		te.utf8Decoder.bytes, te.utf8Decoder.expected, te.utf8Decoder)
	// }

	// Process the output
	i := 0
	processedCount := 0
	for i < len(output) {
		b := output[i]
		processedCount++

		// Safety check for infinite loops
		if processedCount > len(output)*2 {
			te.logDebug("ERROR: Possible infinite loop in ProcessOutput, breaking. i=%d, len=%d", i, len(output))
			break
		}

		// Debug logging for escape sequences (disabled for performance)
		// if te.parser.State != StateGround {
		// 	// In escape sequence processing
		// 	if b >= 0x20 && b < 0x7F {
		// 		te.logDebug("ESC seq byte %d: 0x%02X '%c' state=%d", i, b, b, te.parser.State)
		// 	} else {
		// 		te.logDebug("ESC seq byte %d: 0x%02X state=%d", i, b, te.parser.State)
		// 	}
		// }

		// Special debug for backspace sequences (disabled for performance)
		// if b == 0x08 {
		// 	te.logDebug("Processing BACKSPACE (0x08) at byte[%d], cursor at (%d, %d)", i, te.state.CursorX, te.state.CursorY)
		// } else if b == 0x20 && i > 0 && output[i-1] == 0x08 {
		// 	te.logDebug("Processing SPACE after BACKSPACE at byte[%d]", i)
		// } else if b == 0x7F {
		// 	te.logDebug("Processing DEL (0x7F) at byte[%d], cursor at (%d, %d)", i, te.state.CursorX, te.state.CursorY)
		// }

		// Debug what byte we're processing (disabled for performance)
		// if b >= 0x80 || b < 0x20 {
		// 	te.logDebug("Processing byte[%d]: 0x%02X, parser state=%d, decoder: buffered=%X, expected=%d",
		// 		i, b, te.parser.State, te.utf8Decoder.bytes, te.utf8Decoder.expected)
		// }

		// If in ground state and this could be UTF-8, use custom decoder
		if te.parser.State == StateGround && b >= 0x80 {
			// Always use custom decoder for UTF-8 to handle partial sequences
			if r, complete := te.utf8Decoder.Decode(b); complete && r != 0 {
				te.executeAction(Action{Type: ActionPrint, Data: r})
			}
			i++
			continue
		}

		// Process through VT parser for everything else
		actions := te.parser.ParseByte(b, te.GetScreen(), &te.state, te.utf8Decoder)

		// Execute actions
		for _, action := range actions {
			// te.logDebug("Executing action: %v", action.Type)
			te.executeAction(action)
		}

		i++
	}

	// Log decoder state at end (disabled for performance)
	// if len(output) > 0 && te.utf8Decoder.expected > 0 {
	// 	te.logDebug("Decoder state at end: buffered=%X, expected=%d, decoder_ptr=%p",
	// 		te.utf8Decoder.bytes, te.utf8Decoder.expected, te.utf8Decoder)
	// }

	return nil
}

// logDebug logs debug messages to the configured logger
func (te *TerminalEmulator) logDebug(format string, args ...interface{}) {
	if te.logger != nil {
		te.logger.Debugf(format, args...)
	}
}

// executeAction executes a terminal action
func (te *TerminalEmulator) executeAction(action Action) {
	switch action.Type {
	case ActionPrint:
		te.printChar(action.Data.(rune))
	case ActionMoveCursor:
		te.moveCursor(action.Data.(CursorMove))
	case ActionClearScreen:
		te.clearScreen(action.Data.(int))
	case ActionClearLine:
		te.clearLine(action.Data.(int))
	case ActionSetAttribute:
		te.setAttribute(action.Data.(AttributeChange))
	case ActionScroll:
		te.scroll(action.Data.(string))
	case ActionSetMode:
		te.setMode(action.Data.(string))
	case ActionBell:
		// TODO: Implement bell
	case ActionReset:
		te.resetTerminal()
	case ActionTab:
		te.tab()
	case ActionNewline:
		te.newline()
	case ActionCarriageReturn:
		te.carriageReturn()
	case ActionBackspace:
		// te.logDebug("Executing backspace action at cursor pos (%d, %d)", te.state.CursorX, te.state.CursorY)
		te.backspace()
		// te.logDebug("After backspace, cursor at (%d, %d)", te.state.CursorX, te.state.CursorY)
	case ActionDeleteChar:
		te.deleteChar(action.Data.(int))
	case ActionInsertChar:
		te.insertChar(action.Data.(int))
	case ActionSetScrollRegion:
		te.setScrollRegion(action.Data.(ScrollRegion))
	case ActionSaveCursor:
		te.saveCursor()
	case ActionRestoreCursor:
		te.restoreCursor()
	case ActionSwitchAltScreen:
		te.switchAltScreen(action.Data.(bool))
	case ActionSendResponse:
		// Send response back to remote device
		if te.serialPort != nil && te.serialPort.IsOpen() {
			response := action.Data.(string)
			_, _ = te.serialPort.Write([]byte(response))
		}
	case ActionSetTabStop:
		te.setTabStop()
	case ActionClearTabStop:
		te.clearTabStop(action.Data.(int))
	}
}

// runeWidth returns the display width of a rune using the standard runewidth library
func runeWidth(r rune) int {
	return runewidth.RuneWidth(r)
}

// printChar prints a character at the current cursor position
func (te *TerminalEmulator) printChar(ch rune) {
	// Calculate character width
	charWidth := runeWidth(ch)

	// Debug logging for backspace sequence handling (disabled for performance)
	// if ch == ' ' {
	// 	te.logDebug("Printing space at cursor pos (%d, %d)", te.state.CursorX, te.state.CursorY)
	// }

	// For zero-width characters, don't advance cursor
	if charWidth == 0 {
		return
	}

	// Check if there's enough space for wide characters
	if charWidth == 2 && te.state.CursorX >= te.state.Width-1 {
		// Not enough space for wide character
		if te.state.LineWrap {
			// Line wrap enabled: move to next line
			te.newline()
			te.carriageReturn()
		} else {
			// Line wrap disabled: stay at last column
			te.state.CursorX = te.state.Width - 1
			return
		}
	} else if te.state.CursorX >= te.state.Width {
		if te.state.LineWrap {
			// Line wrap enabled: move to next line
			te.newline()
			te.carriageReturn()
		} else {
			// Line wrap disabled: don't write beyond edge
			return
		}
	}

	if te.state.CursorY >= te.state.Height {
		te.scroll("up")
		te.state.CursorY = te.state.Height - 1
	}

	// Get current screen buffer
	screen := te.GetScreen()

	// Bounds check before writing to buffer
	if te.state.CursorY >= 0 && te.state.CursorY < len(screen.Buffer) &&
		te.state.CursorX >= 0 && te.state.CursorX < len(screen.Buffer[te.state.CursorY]) {
		// Set character in screen buffer
		screen.Buffer[te.state.CursorY][te.state.CursorX] = Cell{
			Char:       ch,
			Attributes: te.state.Attributes,
			Dirty:      true,
		}
		screen.MarkDirty(te.state.CursorX, te.state.CursorY)
	} else {
		te.logDebug("printChar out of bounds: cursor=(%d,%d), screen=%dx%d",
			te.state.CursorX, te.state.CursorY, screen.Width, screen.Height)
		return
	}

	// For wide characters, mark the next cell as continuation
	if charWidth == 2 && te.state.CursorX+1 < te.state.Width {
		if te.state.CursorY >= 0 && te.state.CursorY < len(screen.Buffer) &&
			te.state.CursorX+1 >= 0 && te.state.CursorX+1 < len(screen.Buffer[te.state.CursorY]) {
			screen.Buffer[te.state.CursorY][te.state.CursorX+1] = Cell{
				Char:       0, // Use null character to indicate this cell is part of previous character
				Attributes: te.state.Attributes,
				Dirty:      true,
			}
			screen.MarkDirty(te.state.CursorX+1, te.state.CursorY)
		}
	}

	// Move cursor by character width
	te.state.CursorX += charWidth
	screen.Dirty = true
}

// moveCursor moves the cursor
func (te *TerminalEmulator) moveCursor(move CursorMove) {
	switch move.Direction {
	case "up":
		te.state.CursorY = max(0, te.state.CursorY-move.Count)
	case "down":
		te.state.CursorY = min(te.state.Height-1, te.state.CursorY+move.Count)
	case "left":
		te.state.CursorX = max(0, te.state.CursorX-move.Count)
	case "right":
		te.state.CursorX = min(te.state.Width-1, te.state.CursorX+move.Count)
	case "horizontal":
		// Move to absolute column position
		te.state.CursorX = min(te.state.Width-1, max(0, move.Col))
	case "absolute":
		// Ensure coordinates are within bounds
		// Some terminals send positions beyond screen size
		newX := move.Col
		newY := move.Row

		// Clamp to screen bounds
		if newX < 0 {
			newX = 0
		} else if newX >= te.state.Width {
			newX = te.state.Width - 1
		}

		if newY < 0 {
			newY = 0
		} else if newY >= te.state.Height {
			newY = te.state.Height - 1
		}

		te.state.CursorX = newX
		te.state.CursorY = newY
	}
	// Mark screen as dirty to force cursor update
	te.GetScreen().Dirty = true
}

// Clear clears the entire screen (public method)
func (te *TerminalEmulator) Clear() {
	te.clearScreen(2) // Clear entire screen
}

// clearScreen clears the screen
func (te *TerminalEmulator) clearScreen(mode int) {
	// Exit scroll mode for any clear operation
	if te.isScrolling {
		te.ExitScrollMode()
	}

	switch mode {
	case 0: // Clear from cursor to end of screen
		te.clearFromCursor()
	case 1: // Clear from beginning of screen to cursor
		te.clearToCursor()
	case 2: // Clear entire screen
		te.clearEntireScreen()
		// Always reset cursor to home position when clearing entire screen
		// This must be done AFTER clearEntireScreen
		te.state.CursorX = 0
		te.state.CursorY = 0

		// Log cursor position after reset
		if te.logger != nil {
			te.logger.Debugf("[clearScreen] Mode 2 - Cursor reset to (0,0) from (%d,%d)",
				te.state.CursorX, te.state.CursorY)
		}
	}

	// Force entire screen to be redrawn
	screen := te.GetScreen()
	screen.Dirty = true

	// Mark all lines as dirty when clearing
	screen.mutex.Lock()
	for y := 0; y < screen.Height; y++ {
		if screen.DirtyLines == nil {
			screen.DirtyLines = make(map[int]bool)
		}
		screen.DirtyLines[y] = true
		// Also mark all cells in the line as dirty for clear operations
		if y < len(screen.Buffer) {
			for x := 0; x < len(screen.Buffer[y]); x++ {
				screen.Buffer[y][x].Dirty = true
			}
		}
	}
	// Set dirty bounds to cover entire screen
	screen.DirtyMinY = 0
	screen.DirtyMaxY = screen.Height - 1
	screen.DirtyMinX = 0
	screen.DirtyMaxX = screen.Width - 1
	screen.mutex.Unlock()
}

// clearLine clears the current line
func (te *TerminalEmulator) clearLine(mode int) {
	y := te.state.CursorY
	screen := te.GetScreen()

	switch mode {
	case 0: // Clear from cursor to end of line
		for x := te.state.CursorX; x < te.state.Width; x++ {
			screen.Buffer[y][x] = Cell{Char: ' ', Attributes: DefaultTextAttributes(), Dirty: true}
			screen.MarkDirty(x, y)
		}
	case 1: // Clear from beginning of line to cursor
		for x := 0; x <= te.state.CursorX; x++ {
			screen.Buffer[y][x] = Cell{Char: ' ', Attributes: DefaultTextAttributes(), Dirty: true}
			screen.MarkDirty(x, y)
		}
	case 2: // Clear entire line
		for x := 0; x < te.state.Width; x++ {
			screen.Buffer[y][x] = Cell{Char: ' ', Attributes: DefaultTextAttributes(), Dirty: true}
			screen.MarkDirty(x, y)
		}
	}
	screen.Dirty = true
}

// setAttribute sets text attributes
func (te *TerminalEmulator) setAttribute(change AttributeChange) {
	if change.Reset {
		te.state.Attributes = DefaultTextAttributes()
		return
	}

	if change.Bold != nil {
		te.state.Attributes.Bold = *change.Bold
	}
	if change.Italic != nil {
		te.state.Attributes.Italic = *change.Italic
	}
	if change.Underline != nil {
		te.state.Attributes.Underline = *change.Underline
	}
	if change.Blink != nil {
		te.state.Attributes.Blink = *change.Blink
	}
	if change.Reverse != nil {
		te.state.Attributes.Reverse = *change.Reverse
	}
	if change.Foreground != nil {
		te.state.Attributes.Foreground = *change.Foreground
	}
	if change.Background != nil {
		te.state.Attributes.Background = *change.Background
	}
}

// scroll scrolls the screen
func (te *TerminalEmulator) scroll(direction string) {
	switch direction {
	case "up":
		te.scrollUp()
	case "down":
		te.scrollDown()
	}
	te.GetScreen().Dirty = true
}

// scrollUp scrolls the screen up by one line
func (te *TerminalEmulator) scrollUp() {
	screen := te.GetScreen()

	// Validate scroll region bounds
	if te.state.ScrollBottom >= len(screen.Buffer) {
		te.state.ScrollBottom = len(screen.Buffer) - 1
	}
	if te.state.ScrollTop < 0 {
		te.state.ScrollTop = 0
	}
	if te.state.ScrollTop > te.state.ScrollBottom {
		te.state.ScrollTop = 0
		te.state.ScrollBottom = len(screen.Buffer) - 1
	}

	// Save the top line to scrollback buffer if it's about to be lost
	if te.state.ScrollTop == 0 && len(screen.Buffer) > 0 {
		// Copy the top line to scrollback
		topLine := make([]Cell, len(screen.Buffer[0]))
		copy(topLine, screen.Buffer[0])
		te.scrollbackBuffer = append(te.scrollbackBuffer, topLine)

		// Trim scrollback if it exceeds maximum size
		if len(te.scrollbackBuffer) > te.scrollbackSize {
			te.scrollbackBuffer = te.scrollbackBuffer[1:]
		}
	}

	// Move all lines up within scroll region
	for y := te.state.ScrollTop; y < te.state.ScrollBottom && y < len(screen.Buffer)-1; y++ {
		if y+1 < len(screen.Buffer) {
			copy(screen.Buffer[y], screen.Buffer[y+1])
			// Mark entire line as dirty after copying
			screen.MarkLineDirty(y)
		}
	}

	// Clear bottom line of scroll region
	if te.state.ScrollBottom >= 0 && te.state.ScrollBottom < len(screen.Buffer) {
		line := screen.Buffer[te.state.ScrollBottom]
		for x := 0; x < len(line); x++ {
			line[x] = Cell{Char: ' ', Attributes: DefaultTextAttributes(), Dirty: true}
		}
		// Mark the entire bottom line as dirty
		screen.MarkLineDirty(te.state.ScrollBottom)
	}

	// Make sure screen is marked as dirty
	screen.Dirty = true
}

// scrollDown scrolls the screen down by one line
func (te *TerminalEmulator) scrollDown() {
	screen := te.GetScreen()

	// Validate scroll region bounds
	if te.state.ScrollBottom >= len(screen.Buffer) {
		te.state.ScrollBottom = len(screen.Buffer) - 1
	}
	if te.state.ScrollTop < 0 {
		te.state.ScrollTop = 0
	}
	if te.state.ScrollTop > te.state.ScrollBottom {
		te.state.ScrollTop = 0
		te.state.ScrollBottom = len(screen.Buffer) - 1
	}

	// Move all lines down within scroll region
	for y := te.state.ScrollBottom; y > te.state.ScrollTop; y-- {
		if y > 0 && y < len(screen.Buffer) && y-1 >= 0 && y-1 < len(screen.Buffer) {
			copy(screen.Buffer[y], screen.Buffer[y-1])
			// Mark entire line as dirty after copying
			// Use actual buffer width, not state width
			lineWidth := len(screen.Buffer[y])
			for x := 0; x < lineWidth; x++ {
				screen.Buffer[y][x].Dirty = true
				screen.MarkDirty(x, y)
			}
		}
	}

	// Clear top line of scroll region
	if te.state.ScrollTop >= 0 && te.state.ScrollTop < len(screen.Buffer) {
		line := screen.Buffer[te.state.ScrollTop]
		for x := 0; x < len(line); x++ {
			line[x] = Cell{Char: ' ', Attributes: DefaultTextAttributes(), Dirty: true}
			screen.MarkDirty(x, te.state.ScrollTop)
		}
	}
}

// EnterScrollMode enters scrollback viewing mode
func (te *TerminalEmulator) EnterScrollMode() {
	te.isScrolling = true
	// Set absolute position to current end of scrollback buffer
	// This fixes the view position even as new data arrives
	te.scrollPosition = len(te.scrollbackBuffer)
	te.scrollOffset = 0 // Start at current view
}

// ExitScrollMode exits scrollback viewing mode
func (te *TerminalEmulator) ExitScrollMode() {
	te.isScrolling = false
	te.scrollOffset = 0
	te.scrollPosition = 0
	te.GetScreen().Dirty = true
}

// ScrollUp scrolls up n lines in the scrollback buffer
func (te *TerminalEmulator) ScrollUp(n int) {
	if !te.isScrolling {
		te.EnterScrollMode()
	}

	// Move position up (back in history)
	te.scrollPosition -= n
	if te.scrollPosition < 0 {
		te.scrollPosition = 0
	}
	// Update offset based on new position
	te.scrollOffset = len(te.scrollbackBuffer) - te.scrollPosition
	te.GetScreen().Dirty = true
}

// ScrollDown scrolls down n lines in the scrollback buffer
func (te *TerminalEmulator) ScrollDown(n int) {
	if !te.isScrolling {
		return
	}

	// Calculate the maximum valid position (at the bottom of current view)
	maxPosition := len(te.scrollbackBuffer)

	// Move position down (forward towards newer data)
	te.scrollPosition += n

	// Ensure position doesn't go beyond the maximum valid position
	if te.scrollPosition > maxPosition {
		te.scrollPosition = maxPosition
	}

	// Update offset based on new position
	te.scrollOffset = len(te.scrollbackBuffer) - te.scrollPosition

	// Ensure offset never goes negative
	if te.scrollOffset < 0 {
		te.scrollOffset = 0
		// If offset would be negative, we're at the bottom
		// Adjust position to be exactly at the bottom
		te.scrollPosition = len(te.scrollbackBuffer)
	}

	te.GetScreen().Dirty = true
}

// ScrollToTop scrolls to the top of the scrollback buffer
func (te *TerminalEmulator) ScrollToTop() {
	if !te.isScrolling {
		te.EnterScrollMode()
	}
	te.scrollPosition = 0
	te.scrollOffset = len(te.scrollbackBuffer)
	te.GetScreen().Dirty = true
}

// ScrollToBottom scrolls to the bottom (stays in scroll mode)
func (te *TerminalEmulator) ScrollToBottom() {
	if !te.isScrolling {
		te.EnterScrollMode()
	}
	// Set position to the end of scrollback buffer (shows current screen)
	te.scrollPosition = len(te.scrollbackBuffer)
	te.scrollOffset = 0
	te.GetScreen().Dirty = true
}

// IsScrolling returns whether the terminal is in scroll mode
func (te *TerminalEmulator) IsScrolling() bool {
	return te.isScrolling
}

// GetScrollPosition returns current scroll position info
func (te *TerminalEmulator) GetScrollPosition() (current, total int) {
	if !te.isScrolling {
		return 0, len(te.scrollbackBuffer)
	}
	return te.scrollOffset, len(te.scrollbackBuffer)
}

// GetScrollbackBuffer returns a view of the screen including scrollback
func (te *TerminalEmulator) GetScrollbackView() [][]Cell {
	screen := te.GetScreen()

	if !te.isScrolling || (te.scrollPosition >= len(te.scrollbackBuffer) && te.scrollOffset == 0) {
		// Return normal screen view when not scrolling or at bottom
		return screen.Buffer
	}

	// Create a view combining scrollback and current screen
	viewHeight := screen.Height
	view := make([][]Cell, viewHeight)

	// Use absolute position to maintain stable view
	startIdx := te.scrollPosition

	for i := 0; i < viewHeight; i++ {
		lineIdx := startIdx + i
		if lineIdx < 0 {
			// Before scrollback starts, show empty lines
			view[i] = make([]Cell, screen.Width)
			for j := range view[i] {
				view[i][j] = Cell{Char: ' ', Attributes: DefaultTextAttributes()}
			}
		} else if lineIdx < len(te.scrollbackBuffer) {
			// Show from scrollback
			view[i] = te.scrollbackBuffer[lineIdx]
		} else {
			// Show from current screen
			screenIdx := lineIdx - len(te.scrollbackBuffer)
			if screenIdx < len(screen.Buffer) {
				view[i] = screen.Buffer[screenIdx]
			} else {
				view[i] = make([]Cell, screen.Width)
				for j := range view[i] {
					view[i][j] = Cell{Char: ' ', Attributes: DefaultTextAttributes()}
				}
			}
		}
	}

	return view
}

// ClearScrollback clears the scrollback buffer
func (te *TerminalEmulator) ClearScrollback() {
	te.scrollbackBuffer = make([][]Cell, 0, te.scrollbackSize)
	te.ExitScrollMode()
}

// GetAllLines returns all lines including scrollback buffer
func (te *TerminalEmulator) GetAllLines() [][]Cell {
	var allLines [][]Cell

	// Add scrollback buffer lines
	allLines = append(allLines, te.scrollbackBuffer...)

	// Add current screen lines
	if te.screen != nil {
		allLines = append(allLines, te.screen.Buffer...)
	}

	return allLines
}

// SetLineWrap enables or disables line wrapping
func (te *TerminalEmulator) SetLineWrap(enabled bool) {
	te.state.LineWrap = enabled
}

// SetScrollbackSize sets the maximum number of lines in scrollback buffer
func (te *TerminalEmulator) SetScrollbackSize(size int) {
	if size < 100 {
		size = 100 // Minimum size
	}
	if size > 1000000 {
		size = 1000000 // Maximum 1 million lines to prevent excessive memory use
	}
	te.scrollbackSize = size

	// Trim existing buffer if it exceeds new size
	if len(te.scrollbackBuffer) > size {
		te.scrollbackBuffer = te.scrollbackBuffer[len(te.scrollbackBuffer)-size:]
	}
}

// GetScrollbackSize returns the maximum number of lines in scrollback buffer
func (te *TerminalEmulator) GetScrollbackSize() int {
	return te.scrollbackSize
}

// setMode sets terminal mode
func (te *TerminalEmulator) setMode(mode string) {
	switch mode {
	case "cursor_visible":
		// TODO: Implement cursor visibility
	case "cursor_hidden":
		// TODO: Implement cursor visibility
	case "mouse_x10":
		oldMode := te.state.MouseMode
		te.state.MouseMode = MouseModeX10
		te.logDebug("Mouse mode changed: %v -> %v (X10)", oldMode, te.state.MouseMode)
		if te.onMouseModeChange != nil {
			te.onMouseModeChange(MouseModeX10)
		}
	case "mouse_btn_event":
		oldMode := te.state.MouseMode
		te.state.MouseMode = MouseModeBtnEvent
		te.logDebug("Mouse mode changed: %v -> %v (Button Event)", oldMode, te.state.MouseMode)
		if te.onMouseModeChange != nil {
			te.onMouseModeChange(MouseModeBtnEvent)
		}
	case "mouse_any_event":
		oldMode := te.state.MouseMode
		te.state.MouseMode = MouseModeAnyEvent
		te.logDebug("Mouse mode changed: %v -> %v (Any Event)", oldMode, te.state.MouseMode)
		if te.onMouseModeChange != nil {
			te.onMouseModeChange(MouseModeAnyEvent)
		}
	case "mouse_off":
		oldMode := te.state.MouseMode
		te.state.MouseMode = MouseModeOff
		te.logDebug("Mouse mode changed: %v -> %v (Off)", oldMode, te.state.MouseMode)
		if te.onMouseModeChange != nil {
			te.onMouseModeChange(MouseModeOff)
		}
	}
}

// tab moves cursor to next tab stop
func (te *TerminalEmulator) tab() {
	// Find next tab stop after current position
	nextTab := -1
	for col := te.state.CursorX + 1; col < te.state.Width; col++ {
		if te.tabStops[col] {
			nextTab = col
			break
		}
	}

	if nextTab != -1 {
		te.state.CursorX = nextTab
	} else {
		// No tab stop found, move to end of line
		te.state.CursorX = te.state.Width - 1
	}
}

// newline moves cursor to next line
func (te *TerminalEmulator) newline() {
	// Ensure scroll region is valid based on actual buffer size
	screen := te.GetScreen()
	bufferHeight := len(screen.Buffer)
	if bufferHeight == 0 {
		return
	}

	if te.state.ScrollBottom >= bufferHeight {
		te.state.ScrollBottom = bufferHeight - 1
	}
	if te.state.ScrollTop >= bufferHeight {
		te.state.ScrollTop = 0
	}

	te.state.CursorY++
	if te.state.CursorY >= te.state.Height {
		te.scroll("up")
		te.state.CursorY = te.state.Height - 1
	}
}

// carriageReturn moves cursor to beginning of line
func (te *TerminalEmulator) carriageReturn() {
	te.state.CursorX = 0
}

// backspace moves cursor back one position
func (te *TerminalEmulator) backspace() {
	if te.state.CursorX > 0 {
		// Just move cursor back one position
		// Don't try to be smart about wide characters here
		// The terminal echo will handle the actual deletion properly
		te.state.CursorX--
		// Mark screen as dirty to update cursor position
		te.GetScreen().Dirty = true
	}
}

// deleteChar deletes characters at cursor position
func (te *TerminalEmulator) deleteChar(count int) {
	y := te.state.CursorY
	x := te.state.CursorX
	screen := te.GetScreen()

	// Shift characters left
	for i := x; i < te.state.Width-count; i++ {
		screen.Buffer[y][i] = screen.Buffer[y][i+count]
	}

	// Clear rightmost characters
	for i := te.state.Width - count; i < te.state.Width; i++ {
		screen.Buffer[y][i] = Cell{Char: ' ', Attributes: DefaultTextAttributes()}
	}

	screen.Dirty = true
}

// insertChar inserts blank characters at cursor position
func (te *TerminalEmulator) insertChar(count int) {
	y := te.state.CursorY
	x := te.state.CursorX
	screen := te.GetScreen()

	// Shift characters right
	for i := te.state.Width - 1; i >= x+count; i-- {
		screen.Buffer[y][i] = screen.Buffer[y][i-count]
	}

	// Clear inserted characters
	for i := x; i < x+count && i < te.state.Width; i++ {
		screen.Buffer[y][i] = Cell{Char: ' ', Attributes: DefaultTextAttributes()}
	}

	screen.Dirty = true
}

// setScrollRegion sets the scroll region
func (te *TerminalEmulator) setScrollRegion(region ScrollRegion) {
	// Use actual buffer height instead of state height
	screen := te.GetScreen()
	bufferHeight := len(screen.Buffer)
	if bufferHeight == 0 {
		return
	}

	maxHeight := bufferHeight - 1
	te.state.ScrollTop = max(0, min(maxHeight, region.Top))
	te.state.ScrollBottom = max(te.state.ScrollTop, min(maxHeight, region.Bottom))
}

// clearFromCursor clears from cursor to end of screen
func (te *TerminalEmulator) clearFromCursor() {
	screen := te.GetScreen()

	// Clear from cursor to end of current line
	for x := te.state.CursorX; x < te.state.Width; x++ {
		screen.Buffer[te.state.CursorY][x] = Cell{Char: ' ', Attributes: DefaultTextAttributes(), Dirty: true}
		screen.MarkDirty(x, te.state.CursorY)
	}

	// Clear all lines below current line
	for y := te.state.CursorY + 1; y < te.state.Height; y++ {
		for x := 0; x < te.state.Width; x++ {
			screen.Buffer[y][x] = Cell{Char: ' ', Attributes: DefaultTextAttributes(), Dirty: true}
			screen.MarkDirty(x, y)
		}
	}
	screen.Dirty = true
}

// clearToCursor clears from beginning of screen to cursor
func (te *TerminalEmulator) clearToCursor() {
	screen := te.GetScreen()

	// Clear all lines above current line
	for y := 0; y < te.state.CursorY; y++ {
		for x := 0; x < te.state.Width; x++ {
			screen.Buffer[y][x] = Cell{Char: ' ', Attributes: DefaultTextAttributes(), Dirty: true}
			screen.MarkDirty(x, y)
		}
	}

	// Clear from beginning of current line to cursor
	for x := 0; x <= te.state.CursorX; x++ {
		screen.Buffer[te.state.CursorY][x] = Cell{Char: ' ', Attributes: DefaultTextAttributes(), Dirty: true}
		screen.MarkDirty(x, te.state.CursorY)
	}
	screen.Dirty = true
}

// clearEntireScreen clears the entire screen
func (te *TerminalEmulator) clearEntireScreen() {
	screen := te.GetScreen()

	// Debug logging
	if te.logger != nil {
		te.logger.Debugf("[clearEntireScreen] Start - isScrolling=%v, scrollbackLen=%d, scrollPos=%d",
			te.isScrolling, len(te.scrollbackBuffer), te.scrollPosition)
	}

	// Exit scroll mode if active
	if te.isScrolling {
		te.ExitScrollMode()
	}

	// Save current screen to scrollback before clearing
	// This preserves history like most terminal emulators
	if len(screen.Buffer) > 0 {
		for y := 0; y < te.state.Height && y < len(screen.Buffer); y++ {
			// Only save non-empty lines
			hasContent := false
			for x := 0; x < len(screen.Buffer[y]); x++ {
				if screen.Buffer[y][x].Char != ' ' && screen.Buffer[y][x].Char != 0 {
					hasContent = true
					break
				}
			}
			if hasContent {
				lineCopy := make([]Cell, len(screen.Buffer[y]))
				copy(lineCopy, screen.Buffer[y])
				te.scrollbackBuffer = append(te.scrollbackBuffer, lineCopy)

				// Trim scrollback if it exceeds maximum size
				if len(te.scrollbackBuffer) > te.scrollbackSize {
					te.scrollbackBuffer = te.scrollbackBuffer[1:]
				}
			}
		}
	}

	// Clear all cells and mark them as dirty
	for y := 0; y < te.state.Height && y < len(screen.Buffer); y++ {
		for x := 0; x < te.state.Width && x < len(screen.Buffer[y]); x++ {
			screen.Buffer[y][x] = Cell{Char: ' ', Attributes: DefaultTextAttributes(), Dirty: true}
		}
		// Mark entire line as dirty to ensure it gets redrawn
		screen.MarkLineDirty(y)
	}

	// Reset scroll region when clearing entire screen
	te.state.ScrollTop = 0
	te.state.ScrollBottom = te.state.Height - 1

	// IMPORTANT: Reset cursor position to home (0,0) when clearing entire screen
	// This should be done here, not later
	te.state.CursorX = 0
	te.state.CursorY = 0

	// Reset scroll position to view the current (now empty) screen
	te.scrollPosition = len(te.scrollbackBuffer)
	te.scrollOffset = 0

	screen.Dirty = true

	// Mark this as a clear screen operation for special handling
	screen.mutex.Lock()
	screen.JustCleared = true
	screen.mutex.Unlock()

	// Debug logging
	if te.logger != nil {
		te.logger.Debugf("[clearEntireScreen] End - scrollbackLen=%d, scrollPos=%d, cursor=(%d,%d), JustCleared=true",
			len(te.scrollbackBuffer), te.scrollPosition, te.state.CursorX, te.state.CursorY)
	}
}

// resetTerminal resets the terminal to its initial state
func (te *TerminalEmulator) resetTerminal() {
	// Debug logging
	if te.logger != nil {
		te.logger.Debugf("[resetTerminal] Resetting terminal to initial state")
	}

	// Exit scroll mode if active
	if te.isScrolling {
		te.ExitScrollMode()
	}

	// Clear the entire screen
	te.clearEntireScreen()

	// Reset cursor position to home
	te.state.CursorX = 0
	te.state.CursorY = 0

	// Reset all terminal state to defaults
	te.state.Attributes = DefaultTextAttributes()
	te.state.ScrollTop = 0
	te.state.ScrollBottom = te.state.Height - 1
	te.state.LineWrap = true
	te.state.MouseMode = MouseModeOff

	// Clear saved state
	te.savedState = nil

	// Reset to main screen if using alternate screen
	if te.useAltScreen {
		te.useAltScreen = false
		// Swap screens back
		te.screen, te.altScreen = te.altScreen, te.screen
	}

	// Clear tab stops and set defaults (every 8 columns)
	te.tabStops = make(map[int]bool)
	for i := 8; i < te.state.Width; i += 8 {
		te.tabStops[i] = true
	}

	// Clear the scrollback buffer
	te.scrollbackBuffer = make([][]Cell, 0, te.scrollbackSize)
	te.scrollOffset = 0
	te.scrollPosition = 0

	// Reset parser state
	if te.parser != nil {
		te.parser.Reset()
	}

	// Reset UTF-8 decoder
	if te.utf8Decoder != nil {
		te.utf8Decoder.Reset()
	}

	// Mark screen as needing full redraw
	screen := te.GetScreen()
	screen.JustCleared = true
	screen.Dirty = true

	if te.logger != nil {
		te.logger.Debugf("[resetTerminal] Terminal reset complete")
	}
}

// Resize resizes the terminal
func (te *TerminalEmulator) Resize(width, height int) error {
	if width <= 0 || height <= 0 {
		return fmt.Errorf("invalid dimensions: %dx%d", width, height)
	}

	// Helper function to resize a screen buffer
	resizeScreen := func(oldScreen *Screen) *Screen {
		newScreen := NewScreen(width, height)

		// Copy existing content
		copyHeight := min(height, oldScreen.Height)
		copyWidth := min(width, oldScreen.Width)

		for y := 0; y < copyHeight && y < len(oldScreen.Buffer) && y < len(newScreen.Buffer); y++ {
			for x := 0; x < copyWidth && x < len(oldScreen.Buffer[y]) && x < len(newScreen.Buffer[y]); x++ {
				newScreen.Buffer[y][x] = oldScreen.Buffer[y][x]
			}
		}

		return newScreen
	}

	// Resize both screen buffers
	te.screen = resizeScreen(te.screen)
	te.altScreen = resizeScreen(te.altScreen)

	// Update terminal state
	te.state.Width = width
	te.state.Height = height

	// Adjust cursor position if necessary
	te.state.CursorX = min(te.state.CursorX, width-1)
	te.state.CursorY = min(te.state.CursorY, height-1)

	// Adjust scroll region based on actual buffer size
	bufferHeight := len(te.screen.Buffer)
	if bufferHeight > 0 {
		te.state.ScrollBottom = min(height-1, bufferHeight-1)
	} else {
		te.state.ScrollBottom = height - 1
	}
	te.state.ScrollTop = 0

	// Rebuild tab stops for new width
	oldTabStops := te.tabStops
	te.tabStops = make(map[int]bool)

	// Copy existing tab stops that are still within bounds
	for col := range oldTabStops {
		if col < width {
			te.tabStops[col] = true
		}
	}

	// Ensure default tab stops exist
	for i := 8; i < width; i += 8 {
		if _, exists := te.tabStops[i]; !exists {
			te.tabStops[i] = true
		}
	}

	return nil
}

// EnableMouse enables or disables mouse support
func (te *TerminalEmulator) EnableMouse(enable bool) error {
	if enable {
		te.state.MouseMode = MouseModeX10
	} else {
		te.state.MouseMode = MouseModeOff
	}
	return nil
}

// GetState returns the current terminal state
func (te *TerminalEmulator) GetState() TerminalState {
	te.mu.RLock()
	defer te.mu.RUnlock()
	return te.state
}

// GetScreen returns the terminal screen buffer
func (te *TerminalEmulator) GetScreen() *Screen {
	// Note: No lock here since it's called internally by methods that already hold the lock
	// External callers should be aware that the returned screen can be modified
	if te.useAltScreen {
		return te.altScreen
	}
	return te.screen
}

// saveCursor saves the current cursor position and attributes
func (te *TerminalEmulator) saveCursor() {
	savedState := te.state // Create a copy
	te.savedState = &savedState
}

// restoreCursor restores the saved cursor position and attributes
func (te *TerminalEmulator) restoreCursor() {
	if te.savedState != nil {
		// Restore cursor position and attributes
		te.state.CursorX = te.savedState.CursorX
		te.state.CursorY = te.savedState.CursorY
		te.state.Attributes = te.savedState.Attributes
	}
}

// setTabStop sets a tab stop at the current cursor position
func (te *TerminalEmulator) setTabStop() {
	te.tabStops[te.state.CursorX] = true
}

// clearTabStop clears tab stops based on mode
func (te *TerminalEmulator) clearTabStop(mode int) {
	switch mode {
	case 0: // Clear tab stop at current position
		delete(te.tabStops, te.state.CursorX)
	case 3: // Clear all tab stops
		te.tabStops = make(map[int]bool)
		// Restore default tab stops every 8 columns
		for i := 8; i < te.state.Width; i += 8 {
			te.tabStops[i] = true
		}
	}
}

// switchAltScreen switches between main and alternative screen buffers
func (te *TerminalEmulator) switchAltScreen(useAlt bool) {
	if useAlt && !te.useAltScreen {
		// Switch to alternative screen

		// Debug logging
		if te.logger != nil {
			te.logger.Debugf("[switchAltScreen] Switching to alternate screen")
		}

		// Clear alternative screen first for fresh start
		altScreen := te.altScreen
		for y := 0; y < altScreen.Height && y < len(altScreen.Buffer); y++ {
			for x := 0; x < altScreen.Width && x < len(altScreen.Buffer[y]); x++ {
				altScreen.Buffer[y][x] = Cell{
					Char:       ' ',
					Attributes: DefaultTextAttributes(),
					Dirty:      true,
				}
			}
		}
		altScreen.Dirty = true

		// Now switch to alt screen
		te.useAltScreen = true

		// Reset cursor to top-left for alt screen
		te.state.CursorX = 0
		te.state.CursorY = 0

	} else if !useAlt && te.useAltScreen {
		// Switch back to normal screen

		// Debug logging
		if te.logger != nil {
			te.logger.Debugf("[switchAltScreen] Switching back to main screen")
		}

		te.useAltScreen = false

		// Mark the main screen as needing full redraw
		// This ensures the main screen content is properly restored
		te.screen.Dirty = true
		for y := 0; y < te.screen.Height && y < len(te.screen.Buffer); y++ {
			te.screen.MarkLineDirty(y)
			// Mark all cells as dirty to force redraw
			for x := 0; x < te.screen.Width && x < len(te.screen.Buffer[y]); x++ {
				te.screen.Buffer[y][x].Dirty = true
			}
		}

		// Force update dirty bounds
		te.screen.mutex.Lock()
		te.screen.DirtyMinX = 0
		te.screen.DirtyMaxX = te.screen.Width - 1
		te.screen.DirtyMinY = 0
		te.screen.DirtyMaxY = te.screen.Height - 1
		te.screen.mutex.Unlock()

		// Note: Cursor position should remain where it was on the main screen
		// The sequences ?1049 handles cursor save/restore separately
	}
}

// SetState sets the terminal state
func (te *TerminalEmulator) SetState(state TerminalState) error {
	if err := state.Validate(); err != nil {
		return err
	}

	te.state = state
	return nil
}

// IsRunning returns true if the terminal is running
func (te *TerminalEmulator) IsRunning() bool {
	return te.isRunning
}

// Helper functions

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// MouseEvent represents a mouse event
type MouseEvent struct {
	X      int
	Y      int
	Button MouseButton
	Action MouseAction
	Mods   tcell.ModMask
}

// MouseButton represents mouse buttons
type MouseButton int

const (
	MouseButtonNone MouseButton = iota
	MouseButtonLeft
	MouseButtonMiddle
	MouseButtonRight
	MouseButtonWheelUp
	MouseButtonWheelDown
)

// MouseAction represents mouse actions
type MouseAction int

const (
	MouseActionPress MouseAction = iota
	MouseActionRelease
	MouseActionMove
	MouseActionDrag
)

// String returns the string representation of MouseButton
func (mb MouseButton) String() string {
	switch mb {
	case MouseButtonNone:
		return "none"
	case MouseButtonLeft:
		return "left"
	case MouseButtonMiddle:
		return "middle"
	case MouseButtonRight:
		return "right"
	case MouseButtonWheelUp:
		return "wheel_up"
	case MouseButtonWheelDown:
		return "wheel_down"
	default:
		return "unknown"
	}
}

// String returns the string representation of MouseAction
func (ma MouseAction) String() string {
	switch ma {
	case MouseActionPress:
		return "press"
	case MouseActionRelease:
		return "release"
	case MouseActionMove:
		return "move"
	case MouseActionDrag:
		return "drag"
	default:
		return "unknown"
	}
}

// MouseHandler handles mouse events and converts them to terminal sequences
type MouseHandler struct {
	mode        MouseMode
	lastX       int
	lastY       int
	buttonState map[MouseButton]bool
	dragButton  MouseButton
}

// NewMouseHandler creates a new mouse handler
func NewMouseHandler() *MouseHandler {
	return &MouseHandler{
		mode:        MouseModeOff,
		buttonState: make(map[MouseButton]bool),
		dragButton:  MouseButtonNone,
	}
}

// SetMode sets the mouse mode
func (mh *MouseHandler) SetMode(mode MouseMode) {
	mh.mode = mode
}

// GetMode returns the current mouse mode
func (mh *MouseHandler) GetMode() MouseMode {
	return mh.mode
}

// ProcessTcellEvent processes a tcell mouse event and returns terminal sequences
func (mh *MouseHandler) ProcessTcellEvent(event *tcell.EventMouse) []byte {
	if mh.mode == MouseModeOff {
		return nil
	}

	x, y := event.Position()
	buttons := event.Buttons()

	mouseEvent := mh.tcellToMouseEvent(x, y, buttons)
	return mh.mouseEventToSequence(mouseEvent)
}

// tcellToMouseEvent converts tcell event to MouseEvent
func (mh *MouseHandler) tcellToMouseEvent(x, y int, buttons tcell.ButtonMask) MouseEvent {
	var button MouseButton
	var action MouseAction

	// Determine which button is currently pressed
	currentButton := MouseButtonNone
	switch {
	case buttons&tcell.Button1 != 0:
		currentButton = MouseButtonLeft
	case buttons&tcell.Button2 != 0:
		currentButton = MouseButtonMiddle
	case buttons&tcell.Button3 != 0:
		currentButton = MouseButtonRight
	case buttons&tcell.WheelUp != 0:
		currentButton = MouseButtonWheelUp
	case buttons&tcell.WheelDown != 0:
		currentButton = MouseButtonWheelDown
	}

	// Handle button state changes
	if currentButton == MouseButtonNone {
		// No button pressed now - check if we had a button pressed before
		if mh.dragButton != MouseButtonNone {
			// Button was released
			button = mh.dragButton
			action = MouseActionRelease
			mh.buttonState[mh.dragButton] = false
			// Debug: log release detection
			// fmt.Printf("Mouse: Release detected for button %v\n", button)
			mh.dragButton = MouseButtonNone
		} else if x != mh.lastX || y != mh.lastY {
			// Mouse moved without button
			button = MouseButtonNone
			action = MouseActionMove
		} else {
			// No change
			button = MouseButtonNone
			action = MouseActionMove
		}
	} else {
		// Button is pressed
		button = currentButton
		wasPressed := mh.buttonState[button]

		if !wasPressed {
			// Button just pressed
			action = MouseActionPress
			mh.buttonState[button] = true
			if button != MouseButtonWheelUp && button != MouseButtonWheelDown {
				mh.dragButton = button
			}
		} else if x != mh.lastX || y != mh.lastY {
			// Button held and mouse moved (drag)
			action = MouseActionDrag
		} else {
			// Button held but no movement
			action = MouseActionPress // Keep reporting as press
		}
	}

	mh.lastX = x
	mh.lastY = y

	return MouseEvent{
		X:      x,
		Y:      y,
		Button: button,
		Action: action,
	}
}

// mouseEventToSequence converts MouseEvent to terminal escape sequence
func (mh *MouseHandler) mouseEventToSequence(event MouseEvent) []byte {
	switch mh.mode {
	case MouseModeX10:
		return mh.generateX10Sequence(event)
	case MouseModeVT200:
		return mh.generateVT200Sequence(event)
	case MouseModeBtnEvent:
		return mh.generateBtnEventSequence(event)
	case MouseModeAnyEvent:
		return mh.generateAnyEventSequence(event)
	default:
		return nil
	}
}

// generateX10Sequence generates X10 mouse sequence
func (mh *MouseHandler) generateX10Sequence(event MouseEvent) []byte {
	// X10 mode only reports button press
	if event.Action != MouseActionPress {
		return nil
	}

	cb := mh.buttonToX10Code(event.Button)
	if cb == -1 {
		return nil
	}

	// ESC[M + button + x + y (all offset by 32)
	return []byte{
		0x1B, '[', 'M',
		byte(cb + 32),
		byte(event.X + 33), // 1-based + 32 offset
		byte(event.Y + 33), // 1-based + 32 offset
	}
}

// generateVT200Sequence generates VT200 mouse sequence
func (mh *MouseHandler) generateVT200Sequence(event MouseEvent) []byte {
	// VT200 mode reports press and release
	if event.Action != MouseActionPress && event.Action != MouseActionRelease {
		return nil
	}

	cb := mh.buttonToVT200Code(event.Button, event.Action)
	if cb == -1 {
		return nil
	}

	// ESC[M + button + x + y (all offset by 32)
	return []byte{
		0x1B, '[', 'M',
		byte(cb + 32),
		byte(event.X + 33),
		byte(event.Y + 33),
	}
}

// generateBtnEventSequence generates button event sequence
func (mh *MouseHandler) generateBtnEventSequence(event MouseEvent) []byte {
	// Button event mode reports press, release, and drag
	if event.Action == MouseActionMove && event.Button == MouseButtonNone {
		return nil // Don't report plain moves without button
	}

	var cb int
	if event.Action == MouseActionRelease {
		// Release always uses code 3
		cb = 3
	} else {
		cb = mh.buttonToBtnEventCode(event.Button, event.Action)
	}
	if cb == -1 {
		return nil
	}

	return []byte{
		0x1B, '[', 'M',
		byte(cb + 32),
		byte(event.X + 33),
		byte(event.Y + 33),
	}
}

// generateAnyEventSequence generates any event sequence
func (mh *MouseHandler) generateAnyEventSequence(event MouseEvent) []byte {
	// Any event mode reports all mouse events including moves
	var cb int

	switch event.Action {
	case MouseActionRelease:
		// Release always uses code 3
		cb = 3
	case MouseActionMove:
		if event.Button == MouseButtonNone {
			// Motion without button uses code 35
			cb = 35
		} else {
			// Motion with button (shouldn't happen, but handle it)
			cb = 32 + int(event.Button)
		}
	case MouseActionDrag:
		// Drag uses button code + 32
		cb = 32 + int(event.Button)
	case MouseActionPress:
		// Press uses button code
		cb = int(event.Button)
	default:
		return nil
	}

	return []byte{
		0x1B, '[', 'M',
		byte(cb + 32),
		byte(event.X + 33),
		byte(event.Y + 33),
	}
}

// buttonToX10Code converts button to X10 code
func (mh *MouseHandler) buttonToX10Code(button MouseButton) int {
	switch button {
	case MouseButtonLeft:
		return 0
	case MouseButtonMiddle:
		return 1
	case MouseButtonRight:
		return 2
	default:
		return -1
	}
}

// buttonToVT200Code converts button and action to VT200 code
func (mh *MouseHandler) buttonToVT200Code(button MouseButton, action MouseAction) int {
	base := mh.buttonToX10Code(button)
	if base == -1 {
		return -1
	}

	if action == MouseActionRelease {
		return 3 // Release code
	}

	return base
}

// buttonToBtnEventCode converts button and action to button event code
func (mh *MouseHandler) buttonToBtnEventCode(button MouseButton, action MouseAction) int {
	switch button {
	case MouseButtonLeft:
		if action == MouseActionDrag {
			return 32
		}
		return 0
	case MouseButtonMiddle:
		if action == MouseActionDrag {
			return 33
		}
		return 1
	case MouseButtonRight:
		if action == MouseActionDrag {
			return 34
		}
		return 2
	case MouseButtonWheelUp:
		return 64
	case MouseButtonWheelDown:
		return 65
	default:
		if action == MouseActionRelease {
			return 3
		}
		return -1
	}
}

// buttonToAnyEventCode converts button and action to any event code
// TODO: Uncomment when mouse support is fully implemented
// func (mh *MouseHandler) buttonToAnyEventCode(button MouseButton, action MouseAction) int {
// 	switch action {
// 	case MouseActionMove:
// 		return 35 // Motion code
// 	case MouseActionPress, MouseActionRelease, MouseActionDrag:
// 		return mh.buttonToBtnEventCode(button, action)
// 	default:
// 		return -1
// 	}
// }

// Add mouse support to TerminalRenderer

// EnableMouseTracking enables mouse tracking in the terminal
func (tr *TerminalRenderer) EnableMouseTracking(mode MouseMode) error {
	tr.mutex.Lock()
	defer tr.mutex.Unlock()

	if !tr.running {
		return fmt.Errorf("renderer is not running")
	}

	// Enable mouse events in tcell
	tr.screen.EnableMouse()

	// Set mouse mode in terminal
	tr.terminal.state.MouseMode = mode

	return nil
}

// DisableMouseTracking disables mouse tracking
func (tr *TerminalRenderer) DisableMouseTracking() error {
	tr.mutex.Lock()
	defer tr.mutex.Unlock()

	if !tr.running {
		return fmt.Errorf("renderer is not running")
	}

	// Disable mouse events in tcell
	tr.screen.DisableMouse()

	// Set mouse mode to off
	tr.terminal.state.MouseMode = MouseModeOff

	return nil
}

// ProcessMouseEvent processes a mouse event and sends appropriate sequences
func (tr *TerminalRenderer) ProcessMouseEvent(event *tcell.EventMouse) error {
	if tr.terminal.state.MouseMode == MouseModeOff {
		return nil
	}

	mouseHandler := NewMouseHandler()
	mouseHandler.SetMode(tr.terminal.state.MouseMode)

	sequence := mouseHandler.ProcessTcellEvent(event)
	if len(sequence) > 0 {
		// Send mouse sequence to serial port as if it was keyboard input
		return tr.terminal.ProcessInput(sequence)
	}

	return nil
}

// Add mouse event handling to the event loop
// TODO: Uncomment when mouse support is fully implemented
// func (tr *TerminalRenderer) handleMouseEvent(event *tcell.EventMouse) {
// 	if err := tr.ProcessMouseEvent(event); err != nil {
// 		// Log error or handle it appropriately
// 		// For now, we'll silently ignore errors
// 	}
// }

// Update the handleEvents method to process mouse events
// TODO: Uncomment when mouse support is fully implemented
// func (tr *TerminalRenderer) handleEventsWithMouse() {
// 	for tr.running {
// 		event := tr.screen.PollEvent()
// 		if event == nil {
// 			continue
// 		}
//
// 		switch ev := event.(type) {
// 		case *tcell.EventMouse:
// 			tr.handleMouseEvent(ev)
// 		default:
// 			// Handle other events
// 			select {
// 			case tr.events <- event:
// 			default:
// 				// Event channel is full, drop event
// 			}
// 		}
// 	}
// }

// KeyEvent represents a keyboard event
type KeyEvent struct {
	Key  tcell.Key
	Char rune
	Mods tcell.ModMask
}

// KeyHandler handles keyboard events and converts them to appropriate sequences
type KeyHandler struct {
	applicationMode bool
	cursorKeyMode   bool
}

// NewKeyHandler creates a new keyboard handler
func NewKeyHandler() *KeyHandler {
	return &KeyHandler{
		applicationMode: false,
		cursorKeyMode:   false,
	}
}

// SetApplicationMode sets the keypad application mode
func (kh *KeyHandler) SetApplicationMode(enabled bool) {
	kh.applicationMode = enabled
}

// SetCursorKeyMode sets the cursor key application mode
func (kh *KeyHandler) SetCursorKeyMode(enabled bool) {
	kh.cursorKeyMode = enabled
}

// ProcessTcellEvent processes a tcell keyboard event and returns the appropriate sequence
func (kh *KeyHandler) ProcessTcellEvent(event *tcell.EventKey) []byte {
	key := event.Key()
	char := event.Rune()
	mods := event.Modifiers()

	// Handle special keys first
	if sequence := kh.handleSpecialKey(key, mods); sequence != nil {
		return sequence
	}

	// Handle function keys
	if sequence := kh.handleFunctionKey(key, mods); sequence != nil {
		return sequence
	}

	// Handle cursor keys
	if sequence := kh.handleCursorKey(key, mods); sequence != nil {
		return sequence
	}

	// Handle keypad keys
	if sequence := kh.handleKeypadKey(key, mods); sequence != nil {
		return sequence
	}

	// Handle control characters
	if sequence := kh.handleControlChar(key, char, mods); sequence != nil {
		return sequence
	}

	// Handle regular characters
	if char != 0 {
		return kh.handleRegularChar(char, mods)
	}

	return nil
}

// handleSpecialKey handles special keys like Enter, Tab, Backspace, etc.
func (kh *KeyHandler) handleSpecialKey(key tcell.Key, mods tcell.ModMask) []byte {
	switch key {
	case tcell.KeyEnter:
		return []byte{0x0D} // CR
	case tcell.KeyTab:
		if mods&tcell.ModShift != 0 {
			return []byte{0x1B, '[', 'Z'} // Shift+Tab (Back Tab)
		}
		return []byte{0x09} // HT
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if mods&tcell.ModAlt != 0 {
			return []byte{0x1B, 0x7F} // Alt+Backspace
		}
		return []byte{0x7F} // DEL
	case tcell.KeyDelete:
		if mods&tcell.ModAlt != 0 {
			return []byte{0x1B, '[', '3', '~'} // Alt+Delete
		}
		return []byte{0x1B, '[', '3', '~'} // Delete
	case tcell.KeyInsert:
		return []byte{0x1B, '[', '2', '~'} // Insert
	case tcell.KeyEscape:
		return []byte{0x1B} // ESC
	}

	return nil
}

// handleFunctionKey handles function keys F1-F12 with full modifier support
func (kh *KeyHandler) handleFunctionKey(key tcell.Key, mods tcell.ModMask) []byte {
	// For F1-F4 with modifiers, use xterm-style sequences
	if mods != 0 && key >= tcell.KeyF1 && key <= tcell.KeyF4 {
		// Calculate modifier value
		modValue := 1
		if mods&tcell.ModShift != 0 {
			modValue += 1
		}
		if mods&tcell.ModAlt != 0 {
			modValue += 2
		}
		if mods&tcell.ModCtrl != 0 {
			modValue += 4
		}

		// F1-F4 with modifiers: ESC[1;modifierP/Q/R/S
		letter := []byte{'P', 'Q', 'R', 'S'}[key-tcell.KeyF1]
		return []byte{0x1B, '[', '1', ';', byte('0' + modValue), letter}
	}

	var base []byte

	switch key {
	case tcell.KeyF1:
		base = []byte{0x1B, 'O', 'P'}
	case tcell.KeyF2:
		base = []byte{0x1B, 'O', 'Q'}
	case tcell.KeyF3:
		base = []byte{0x1B, 'O', 'R'}
	case tcell.KeyF4:
		base = []byte{0x1B, 'O', 'S'}
	case tcell.KeyF5:
		base = []byte{0x1B, '[', '1', '5', '~'}
	case tcell.KeyF6:
		base = []byte{0x1B, '[', '1', '7', '~'}
	case tcell.KeyF7:
		base = []byte{0x1B, '[', '1', '8', '~'}
	case tcell.KeyF8:
		base = []byte{0x1B, '[', '1', '9', '~'}
	case tcell.KeyF9:
		base = []byte{0x1B, '[', '2', '0', '~'}
	case tcell.KeyF10:
		base = []byte{0x1B, '[', '2', '1', '~'}
	case tcell.KeyF11:
		base = []byte{0x1B, '[', '2', '3', '~'}
	case tcell.KeyF12:
		base = []byte{0x1B, '[', '2', '4', '~'}
	default:
		return nil
	}

	// Add modifiers for F5-F12 if present
	if mods != 0 && key >= tcell.KeyF5 {
		return kh.addFunctionKeyModifiers(base, mods)
	}

	return base
}

// addFunctionKeyModifiers adds modifier to function key sequences (F5-F12)
func (kh *KeyHandler) addFunctionKeyModifiers(base []byte, mods tcell.ModMask) []byte {
	// Calculate modifier value
	modValue := 1
	if mods&tcell.ModShift != 0 {
		modValue += 1
	}
	if mods&tcell.ModAlt != 0 {
		modValue += 2
	}
	if mods&tcell.ModCtrl != 0 {
		modValue += 4
	}

	if modValue == 1 {
		return base // No modifiers
	}

	// For sequences like ESC[15~, convert to ESC[15;2~ for modified version
	if len(base) >= 4 && base[0] == 0x1B && base[1] == '[' && base[len(base)-1] == '~' {
		result := make([]byte, 0, len(base)+3)
		result = append(result, base[:len(base)-1]...)        // Everything except ~
		result = append(result, ';', byte('0'+modValue), '~') // Add ;modifier~
		return result
	}

	return base
}

// handleCursorKey handles cursor movement keys
func (kh *KeyHandler) handleCursorKey(key tcell.Key, mods tcell.ModMask) []byte {
	var sequence []byte

	if kh.cursorKeyMode {
		// Application mode
		switch key {
		case tcell.KeyUp:
			sequence = []byte{0x1B, 'O', 'A'}
		case tcell.KeyDown:
			sequence = []byte{0x1B, 'O', 'B'}
		case tcell.KeyRight:
			sequence = []byte{0x1B, 'O', 'C'}
		case tcell.KeyLeft:
			sequence = []byte{0x1B, 'O', 'D'}
		}
	} else {
		// Normal mode
		switch key {
		case tcell.KeyUp:
			sequence = []byte{0x1B, '[', 'A'}
		case tcell.KeyDown:
			sequence = []byte{0x1B, '[', 'B'}
		case tcell.KeyRight:
			sequence = []byte{0x1B, '[', 'C'}
		case tcell.KeyLeft:
			sequence = []byte{0x1B, '[', 'D'}
		}
	}

	if sequence != nil && mods != 0 {
		return kh.addModifiers(sequence, mods)
	}

	return sequence
}

// handleKeypadKey handles numeric keypad keys
func (kh *KeyHandler) handleKeypadKey(key tcell.Key, mods tcell.ModMask) []byte {
	if kh.applicationMode {
		// Application mode sequences
		switch key {
		case tcell.KeyHome:
			return []byte{0x1B, 'O', 'H'}
		case tcell.KeyEnd:
			return []byte{0x1B, 'O', 'F'}
		}
	} else {
		// Normal mode sequences
		switch key {
		case tcell.KeyHome:
			return []byte{0x1B, '[', 'H'}
		case tcell.KeyEnd:
			return []byte{0x1B, '[', 'F'}
		case tcell.KeyPgUp:
			return []byte{0x1B, '[', '5', '~'}
		case tcell.KeyPgDn:
			return []byte{0x1B, '[', '6', '~'}
		}
	}

	return nil
}

// handleControlChar handles control character combinations
func (kh *KeyHandler) handleControlChar(key tcell.Key, char rune, mods tcell.ModMask) []byte {
	// Handle Ctrl+key combinations
	if mods&tcell.ModCtrl != 0 {
		switch {
		case char >= 'a' && char <= 'z':
			// Ctrl+letter
			return []byte{byte(char - 'a' + 1)}
		case char >= 'A' && char <= 'Z':
			// Ctrl+Letter
			return []byte{byte(char - 'A' + 1)}
		case char == ' ':
			// Ctrl+Space
			return []byte{0x00}
		case char == '\\':
			// Ctrl+\
			return []byte{0x1C}
		case char == ']':
			// Ctrl+]
			return []byte{0x1D}
		case char == '^':
			// Ctrl+^
			return []byte{0x1E}
		case char == '_':
			// Ctrl+_
			return []byte{0x1F}
		}
	}

	// Handle Alt+key combinations
	if mods&tcell.ModAlt != 0 && char != 0 {
		// Alt+char sends ESC followed by char
		return []byte{0x1B, byte(char)}
	}

	return nil
}

// handleRegularChar handles regular printable characters
func (kh *KeyHandler) handleRegularChar(char rune, mods tcell.ModMask) []byte {
	// Handle Alt modifier
	if mods&tcell.ModAlt != 0 {
		// Alt+char sends ESC followed by char
		return []byte{0x1B, byte(char)}
	}

	// Regular character
	if char <= 0x7F {
		return []byte{byte(char)}
	}

	// UTF-8 character
	return []byte(string(char))
}

// addModifiers adds modifier information to escape sequences
func (kh *KeyHandler) addModifiers(base []byte, mods tcell.ModMask) []byte {
	// Calculate modifier parameter
	modParam := 1
	if mods&tcell.ModShift != 0 {
		modParam += 1
	}
	if mods&tcell.ModAlt != 0 {
		modParam += 2
	}
	if mods&tcell.ModCtrl != 0 {
		modParam += 4
	}

	if modParam == 1 {
		return base // No modifiers
	}

	// Insert modifier parameter into sequence
	// For sequences like ESC[A, convert to ESC[1;2A for Shift+Up
	if len(base) >= 3 && base[0] == 0x1B && base[1] == '[' {
		result := make([]byte, 0, len(base)+4)
		result = append(result, base[:2]...)        // ESC[
		result = append(result, '1', ';')           // 1;
		result = append(result, byte('0'+modParam)) // modifier
		result = append(result, base[2:]...)        // rest of sequence
		return result
	}

	return base
}

// InputProcessor processes all input events and coordinates with terminal
type InputProcessor struct {
	keyHandler   *KeyHandler
	mouseHandler *MouseHandler
	terminal     *TerminalEmulator
}

// NewInputProcessor creates a new input processor
func NewInputProcessor(terminal *TerminalEmulator) *InputProcessor {
	return &InputProcessor{
		keyHandler:   NewKeyHandler(),
		mouseHandler: NewMouseHandler(),
		terminal:     terminal,
	}
}

// ProcessEvent processes any tcell event and sends appropriate data to terminal
func (ip *InputProcessor) ProcessEvent(event tcell.Event) error {
	switch ev := event.(type) {
	case *tcell.EventKey:
		return ip.processKeyEvent(ev)
	case *tcell.EventMouse:
		return ip.processMouseEvent(ev)
	case *tcell.EventResize:
		return ip.processResizeEvent(ev)
	}

	return nil
}

// processKeyEvent processes keyboard events
func (ip *InputProcessor) processKeyEvent(event *tcell.EventKey) error {
	sequence := ip.keyHandler.ProcessTcellEvent(event)
	if len(sequence) > 0 {
		return ip.terminal.ProcessInput(sequence)
	}
	return nil
}

// ProcessKeyEvent processes keyboard events and returns the data to send
func (ip *InputProcessor) ProcessKeyEvent(event *tcell.EventKey) []byte {
	return ip.keyHandler.ProcessTcellEvent(event)
}

// processMouseEvent processes mouse events
func (ip *InputProcessor) processMouseEvent(event *tcell.EventMouse) error {
	sequence := ip.mouseHandler.ProcessTcellEvent(event)
	if len(sequence) > 0 {
		return ip.terminal.ProcessInput(sequence)
	}
	return nil
}

// ProcessMouseEvent processes mouse events and returns the data to send
func (ip *InputProcessor) ProcessMouseEvent(event *tcell.EventMouse) []byte {
	// Set the mouse mode from terminal state before processing
	if ip.terminal != nil {
		currentMode := ip.terminal.GetState().MouseMode
		ip.mouseHandler.SetMode(currentMode)
	}
	return ip.mouseHandler.ProcessTcellEvent(event)
}

// processResizeEvent processes terminal resize events
func (ip *InputProcessor) processResizeEvent(event *tcell.EventResize) error {
	width, height := event.Size()
	return ip.terminal.Resize(width, height)
}

// SetKeypadApplicationMode sets keypad application mode
func (ip *InputProcessor) SetKeypadApplicationMode(enabled bool) {
	ip.keyHandler.SetApplicationMode(enabled)
}

// SetCursorKeyApplicationMode sets cursor key application mode
func (ip *InputProcessor) SetCursorKeyApplicationMode(enabled bool) {
	ip.keyHandler.SetCursorKeyMode(enabled)
}

// SetMouseMode sets mouse tracking mode
func (ip *InputProcessor) SetMouseMode(mode MouseMode) {
	ip.mouseHandler.SetMode(mode)
}

// GetKeyHandler returns the key handler
func (ip *InputProcessor) GetKeyHandler() *KeyHandler {
	return ip.keyHandler
}

// GetMouseHandler returns the mouse handler
func (ip *InputProcessor) GetMouseHandler() *MouseHandler {
	return ip.mouseHandler
}

// Update TerminalRenderer to use InputProcessor
func (tr *TerminalRenderer) SetInputProcessor(processor *InputProcessor) {
	// This would be used to coordinate input processing with rendering
	// Implementation would depend on the specific architecture
}

// KeySequence represents a key sequence mapping
type KeySequence struct {
	Name     string
	Sequence []byte
	Mods     tcell.ModMask
}

// Common key sequences for reference
var CommonKeySequences = []KeySequence{
	{"Ctrl+C", []byte{0x03}, tcell.ModCtrl},
	{"Ctrl+D", []byte{0x04}, tcell.ModCtrl},
	{"Ctrl+Z", []byte{0x1A}, tcell.ModCtrl},
	{"Ctrl+L", []byte{0x0C}, tcell.ModCtrl},
	{"Alt+Enter", []byte{0x1B, 0x0D}, tcell.ModAlt},
	{"Shift+Tab", []byte{0x1B, '[', 'Z'}, tcell.ModShift},
}

// GetKeySequenceByName returns the sequence for a named key combination
func GetKeySequenceByName(name string) []byte {
	for _, seq := range CommonKeySequences {
		if seq.Name == name {
			return seq.Sequence
		}
	}
	return nil
}

// ShortcutAction represents different types of shortcut actions
type ShortcutAction int

const (
	ActionExit ShortcutAction = iota
	ActionSave
	ActionClear
	ActionCopy
	ActionPaste
	ActionFind
	ActionHelp
	ActionSettings
	ActionConnect
	ActionDisconnect
	ActionCustom
)

// String returns the string representation of ShortcutAction
func (sa ShortcutAction) String() string {
	switch sa {
	case ActionExit:
		return "exit"
	case ActionSave:
		return "save"
	case ActionClear:
		return "clear"
	case ActionCopy:
		return "copy"
	case ActionPaste:
		return "paste"
	case ActionFind:
		return "find"
	case ActionHelp:
		return "help"
	case ActionSettings:
		return "settings"
	case ActionConnect:
		return "connect"
	case ActionDisconnect:
		return "disconnect"
	case ActionCustom:
		return "custom"
	default:
		return "unknown"
	}
}

// Shortcut represents a keyboard shortcut
type Shortcut struct {
	Name        string
	Key         tcell.Key
	Char        rune
	Mods        tcell.ModMask
	Action      ShortcutAction
	Handler     func() error
	Description string
	Enabled     bool
}

// Matches checks if the given key event matches this shortcut
func (s *Shortcut) Matches(key tcell.Key, char rune, mods tcell.ModMask) bool {
	if !s.Enabled {
		return false
	}

	// Check modifiers first
	if s.Mods != mods {
		return false
	}

	// Check key or character
	if s.Key != tcell.KeyRune {
		// For non-rune keys, only compare the key
		return s.Key == key
	}

	// For rune keys, compare both key and character
	return key == tcell.KeyRune && s.Char == char
}

// Execute executes the shortcut action
func (s *Shortcut) Execute() error {
	if !s.Enabled {
		return fmt.Errorf("shortcut %s is disabled", s.Name)
	}

	if s.Handler != nil {
		return s.Handler()
	}

	return fmt.Errorf("no handler defined for shortcut %s", s.Name)
}

// ShortcutManager manages keyboard shortcuts
type ShortcutManager struct {
	shortcuts map[string]*Shortcut
	enabled   bool
}

// NewShortcutManager creates a new shortcut manager
func NewShortcutManager() *ShortcutManager {
	sm := &ShortcutManager{
		shortcuts: make(map[string]*Shortcut),
		enabled:   true,
	}

	// Add default shortcuts
	sm.addDefaultShortcuts()

	return sm
}

// addDefaultShortcuts adds the default application shortcuts
func (sm *ShortcutManager) addDefaultShortcuts() {
	// Note: Using Ctrl+Shift+Q for exit to avoid conflicts with terminal programs
	sm.AddShortcut(&Shortcut{
		Name:        "exit",
		Key:         tcell.KeyRune,
		Char:        'Q',
		Mods:        tcell.ModCtrl | tcell.ModShift,
		Action:      ActionExit,
		Description: "Exit application",
		Enabled:     true,
	})

	sm.AddShortcut(&Shortcut{
		Name:        "save",
		Key:         tcell.KeyRune,
		Char:        'S',
		Mods:        tcell.ModCtrl | tcell.ModShift,
		Action:      ActionSave,
		Description: "Save history to file",
		Enabled:     true,
	})

	sm.AddShortcut(&Shortcut{
		Name:        "clear",
		Key:         tcell.KeyRune,
		Char:        'L',
		Mods:        tcell.ModCtrl | tcell.ModShift,
		Action:      ActionClear,
		Description: "Clear terminal screen",
		Enabled:     true,
	})

	sm.AddShortcut(&Shortcut{
		Name:        "help",
		Key:         tcell.KeyF1,
		Char:        0,
		Mods:        0,
		Action:      ActionHelp,
		Description: "Show help",
		Enabled:     true,
	})

	sm.AddShortcut(&Shortcut{
		Name:        "settings",
		Key:         tcell.KeyRune,
		Char:        ',',
		Mods:        tcell.ModCtrl,
		Action:      ActionSettings,
		Description: "Open settings",
		Enabled:     true,
	})

	sm.AddShortcut(&Shortcut{
		Name:        "connect",
		Key:         tcell.KeyRune,
		Char:        'O',
		Mods:        tcell.ModCtrl | tcell.ModShift,
		Action:      ActionConnect,
		Description: "Connect to serial port",
		Enabled:     true,
	})

	sm.AddShortcut(&Shortcut{
		Name:        "disconnect",
		Key:         tcell.KeyRune,
		Char:        'D',
		Mods:        tcell.ModCtrl | tcell.ModShift,
		Action:      ActionDisconnect,
		Description: "Disconnect from serial port",
		Enabled:     true,
	})
}

// AddShortcut adds a new shortcut
func (sm *ShortcutManager) AddShortcut(shortcut *Shortcut) {
	sm.shortcuts[shortcut.Name] = shortcut
}

// RemoveShortcut removes a shortcut by name
func (sm *ShortcutManager) RemoveShortcut(name string) {
	delete(sm.shortcuts, name)
}

// GetShortcut returns a shortcut by name
func (sm *ShortcutManager) GetShortcut(name string) *Shortcut {
	return sm.shortcuts[name]
}

// ListShortcuts returns all shortcuts
func (sm *ShortcutManager) ListShortcuts() []*Shortcut {
	shortcuts := make([]*Shortcut, 0, len(sm.shortcuts))
	for _, shortcut := range sm.shortcuts {
		shortcuts = append(shortcuts, shortcut)
	}
	return shortcuts
}

// EnableShortcut enables a shortcut by name
func (sm *ShortcutManager) EnableShortcut(name string) {
	if shortcut := sm.shortcuts[name]; shortcut != nil {
		shortcut.Enabled = true
	}
}

// DisableShortcut disables a shortcut by name
func (sm *ShortcutManager) DisableShortcut(name string) {
	if shortcut := sm.shortcuts[name]; shortcut != nil {
		shortcut.Enabled = false
	}
}

// SetEnabled enables or disables the entire shortcut system
func (sm *ShortcutManager) SetEnabled(enabled bool) {
	sm.enabled = enabled
}

// IsEnabled returns whether the shortcut system is enabled
func (sm *ShortcutManager) IsEnabled() bool {
	return sm.enabled
}

// ProcessKeyEvent processes a key event and executes matching shortcuts
func (sm *ShortcutManager) ProcessKeyEvent(key tcell.Key, char rune, mods tcell.ModMask) (bool, error) {
	if !sm.enabled {
		return false, nil
	}

	for _, shortcut := range sm.shortcuts {
		if shortcut.Matches(key, char, mods) {
			err := shortcut.Execute()
			return true, err // Return true to indicate shortcut was handled
		}
	}

	return false, nil // No shortcut matched
}

// SetShortcutHandler sets the handler for a specific shortcut
func (sm *ShortcutManager) SetShortcutHandler(name string, handler func() error) error {
	shortcut := sm.shortcuts[name]
	if shortcut == nil {
		return fmt.Errorf("shortcut %s not found", name)
	}

	shortcut.Handler = handler
	return nil
}

// CustomShortcut creates a custom shortcut
func (sm *ShortcutManager) CustomShortcut(name, description string, key tcell.Key, char rune, mods tcell.ModMask, handler func() error) {
	shortcut := &Shortcut{
		Name:        name,
		Key:         key,
		Char:        char,
		Mods:        mods,
		Action:      ActionCustom,
		Handler:     handler,
		Description: description,
		Enabled:     true,
	}

	sm.AddShortcut(shortcut)
}

// GetShortcutHelp returns help text for all shortcuts
func (sm *ShortcutManager) GetShortcutHelp() string {
	help := "Available Shortcuts:\n\n"

	for _, shortcut := range sm.shortcuts {
		if !shortcut.Enabled {
			continue
		}

		keyDesc := sm.formatKeyDescription(shortcut)
		help += fmt.Sprintf("  %-20s %s\n", keyDesc, shortcut.Description)
	}

	return help
}

// formatKeyDescription formats a key combination for display
func (sm *ShortcutManager) formatKeyDescription(shortcut *Shortcut) string {
	var parts []string

	if shortcut.Mods&tcell.ModCtrl != 0 {
		parts = append(parts, "Ctrl")
	}
	if shortcut.Mods&tcell.ModAlt != 0 {
		parts = append(parts, "Alt")
	}
	if shortcut.Mods&tcell.ModShift != 0 {
		parts = append(parts, "Shift")
	}

	var keyName string
	if shortcut.Key == tcell.KeyRune {
		keyName = string(shortcut.Char)
	} else {
		keyName = sm.keyToString(shortcut.Key)
	}

	if len(parts) > 0 {
		return fmt.Sprintf("%s+%s", joinStrings(parts, "+"), keyName)
	}

	return keyName
}

// keyToString converts tcell.Key to string
func (sm *ShortcutManager) keyToString(key tcell.Key) string {
	switch key {
	case tcell.KeyF1:
		return "F1"
	case tcell.KeyF2:
		return "F2"
	case tcell.KeyF3:
		return "F3"
	case tcell.KeyF4:
		return "F4"
	case tcell.KeyF5:
		return "F5"
	case tcell.KeyF6:
		return "F6"
	case tcell.KeyF7:
		return "F7"
	case tcell.KeyF8:
		return "F8"
	case tcell.KeyF9:
		return "F9"
	case tcell.KeyF10:
		return "F10"
	case tcell.KeyF11:
		return "F11"
	case tcell.KeyF12:
		return "F12"
	case tcell.KeyEnter:
		return "Enter"
	case tcell.KeyTab:
		return "Tab"
	case tcell.KeyBackspace:
		return "Backspace"
	case tcell.KeyDelete:
		return "Delete"
	case tcell.KeyInsert:
		return "Insert"
	case tcell.KeyHome:
		return "Home"
	case tcell.KeyEnd:
		return "End"
	case tcell.KeyPgUp:
		return "PgUp"
	case tcell.KeyPgDn:
		return "PgDn"
	case tcell.KeyUp:
		return "Up"
	case tcell.KeyDown:
		return "Down"
	case tcell.KeyLeft:
		return "Left"
	case tcell.KeyRight:
		return "Right"
	case tcell.KeyEscape:
		return "Esc"
	default:
		return "Unknown"
	}
}

// joinStrings joins strings with a separator
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}

	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}

	return result
}

// ShortcutConfig represents shortcut configuration for persistence
type ShortcutConfig struct {
	Name        string   `json:"name"`
	Key         string   `json:"key"`
	Char        string   `json:"char"`
	Mods        []string `json:"mods"`
	Action      string   `json:"action"`
	Description string   `json:"description"`
	Enabled     bool     `json:"enabled"`
}

// ToConfig converts a Shortcut to ShortcutConfig for serialization
func (s *Shortcut) ToConfig() ShortcutConfig {
	config := ShortcutConfig{
		Name:        s.Name,
		Key:         "",
		Char:        "",
		Mods:        []string{},
		Action:      s.Action.String(),
		Description: s.Description,
		Enabled:     s.Enabled,
	}

	// Convert key
	if s.Key == tcell.KeyRune {
		config.Char = string(s.Char)
	} else {
		sm := NewShortcutManager()
		config.Key = sm.keyToString(s.Key)
	}

	// Convert modifiers
	if s.Mods&tcell.ModCtrl != 0 {
		config.Mods = append(config.Mods, "ctrl")
	}
	if s.Mods&tcell.ModAlt != 0 {
		config.Mods = append(config.Mods, "alt")
	}
	if s.Mods&tcell.ModShift != 0 {
		config.Mods = append(config.Mods, "shift")
	}

	return config
}

// FromConfig creates a Shortcut from ShortcutConfig
func ShortcutFromConfig(config ShortcutConfig) (*Shortcut, error) {
	shortcut := &Shortcut{
		Name:        config.Name,
		Description: config.Description,
		Enabled:     config.Enabled,
	}

	// Parse action
	switch config.Action {
	case "exit":
		shortcut.Action = ActionExit
	case "save":
		shortcut.Action = ActionSave
	case "clear":
		shortcut.Action = ActionClear
	case "copy":
		shortcut.Action = ActionCopy
	case "paste":
		shortcut.Action = ActionPaste
	case "find":
		shortcut.Action = ActionFind
	case "help":
		shortcut.Action = ActionHelp
	case "settings":
		shortcut.Action = ActionSettings
	case "connect":
		shortcut.Action = ActionConnect
	case "disconnect":
		shortcut.Action = ActionDisconnect
	case "custom":
		shortcut.Action = ActionCustom
	default:
		return nil, fmt.Errorf("unknown action: %s", config.Action)
	}

	// Parse modifiers
	var mods tcell.ModMask
	for _, mod := range config.Mods {
		switch mod {
		case "ctrl":
			mods |= tcell.ModCtrl
		case "alt":
			mods |= tcell.ModAlt
		case "shift":
			mods |= tcell.ModShift
		}
	}
	shortcut.Mods = mods

	// Parse key or character
	if config.Char != "" {
		shortcut.Key = tcell.KeyRune
		if len(config.Char) > 0 {
			shortcut.Char = rune(config.Char[0])
		}
	} else {
		key, err := stringToKey(config.Key)
		if err != nil {
			return nil, fmt.Errorf("invalid key: %s", config.Key)
		}
		shortcut.Key = key
	}

	return shortcut, nil
}

// stringToKey converts string to tcell.Key
func stringToKey(keyStr string) (tcell.Key, error) {
	switch keyStr {
	case "F1":
		return tcell.KeyF1, nil
	case "F2":
		return tcell.KeyF2, nil
	case "F3":
		return tcell.KeyF3, nil
	case "F4":
		return tcell.KeyF4, nil
	case "F5":
		return tcell.KeyF5, nil
	case "F6":
		return tcell.KeyF6, nil
	case "F7":
		return tcell.KeyF7, nil
	case "F8":
		return tcell.KeyF8, nil
	case "F9":
		return tcell.KeyF9, nil
	case "F10":
		return tcell.KeyF10, nil
	case "F11":
		return tcell.KeyF11, nil
	case "F12":
		return tcell.KeyF12, nil
	case "Enter":
		return tcell.KeyEnter, nil
	case "Tab":
		return tcell.KeyTab, nil
	case "Backspace":
		return tcell.KeyBackspace, nil
	case "Delete":
		return tcell.KeyDelete, nil
	case "Insert":
		return tcell.KeyInsert, nil
	case "Home":
		return tcell.KeyHome, nil
	case "End":
		return tcell.KeyEnd, nil
	case "PgUp":
		return tcell.KeyPgUp, nil
	case "PgDn":
		return tcell.KeyPgDn, nil
	case "Up":
		return tcell.KeyUp, nil
	case "Down":
		return tcell.KeyDown, nil
	case "Left":
		return tcell.KeyLeft, nil
	case "Right":
		return tcell.KeyRight, nil
	case "Esc":
		return tcell.KeyEscape, nil
	default:
		return tcell.KeyRune, fmt.Errorf("unknown key: %s", keyStr)
	}
}

// Update InputProcessor to handle shortcuts
func (ip *InputProcessor) SetShortcutManager(sm *ShortcutManager) {
	// This would integrate shortcut handling into the input processor
	// The actual implementation would depend on the application architecture
}

// Enhanced processKeyEvent with shortcut handling
// TODO: Uncomment when shortcut integration is needed
// func (ip *InputProcessor) processKeyEventWithShortcuts(event *tcell.EventKey, sm *ShortcutManager) error {
// 	// First check for shortcuts
// 	if sm != nil {
// 		handled, err := sm.ProcessKeyEvent(event.Key(), event.Rune(), event.Modifiers())
// 		if handled {
// 			return err // Shortcut was handled, don't process as regular input
// 		}
// 	}
//
// 	// Process as regular key input
// 	return ip.processKeyEvent(event)
// }
