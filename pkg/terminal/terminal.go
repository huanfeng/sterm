// Package terminal provides terminal emulation functionality
package terminal

import (
	"fmt"
	"serial-terminal/pkg/history"
	"serial-terminal/pkg/serial"
	"sync"

	"github.com/gdamore/tcell/v2"
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

// TerminalEmulator implements the Terminal interface
type TerminalEmulator struct {
	screen         *Screen
	parser         *VTParser
	serialPort     serial.SerialPort
	historyManager history.HistoryManager
	state          TerminalState
	isRunning      bool
}

// NewTerminalEmulator creates a new terminal emulator
func NewTerminalEmulator(serialPort serial.SerialPort, historyManager history.HistoryManager, width, height int) *TerminalEmulator {
	return &TerminalEmulator{
		screen:         NewScreen(width, height),
		parser:         NewVTParser(),
		serialPort:     serialPort,
		historyManager: historyManager,
		state:          DefaultTerminalState(width, height),
		isRunning:      false,
	}
}

// Screen represents the terminal screen buffer
type Screen struct {
	Width  int
	Height int
	Buffer [][]Cell
	Dirty  bool
}

// Cell represents a single character cell in the terminal
type Cell struct {
	Char       rune           `json:"char"`
	Attributes TextAttributes `json:"attributes"`
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
			}
		}
	}

	return &Screen{
		Width:  width,
		Height: height,
		Buffer: buffer,
		Dirty:  true,
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
func (vt *VTParser) ParseByte(b byte, screen *Screen, state *TerminalState) []Action {
	var actions []Action

	switch vt.State {
	case StateGround:
		actions = vt.handleGround(b, screen, state)
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
)

// handleGround processes characters in ground state
func (vt *VTParser) handleGround(b byte, screen *Screen, state *TerminalState) []Action {
	switch b {
	case 0x1B: // ESC
		vt.State = StateEscape
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
		// Ignore other control characters
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
		// TODO: Implement tab stops
		return nil
	case '7': // DECSC - Save Cursor
		vt.Reset()
		// TODO: Implement cursor save
		return nil
	case '8': // DECRC - Restore Cursor
		vt.Reset()
		// TODO: Implement cursor restore
		return nil
	case '=': // DECKPAM - Keypad Application Mode
		vt.Reset()
		return []Action{{Type: ActionSetMode, Data: "keypad_app"}}
	case '>': // DECKPNM - Keypad Numeric Mode
		vt.Reset()
		return []Action{{Type: ActionSetMode, Data: "keypad_num"}}
	default:
		vt.Reset()
		return nil
	}
}

// handleCSI processes Control Sequence Introducer sequences
func (vt *VTParser) handleCSI(b byte, screen *Screen, state *TerminalState) []Action {
	if b >= 0x30 && b <= 0x3F { // Parameter bytes
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
		// TODO: Implement cursor save
		return nil
	case 'u': // SCORC - Restore Cursor Position
		// TODO: Implement cursor restore
		return nil
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

	for _, param := range vt.Params {
		var mode string
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
		case 25: // DECTCEM - Text Cursor Enable Mode
			if set {
				mode = "cursor_visible"
			} else {
				mode = "cursor_hidden"
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
		case 1049: // Alternative screen buffer
			if set {
				mode = "alt_screen"
			} else {
				mode = "normal_screen"
			}
		default:
			continue
		}

		actions = append(actions, Action{Type: ActionSetMode, Data: mode})
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
			style := tr.attributesToStyle(cell.Attributes)
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
			te.historyManager.Write(input, history.DirectionOutput)
		}
	}

	return nil
}

// ProcessOutput processes output from the serial port
func (te *TerminalEmulator) ProcessOutput(output []byte) error {
	if !te.isRunning {
		return fmt.Errorf("terminal is not running")
	}

	// Log output to history
	if te.historyManager != nil {
		te.historyManager.Write(output, history.DirectionInput)
	}

	// Process each byte through the VT parser
	for _, b := range output {
		actions := te.parser.ParseByte(b, te.screen, &te.state)

		// Execute actions
		for _, action := range actions {
			te.executeAction(action)
		}
	}

	return nil
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
	case ActionTab:
		te.tab()
	case ActionNewline:
		te.newline()
	case ActionCarriageReturn:
		te.carriageReturn()
	case ActionBackspace:
		te.backspace()
	case ActionDeleteChar:
		te.deleteChar(action.Data.(int))
	case ActionInsertChar:
		te.insertChar(action.Data.(int))
	case ActionSetScrollRegion:
		te.setScrollRegion(action.Data.(ScrollRegion))
	}
}

// printChar prints a character at the current cursor position
func (te *TerminalEmulator) printChar(ch rune) {
	if te.state.CursorX >= te.state.Width {
		te.newline()
		te.carriageReturn()
	}

	if te.state.CursorY >= te.state.Height {
		te.scroll("up")
		te.state.CursorY = te.state.Height - 1
	}

	// Set character in screen buffer
	te.screen.Buffer[te.state.CursorY][te.state.CursorX] = Cell{
		Char:       ch,
		Attributes: te.state.Attributes,
	}

	te.state.CursorX++
	te.screen.Dirty = true
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
	case "absolute":
		te.state.CursorX = max(0, min(te.state.Width-1, move.Col))
		te.state.CursorY = max(0, min(te.state.Height-1, move.Row))
	}
}

// clearScreen clears the screen
func (te *TerminalEmulator) clearScreen(mode int) {
	switch mode {
	case 0: // Clear from cursor to end of screen
		te.clearFromCursor()
	case 1: // Clear from beginning of screen to cursor
		te.clearToCursor()
	case 2: // Clear entire screen
		te.clearEntireScreen()
	}
	te.screen.Dirty = true
}

// clearLine clears the current line
func (te *TerminalEmulator) clearLine(mode int) {
	y := te.state.CursorY

	switch mode {
	case 0: // Clear from cursor to end of line
		for x := te.state.CursorX; x < te.state.Width; x++ {
			te.screen.Buffer[y][x] = Cell{Char: ' ', Attributes: DefaultTextAttributes()}
		}
	case 1: // Clear from beginning of line to cursor
		for x := 0; x <= te.state.CursorX; x++ {
			te.screen.Buffer[y][x] = Cell{Char: ' ', Attributes: DefaultTextAttributes()}
		}
	case 2: // Clear entire line
		for x := 0; x < te.state.Width; x++ {
			te.screen.Buffer[y][x] = Cell{Char: ' ', Attributes: DefaultTextAttributes()}
		}
	}
	te.screen.Dirty = true
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
	te.screen.Dirty = true
}

// scrollUp scrolls the screen up by one line
func (te *TerminalEmulator) scrollUp() {
	// Move all lines up
	for y := te.state.ScrollTop; y < te.state.ScrollBottom; y++ {
		copy(te.screen.Buffer[y], te.screen.Buffer[y+1])
	}

	// Clear bottom line
	for x := 0; x < te.state.Width; x++ {
		te.screen.Buffer[te.state.ScrollBottom][x] = Cell{Char: ' ', Attributes: DefaultTextAttributes()}
	}
}

// scrollDown scrolls the screen down by one line
func (te *TerminalEmulator) scrollDown() {
	// Move all lines down
	for y := te.state.ScrollBottom; y > te.state.ScrollTop; y-- {
		copy(te.screen.Buffer[y], te.screen.Buffer[y-1])
	}

	// Clear top line
	for x := 0; x < te.state.Width; x++ {
		te.screen.Buffer[te.state.ScrollTop][x] = Cell{Char: ' ', Attributes: DefaultTextAttributes()}
	}
}

// setMode sets terminal mode
func (te *TerminalEmulator) setMode(mode string) {
	switch mode {
	case "cursor_visible":
		// TODO: Implement cursor visibility
	case "cursor_hidden":
		// TODO: Implement cursor visibility
	case "mouse_x10":
		te.state.MouseMode = MouseModeX10
	case "mouse_btn_event":
		te.state.MouseMode = MouseModeBtnEvent
	case "mouse_any_event":
		te.state.MouseMode = MouseModeAnyEvent
	case "mouse_off":
		te.state.MouseMode = MouseModeOff
	}
}

// tab moves cursor to next tab stop
func (te *TerminalEmulator) tab() {
	te.state.CursorX = ((te.state.CursorX / 8) + 1) * 8
	if te.state.CursorX >= te.state.Width {
		te.state.CursorX = te.state.Width - 1
	}
}

// newline moves cursor to next line
func (te *TerminalEmulator) newline() {
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
		te.state.CursorX--
	}
}

// deleteChar deletes characters at cursor position
func (te *TerminalEmulator) deleteChar(count int) {
	y := te.state.CursorY
	x := te.state.CursorX

	// Shift characters left
	for i := x; i < te.state.Width-count; i++ {
		te.screen.Buffer[y][i] = te.screen.Buffer[y][i+count]
	}

	// Clear rightmost characters
	for i := te.state.Width - count; i < te.state.Width; i++ {
		te.screen.Buffer[y][i] = Cell{Char: ' ', Attributes: DefaultTextAttributes()}
	}

	te.screen.Dirty = true
}

// insertChar inserts blank characters at cursor position
func (te *TerminalEmulator) insertChar(count int) {
	y := te.state.CursorY
	x := te.state.CursorX

	// Shift characters right
	for i := te.state.Width - 1; i >= x+count; i-- {
		te.screen.Buffer[y][i] = te.screen.Buffer[y][i-count]
	}

	// Clear inserted characters
	for i := x; i < x+count && i < te.state.Width; i++ {
		te.screen.Buffer[y][i] = Cell{Char: ' ', Attributes: DefaultTextAttributes()}
	}

	te.screen.Dirty = true
}

// setScrollRegion sets the scroll region
func (te *TerminalEmulator) setScrollRegion(region ScrollRegion) {
	te.state.ScrollTop = max(0, min(te.state.Height-1, region.Top))
	te.state.ScrollBottom = max(te.state.ScrollTop, min(te.state.Height-1, region.Bottom))
}

// clearFromCursor clears from cursor to end of screen
func (te *TerminalEmulator) clearFromCursor() {
	// Clear from cursor to end of current line
	for x := te.state.CursorX; x < te.state.Width; x++ {
		te.screen.Buffer[te.state.CursorY][x] = Cell{Char: ' ', Attributes: DefaultTextAttributes()}
	}

	// Clear all lines below current line
	for y := te.state.CursorY + 1; y < te.state.Height; y++ {
		for x := 0; x < te.state.Width; x++ {
			te.screen.Buffer[y][x] = Cell{Char: ' ', Attributes: DefaultTextAttributes()}
		}
	}
}

// clearToCursor clears from beginning of screen to cursor
func (te *TerminalEmulator) clearToCursor() {
	// Clear all lines above current line
	for y := 0; y < te.state.CursorY; y++ {
		for x := 0; x < te.state.Width; x++ {
			te.screen.Buffer[y][x] = Cell{Char: ' ', Attributes: DefaultTextAttributes()}
		}
	}

	// Clear from beginning of current line to cursor
	for x := 0; x <= te.state.CursorX; x++ {
		te.screen.Buffer[te.state.CursorY][x] = Cell{Char: ' ', Attributes: DefaultTextAttributes()}
	}
}

// clearEntireScreen clears the entire screen
func (te *TerminalEmulator) clearEntireScreen() {
	for y := 0; y < te.state.Height; y++ {
		for x := 0; x < te.state.Width; x++ {
			te.screen.Buffer[y][x] = Cell{Char: ' ', Attributes: DefaultTextAttributes()}
		}
	}
}

// Resize resizes the terminal
func (te *TerminalEmulator) Resize(width, height int) error {
	if width <= 0 || height <= 0 {
		return fmt.Errorf("invalid dimensions: %dx%d", width, height)
	}

	// Create new screen buffer
	newScreen := NewScreen(width, height)

	// Copy existing content
	copyHeight := min(height, te.screen.Height)
	copyWidth := min(width, te.screen.Width)

	for y := 0; y < copyHeight; y++ {
		for x := 0; x < copyWidth; x++ {
			newScreen.Buffer[y][x] = te.screen.Buffer[y][x]
		}
	}

	// Update terminal state
	te.screen = newScreen
	te.state.Width = width
	te.state.Height = height

	// Adjust cursor position if necessary
	te.state.CursorX = min(te.state.CursorX, width-1)
	te.state.CursorY = min(te.state.CursorY, height-1)

	// Adjust scroll region
	te.state.ScrollBottom = min(te.state.ScrollBottom, height-1)
	te.state.ScrollTop = min(te.state.ScrollTop, te.state.ScrollBottom)

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
	return te.state
}

// GetScreen returns the terminal screen buffer
func (te *TerminalEmulator) GetScreen() *Screen {
	return te.screen
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

	// Determine button
	switch {
	case buttons&tcell.Button1 != 0:
		button = MouseButtonLeft
	case buttons&tcell.Button2 != 0:
		button = MouseButtonMiddle
	case buttons&tcell.Button3 != 0:
		button = MouseButtonRight
	case buttons&tcell.WheelUp != 0:
		button = MouseButtonWheelUp
	case buttons&tcell.WheelDown != 0:
		button = MouseButtonWheelDown
	default:
		button = MouseButtonNone
	}

	// Determine action
	if button == MouseButtonNone {
		// Mouse move without button
		if mh.dragButton != MouseButtonNone {
			action = MouseActionRelease
			button = mh.dragButton
			mh.dragButton = MouseButtonNone
		} else {
			action = MouseActionMove
		}
	} else {
		// Button event
		wasPressed := mh.buttonState[button]
		isPressed := buttons != 0

		if isPressed && !wasPressed {
			action = MouseActionPress
			if button != MouseButtonWheelUp && button != MouseButtonWheelDown {
				mh.dragButton = button
			}
		} else if !isPressed && wasPressed {
			action = MouseActionRelease
			mh.dragButton = MouseButtonNone
		} else if isPressed && (x != mh.lastX || y != mh.lastY) {
			action = MouseActionDrag
		} else {
			action = MouseActionMove
		}

		mh.buttonState[button] = isPressed
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
		return nil // Don't report plain moves
	}

	cb := mh.buttonToBtnEventCode(event.Button, event.Action)
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
	// Any event mode reports all mouse events
	cb := mh.buttonToAnyEventCode(event.Button, event.Action)
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
func (mh *MouseHandler) buttonToAnyEventCode(button MouseButton, action MouseAction) int {
	switch action {
	case MouseActionMove:
		return 35 // Motion code
	case MouseActionPress, MouseActionRelease, MouseActionDrag:
		return mh.buttonToBtnEventCode(button, action)
	default:
		return -1
	}
}

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
func (tr *TerminalRenderer) handleMouseEvent(event *tcell.EventMouse) {
	if err := tr.ProcessMouseEvent(event); err != nil {
		// Log error or handle it appropriately
		// For now, we'll silently ignore errors
	}
}

// Update the handleEvents method to process mouse events
func (tr *TerminalRenderer) handleEventsWithMouse() {
	for tr.running {
		event := tr.screen.PollEvent()
		if event == nil {
			continue
		}

		switch ev := event.(type) {
		case *tcell.EventMouse:
			tr.handleMouseEvent(ev)
		default:
			// Handle other events
			select {
			case tr.events <- event:
			default:
				// Event channel is full, drop event
			}
		}
	}
}

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

// handleFunctionKey handles function keys F1-F12
func (kh *KeyHandler) handleFunctionKey(key tcell.Key, mods tcell.ModMask) []byte {
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

	// Add modifiers if present
	if mods != 0 {
		return kh.addModifiers(base, mods)
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
func (ip *InputProcessor) processKeyEventWithShortcuts(event *tcell.EventKey, sm *ShortcutManager) error {
	// First check for shortcuts
	if sm != nil {
		handled, err := sm.ProcessKeyEvent(event.Key(), event.Rune(), event.Modifiers())
		if handled {
			return err // Shortcut was handled, don't process as regular input
		}
	}

	// Process as regular key input
	return ip.processKeyEvent(event)
}
