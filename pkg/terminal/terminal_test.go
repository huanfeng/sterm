package terminal

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestTerminalState_Validate(t *testing.T) {
	tests := []struct {
		name    string
		state   TerminalState
		wantErr bool
	}{
		{
			name: "valid state",
			state: TerminalState{
				CursorX:      10,
				CursorY:      5,
				Width:        80,
				Height:       24,
				ScrollTop:    0,
				ScrollBottom: 23,
			},
			wantErr: false,
		},
		{
			name: "zero width",
			state: TerminalState{
				CursorX:      0,
				CursorY:      0,
				Width:        0,
				Height:       24,
				ScrollTop:    0,
				ScrollBottom: 23,
			},
			wantErr: true,
		},
		{
			name: "zero height",
			state: TerminalState{
				CursorX:      0,
				CursorY:      0,
				Width:        80,
				Height:       0,
				ScrollTop:    0,
				ScrollBottom: 0,
			},
			wantErr: true,
		},
		{
			name: "cursor X out of bounds",
			state: TerminalState{
				CursorX:      80,
				CursorY:      0,
				Width:        80,
				Height:       24,
				ScrollTop:    0,
				ScrollBottom: 23,
			},
			wantErr: true,
		},
		{
			name: "cursor Y out of bounds",
			state: TerminalState{
				CursorX:      0,
				CursorY:      24,
				Width:        80,
				Height:       24,
				ScrollTop:    0,
				ScrollBottom: 23,
			},
			wantErr: true,
		},
		{
			name: "scroll top out of bounds",
			state: TerminalState{
				CursorX:      0,
				CursorY:      0,
				Width:        80,
				Height:       24,
				ScrollTop:    24,
				ScrollBottom: 23,
			},
			wantErr: true,
		},
		{
			name: "scroll bottom out of bounds",
			state: TerminalState{
				CursorX:      0,
				CursorY:      0,
				Width:        80,
				Height:       24,
				ScrollTop:    0,
				ScrollBottom: 24,
			},
			wantErr: true,
		},
		{
			name: "scroll top greater than scroll bottom",
			state: TerminalState{
				CursorX:      0,
				CursorY:      0,
				Width:        80,
				Height:       24,
				ScrollTop:    10,
				ScrollBottom: 5,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.state.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("TerminalState.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultTerminalState(t *testing.T) {
	width, height := 80, 24
	state := DefaultTerminalState(width, height)

	if err := state.Validate(); err != nil {
		t.Errorf("DefaultTerminalState() should return valid state: %v", err)
	}

	if state.Width != width {
		t.Errorf("DefaultTerminalState() Width = %d, want %d", state.Width, width)
	}

	if state.Height != height {
		t.Errorf("DefaultTerminalState() Height = %d, want %d", state.Height, height)
	}

	if state.CursorX != 0 {
		t.Errorf("DefaultTerminalState() CursorX = %d, want 0", state.CursorX)
	}

	if state.CursorY != 0 {
		t.Errorf("DefaultTerminalState() CursorY = %d, want 0", state.CursorY)
	}

	if state.ScrollTop != 0 {
		t.Errorf("DefaultTerminalState() ScrollTop = %d, want 0", state.ScrollTop)
	}

	if state.ScrollBottom != height-1 {
		t.Errorf("DefaultTerminalState() ScrollBottom = %d, want %d", state.ScrollBottom, height-1)
	}

	if state.MouseMode != MouseModeOff {
		t.Errorf("DefaultTerminalState() MouseMode = %v, want %v", state.MouseMode, MouseModeOff)
	}

	if state.IsRunning {
		t.Error("DefaultTerminalState() IsRunning should be false")
	}
}

func TestColor_String(t *testing.T) {
	tests := []struct {
		color    Color
		expected string
	}{
		{ColorBlack, "black"},
		{ColorRed, "red"},
		{ColorGreen, "green"},
		{ColorYellow, "yellow"},
		{ColorBlue, "blue"},
		{ColorMagenta, "magenta"},
		{ColorCyan, "cyan"},
		{ColorWhite, "white"},
		{ColorBrightBlack, "bright_black"},
		{ColorBrightRed, "bright_red"},
		{ColorBrightGreen, "bright_green"},
		{ColorBrightYellow, "bright_yellow"},
		{ColorBrightBlue, "bright_blue"},
		{ColorBrightMagenta, "bright_magenta"},
		{ColorBrightCyan, "bright_cyan"},
		{ColorBrightWhite, "bright_white"},
		{Color(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.color.String(); got != tt.expected {
				t.Errorf("Color.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestMouseMode_String(t *testing.T) {
	tests := []struct {
		mode     MouseMode
		expected string
	}{
		{MouseModeOff, "off"},
		{MouseModeX10, "x10"},
		{MouseModeVT200, "vt200"},
		{MouseModeVT200Highlight, "vt200_highlight"},
		{MouseModeBtnEvent, "btn_event"},
		{MouseModeAnyEvent, "any_event"},
		{MouseMode(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.mode.String(); got != tt.expected {
				t.Errorf("MouseMode.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDefaultTextAttributes(t *testing.T) {
	attrs := DefaultTextAttributes()

	if attrs.Foreground != ColorDefault {
		t.Errorf("DefaultTextAttributes() Foreground = %v, want %v", attrs.Foreground, ColorDefault)
	}

	if attrs.Background != ColorDefault {
		t.Errorf("DefaultTextAttributes() Background = %v, want %v", attrs.Background, ColorDefault)
	}

	if attrs.Bold {
		t.Error("DefaultTextAttributes() Bold should be false")
	}

	if attrs.Italic {
		t.Error("DefaultTextAttributes() Italic should be false")
	}

	if attrs.Underline {
		t.Error("DefaultTextAttributes() Underline should be false")
	}

	if attrs.Reverse {
		t.Error("DefaultTextAttributes() Reverse should be false")
	}

	if attrs.Blink {
		t.Error("DefaultTextAttributes() Blink should be false")
	}
}

func TestNewScreen(t *testing.T) {
	width, height := 80, 24
	screen := NewScreen(width, height)

	if screen == nil {
		t.Error("NewScreen() returned nil")
	}

	if screen.Width != width {
		t.Errorf("NewScreen() Width = %d, want %d", screen.Width, width)
	}

	if screen.Height != height {
		t.Errorf("NewScreen() Height = %d, want %d", screen.Height, height)
	}

	if len(screen.Buffer) != height {
		t.Errorf("NewScreen() Buffer height = %d, want %d", len(screen.Buffer), height)
	}

	for i, row := range screen.Buffer {
		if len(row) != width {
			t.Errorf("NewScreen() Buffer row %d width = %d, want %d", i, len(row), width)
		}

		for j, cell := range row {
			if cell.Char != ' ' {
				t.Errorf("NewScreen() Buffer[%d][%d].Char = %c, want space", i, j, cell.Char)
			}

			if cell.Attributes.Foreground != ColorDefault {
				t.Errorf("NewScreen() Buffer[%d][%d] should have default attributes", i, j)
			}
		}
	}

	if !screen.Dirty {
		t.Error("NewScreen() should mark screen as dirty")
	}
}

func TestNewVTParser(t *testing.T) {
	parser := NewVTParser()

	if parser == nil {
		t.Error("NewVTParser() returned nil")
	}

	if parser.State != StateGround {
		t.Errorf("NewVTParser() State = %v, want %v", parser.State, StateGround)
	}

	if parser.Buffer == nil {
		t.Error("NewVTParser() Buffer should not be nil")
	}

	if cap(parser.Buffer) != 256 {
		t.Errorf("NewVTParser() Buffer capacity = %d, want 256", cap(parser.Buffer))
	}

	if parser.Params == nil {
		t.Error("NewVTParser() Params should not be nil")
	}

	if cap(parser.Params) != 16 {
		t.Errorf("NewVTParser() Params capacity = %d, want 16", cap(parser.Params))
	}

	if parser.Intermediate == nil {
		t.Error("NewVTParser() Intermediate should not be nil")
	}
}

func TestVTParser_Reset(t *testing.T) {
	parser := NewVTParser()

	// Modify parser state
	parser.State = StateCSI
	parser.Buffer = append(parser.Buffer, 'A', 'B', 'C')
	parser.Params = append(parser.Params, 1, 2, 3)
	parser.Intermediate = append(parser.Intermediate, 'X')

	// Reset
	parser.Reset()

	if parser.State != StateGround {
		t.Errorf("Reset() State = %v, want %v", parser.State, StateGround)
	}

	if len(parser.Buffer) != 0 {
		t.Errorf("Reset() Buffer length = %d, want 0", len(parser.Buffer))
	}

	if len(parser.Params) != 0 {
		t.Errorf("Reset() Params length = %d, want 0", len(parser.Params))
	}

	if len(parser.Intermediate) != 0 {
		t.Errorf("Reset() Intermediate length = %d, want 0", len(parser.Intermediate))
	}
}

func TestVTParser_ParseByte_PrintableCharacters(t *testing.T) {
	parser := NewVTParser()
	screen := NewScreen(80, 24)
	state := DefaultTerminalState(80, 24)
	utf8Decoder := NewUTF8Decoder()

	// Test printable ASCII characters
	testChars := []byte{'A', 'B', 'C', '1', '2', '3', ' ', '!', '@'}

	for _, ch := range testChars {
		actions := parser.ParseByte(ch, screen, &state, utf8Decoder)

		if len(actions) != 1 {
			t.Errorf("ParseByte(%c) returned %d actions, want 1", ch, len(actions))
			continue
		}

		if actions[0].Type != ActionPrint {
			t.Errorf("ParseByte(%c) action type = %v, want %v", ch, actions[0].Type, ActionPrint)
		}

		if actions[0].Data != rune(ch) {
			t.Errorf("ParseByte(%c) action data = %v, want %v", ch, actions[0].Data, rune(ch))
		}
	}
}

func TestVTParser_ParseByte_ControlCharacters(t *testing.T) {
	parser := NewVTParser()
	screen := NewScreen(80, 24)
	state := DefaultTerminalState(80, 24)
	utf8Decoder := NewUTF8Decoder()

	tests := []struct {
		char     byte
		expected ActionType
	}{
		{0x07, ActionBell},
		{0x08, ActionBackspace},
		{0x09, ActionTab},
		{0x0A, ActionNewline},
		{0x0D, ActionCarriageReturn},
	}

	for _, tt := range tests {
		parser.Reset()
		actions := parser.ParseByte(tt.char, screen, &state, utf8Decoder)

		if len(actions) != 1 {
			t.Errorf("ParseByte(0x%02X) returned %d actions, want 1", tt.char, len(actions))
			continue
		}

		if actions[0].Type != tt.expected {
			t.Errorf("ParseByte(0x%02X) action type = %v, want %v", tt.char, actions[0].Type, tt.expected)
		}
	}
}

func TestVTParser_ParseByte_EscapeSequence(t *testing.T) {
	parser := NewVTParser()
	screen := NewScreen(80, 24)
	state := DefaultTerminalState(80, 24)
	utf8Decoder := NewUTF8Decoder()

	// Test ESC character
	actions := parser.ParseByte(0x1B, screen, &state, utf8Decoder)

	if len(actions) != 0 {
		t.Errorf("ParseByte(ESC) returned %d actions, want 0", len(actions))
	}

	if parser.State != StateEscape {
		t.Errorf("ParseByte(ESC) state = %v, want %v", parser.State, StateEscape)
	}
}

func TestVTParser_ParseByte_CSISequence(t *testing.T) {
	parser := NewVTParser()
	screen := NewScreen(80, 24)
	state := DefaultTerminalState(80, 24)
	utf8Decoder := NewUTF8Decoder()

	// Test CSI sequence: ESC[2J (clear screen)
	sequence := []byte{0x1B, '[', '2', 'J'}
	var allActions []Action

	for _, b := range sequence {
		actions := parser.ParseByte(b, screen, &state, utf8Decoder)
		allActions = append(allActions, actions...)
	}

	if len(allActions) != 1 {
		t.Errorf("CSI sequence returned %d actions, want 1", len(allActions))
	}

	if allActions[0].Type != ActionClearScreen {
		t.Errorf("CSI sequence action type = %v, want %v", allActions[0].Type, ActionClearScreen)
	}

	if allActions[0].Data != 2 {
		t.Errorf("CSI sequence action data = %v, want 2", allActions[0].Data)
	}
}

func TestVTParser_ParseByte_CursorMovement(t *testing.T) {
	parser := NewVTParser()
	screen := NewScreen(80, 24)
	state := DefaultTerminalState(80, 24)
	utf8Decoder := NewUTF8Decoder()

	tests := []struct {
		sequence  []byte
		direction string
		count     int
	}{
		{[]byte{0x1B, '[', 'A'}, "up", 1},
		{[]byte{0x1B, '[', '5', 'A'}, "up", 5},
		{[]byte{0x1B, '[', 'B'}, "down", 1},
		{[]byte{0x1B, '[', '3', 'B'}, "down", 3},
		{[]byte{0x1B, '[', 'C'}, "right", 1},
		{[]byte{0x1B, '[', '2', 'C'}, "right", 2},
		{[]byte{0x1B, '[', 'D'}, "left", 1},
		{[]byte{0x1B, '[', '4', 'D'}, "left", 4},
	}

	for _, tt := range tests {
		parser.Reset()
		var actions []Action

		for _, b := range tt.sequence {
			newActions := parser.ParseByte(b, screen, &state, utf8Decoder)
			actions = append(actions, newActions...)
		}

		if len(actions) != 1 {
			t.Errorf("Sequence %v returned %d actions, want 1", tt.sequence, len(actions))
			continue
		}

		if actions[0].Type != ActionMoveCursor {
			t.Errorf("Sequence %v action type = %v, want %v", tt.sequence, actions[0].Type, ActionMoveCursor)
			continue
		}

		move, ok := actions[0].Data.(CursorMove)
		if !ok {
			t.Errorf("Sequence %v action data is not CursorMove", tt.sequence)
			continue
		}

		if move.Direction != tt.direction {
			t.Errorf("Sequence %v direction = %s, want %s", tt.sequence, move.Direction, tt.direction)
		}

		if move.Count != tt.count {
			t.Errorf("Sequence %v count = %d, want %d", tt.sequence, move.Count, tt.count)
		}
	}
}

func TestVTParser_ParseByte_CursorPosition(t *testing.T) {
	parser := NewVTParser()
	screen := NewScreen(80, 24)
	state := DefaultTerminalState(80, 24)
	utf8Decoder := NewUTF8Decoder()

	// Test cursor position: ESC[10;20H
	sequence := []byte{0x1B, '[', '1', '0', ';', '2', '0', 'H'}
	var actions []Action

	for _, b := range sequence {
		newActions := parser.ParseByte(b, screen, &state, utf8Decoder)
		actions = append(actions, newActions...)
	}

	if len(actions) != 1 {
		t.Errorf("Cursor position sequence returned %d actions, want 1", len(actions))
	}

	if actions[0].Type != ActionMoveCursor {
		t.Errorf("Cursor position action type = %v, want %v", actions[0].Type, ActionMoveCursor)
	}

	move, ok := actions[0].Data.(CursorMove)
	if !ok {
		t.Error("Cursor position action data is not CursorMove")
	}

	if move.Direction != "absolute" {
		t.Errorf("Cursor position direction = %s, want absolute", move.Direction)
	}

	if move.Row != 9 { // 10-1 (1-based to 0-based)
		t.Errorf("Cursor position row = %d, want 9", move.Row)
	}

	if move.Col != 19 { // 20-1 (1-based to 0-based)
		t.Errorf("Cursor position col = %d, want 19", move.Col)
	}
}

func TestVTParser_ParseByte_SGR(t *testing.T) {
	parser := NewVTParser()
	screen := NewScreen(80, 24)
	state := DefaultTerminalState(80, 24)
	utf8Decoder := NewUTF8Decoder()

	// Test SGR sequence: ESC[1;31m (bold red)
	sequence := []byte{0x1B, '[', '1', ';', '3', '1', 'm'}
	var actions []Action

	for _, b := range sequence {
		newActions := parser.ParseByte(b, screen, &state, utf8Decoder)
		actions = append(actions, newActions...)
	}

	if len(actions) != 2 {
		t.Errorf("SGR sequence returned %d actions, want 2", len(actions))
	}

	// Check bold action
	if actions[0].Type != ActionSetAttribute {
		t.Errorf("First SGR action type = %v, want %v", actions[0].Type, ActionSetAttribute)
	}

	attr1, ok := actions[0].Data.(AttributeChange)
	if !ok {
		t.Error("First SGR action data is not AttributeChange")
	}

	if attr1.Bold == nil || !*attr1.Bold {
		t.Error("First SGR action should set bold to true")
	}

	// Check color action
	if actions[1].Type != ActionSetAttribute {
		t.Errorf("Second SGR action type = %v, want %v", actions[1].Type, ActionSetAttribute)
	}

	attr2, ok := actions[1].Data.(AttributeChange)
	if !ok {
		t.Error("Second SGR action data is not AttributeChange")
	}

	if attr2.Foreground == nil || *attr2.Foreground != ColorRed {
		t.Error("Second SGR action should set foreground to red")
	}
}

func TestVTParser_ParseByte_ComplexSequences(t *testing.T) {
	parser := NewVTParser()
	screen := NewScreen(80, 24)
	state := DefaultTerminalState(80, 24)
	utf8Decoder := NewUTF8Decoder()

	tests := []struct {
		name            string
		sequence        []byte
		expectedActions []struct {
			actionType ActionType
			validation func(Action) bool
		}
	}{
		{
			name:     "Erase to end of line",
			sequence: []byte{0x1B, '[', 'K'},
			expectedActions: []struct {
				actionType ActionType
				validation func(Action) bool
			}{
				{ActionClearLine, func(a Action) bool { return a.Data == 0 }},
			},
		},
		{
			name:     "Erase entire line",
			sequence: []byte{0x1B, '[', '2', 'K'},
			expectedActions: []struct {
				actionType ActionType
				validation func(Action) bool
			}{
				{ActionClearLine, func(a Action) bool { return a.Data == 2 }},
			},
		},
		{
			name:     "Set scroll region",
			sequence: []byte{0x1B, '[', '5', ';', '2', '0', 'r'},
			expectedActions: []struct {
				actionType ActionType
				validation func(Action) bool
			}{
				{ActionSetScrollRegion, func(a Action) bool {
					if region, ok := a.Data.(ScrollRegion); ok {
						return region.Top == 4 && region.Bottom == 19
					}
					return false
				}},
			},
		},
		{
			name:     "Delete characters",
			sequence: []byte{0x1B, '[', '3', 'P'},
			expectedActions: []struct {
				actionType ActionType
				validation func(Action) bool
			}{
				{ActionDeleteChar, func(a Action) bool { return a.Data == 3 }},
			},
		},
		{
			name:     "Insert characters",
			sequence: []byte{0x1B, '[', '2', '@'},
			expectedActions: []struct {
				actionType ActionType
				validation func(Action) bool
			}{
				{ActionInsertChar, func(a Action) bool { return a.Data == 2 }},
			},
		},
		{
			name:     "256 color foreground",
			sequence: []byte{0x1B, '[', '3', '8', ';', '5', ';', '1', '2', '3', 'm'},
			expectedActions: []struct {
				actionType ActionType
				validation func(Action) bool
			}{
				// This would require extending the parser to handle 256 colors
				// For now, we test that it doesn't crash
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser.Reset()
			var actions []Action

			for _, b := range tt.sequence {
				newActions := parser.ParseByte(b, screen, &state, utf8Decoder)
				actions = append(actions, newActions...)
			}

			if len(tt.expectedActions) > 0 {
				if len(actions) != len(tt.expectedActions) {
					t.Errorf("%s: got %d actions, want %d", tt.name, len(actions), len(tt.expectedActions))
					return
				}

				for i, expected := range tt.expectedActions {
					if actions[i].Type != expected.actionType {
						t.Errorf("%s: action[%d] type = %v, want %v", tt.name, i, actions[i].Type, expected.actionType)
					}

					if expected.validation != nil && !expected.validation(actions[i]) {
						t.Errorf("%s: action[%d] validation failed", tt.name, i)
					}
				}
			}
		})
	}
}

func TestVTParser_ParseByte_EscapeSequences(t *testing.T) {
	parser := NewVTParser()
	screen := NewScreen(80, 24)
	state := DefaultTerminalState(80, 24)
	utf8Decoder := NewUTF8Decoder()

	tests := []struct {
		name           string
		sequence       []byte
		expectedAction ActionType
	}{
		{"Index", []byte{0x1B, 'D'}, ActionScroll},
		{"Reverse Index", []byte{0x1B, 'M'}, ActionScroll},
		{"Next Line", []byte{0x1B, 'E'}, ActionNewline},
		{"Set Keypad Application Mode", []byte{0x1B, '='}, ActionSetMode},
		{"Set Keypad Numeric Mode", []byte{0x1B, '>'}, ActionSetMode},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser.Reset()
			var actions []Action

			for _, b := range tt.sequence {
				newActions := parser.ParseByte(b, screen, &state, utf8Decoder)
				actions = append(actions, newActions...)
			}

			if len(actions) == 0 {
				t.Errorf("%s: no actions returned", tt.name)
				return
			}

			found := false
			for _, action := range actions {
				if action.Type == tt.expectedAction {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("%s: expected action type %v not found", tt.name, tt.expectedAction)
			}
		})
	}
}

func TestVTParser_ParseByte_BrightColors(t *testing.T) {
	parser := NewVTParser()
	screen := NewScreen(80, 24)
	state := DefaultTerminalState(80, 24)
	utf8Decoder := NewUTF8Decoder()

	tests := []struct {
		name          string
		sequence      []byte
		expectedColor Color
		isForeground  bool
	}{
		{"Bright Red Foreground", []byte{0x1B, '[', '9', '1', 'm'}, ColorBrightRed, true},
		{"Bright Green Foreground", []byte{0x1B, '[', '9', '2', 'm'}, ColorBrightGreen, true},
		{"Bright Yellow Foreground", []byte{0x1B, '[', '9', '3', 'm'}, ColorBrightYellow, true},
		{"Bright Blue Foreground", []byte{0x1B, '[', '9', '4', 'm'}, ColorBrightBlue, true},
		{"Bright Magenta Foreground", []byte{0x1B, '[', '9', '5', 'm'}, ColorBrightMagenta, true},
		{"Bright Cyan Foreground", []byte{0x1B, '[', '9', '6', 'm'}, ColorBrightCyan, true},
		{"Bright White Foreground", []byte{0x1B, '[', '9', '7', 'm'}, ColorBrightWhite, true},
		{"Bright Red Background", []byte{0x1B, '[', '1', '0', '1', 'm'}, ColorBrightRed, false},
		{"Bright Green Background", []byte{0x1B, '[', '1', '0', '2', 'm'}, ColorBrightGreen, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser.Reset()
			var actions []Action

			for _, b := range tt.sequence {
				newActions := parser.ParseByte(b, screen, &state, utf8Decoder)
				actions = append(actions, newActions...)
			}

			if len(actions) != 1 {
				t.Errorf("%s: got %d actions, want 1", tt.name, len(actions))
				return
			}

			if actions[0].Type != ActionSetAttribute {
				t.Errorf("%s: action type = %v, want %v", tt.name, actions[0].Type, ActionSetAttribute)
				return
			}

			attr, ok := actions[0].Data.(AttributeChange)
			if !ok {
				t.Errorf("%s: action data is not AttributeChange", tt.name)
				return
			}

			if tt.isForeground {
				if attr.Foreground == nil || *attr.Foreground != tt.expectedColor {
					t.Errorf("%s: foreground color = %v, want %v", tt.name, attr.Foreground, tt.expectedColor)
				}
			} else {
				if attr.Background == nil || *attr.Background != tt.expectedColor {
					t.Errorf("%s: background color = %v, want %v", tt.name, attr.Background, tt.expectedColor)
				}
			}
		})
	}
}

func TestVTParser_parseParams(t *testing.T) {
	parser := NewVTParser()

	tests := []struct {
		input    string
		expected []int
	}{
		{"", []int{}},
		{"1", []int{1}},
		{"1;2;3", []int{1, 2, 3}},
		{"10;20", []int{10, 20}},
		{"1;;3", []int{1, 0, 3}},
		{"1;", []int{1, 0}},
		{";2", []int{0, 2}},
	}

	for _, tt := range tests {
		parser.Buffer = []byte(tt.input)
		parser.parseParams()

		if len(parser.Params) != len(tt.expected) {
			t.Errorf("parseParams(%s) length = %d, want %d", tt.input, len(parser.Params), len(tt.expected))
			continue
		}

		for i, expected := range tt.expected {
			if parser.Params[i] != expected {
				t.Errorf("parseParams(%s)[%d] = %d, want %d", tt.input, i, parser.Params[i], expected)
			}
		}
	}
}

func TestVTParser_getParam(t *testing.T) {
	parser := NewVTParser()
	parser.Params = []int{10, 20, 30}

	tests := []struct {
		index        int
		defaultValue int
		expected     int
	}{
		{0, 1, 10},
		{1, 1, 20},
		{2, 1, 30},
		{3, 1, 1},   // Out of bounds, should return default
		{5, 99, 99}, // Out of bounds, should return default
	}

	for _, tt := range tests {
		result := parser.getParam(tt.index, tt.defaultValue)
		if result != tt.expected {
			t.Errorf("getParam(%d, %d) = %d, want %d", tt.index, tt.defaultValue, result, tt.expected)
		}
	}
}

func TestTerminalEmulator_Start(t *testing.T) {
	emulator := NewTerminalEmulator(nil, nil, 80, 24)

	err := emulator.Start()
	if err != nil {
		t.Errorf("Start() failed: %v", err)
	}

	if !emulator.IsRunning() {
		t.Error("Terminal should be running after Start()")
	}

	if !emulator.GetState().IsRunning {
		t.Error("Terminal state should indicate running")
	}

	// Test double start
	err = emulator.Start()
	if err == nil {
		t.Error("Start() should fail when already running")
	}
}

func TestTerminalEmulator_Stop(t *testing.T) {
	emulator := NewTerminalEmulator(nil, nil, 80, 24)

	// Start first
	emulator.Start()

	err := emulator.Stop()
	if err != nil {
		t.Errorf("Stop() failed: %v", err)
	}

	if emulator.IsRunning() {
		t.Error("Terminal should not be running after Stop()")
	}

	if emulator.GetState().IsRunning {
		t.Error("Terminal state should indicate not running")
	}

	// Test double stop
	err = emulator.Stop()
	if err != nil {
		t.Errorf("Stop() should not fail when already stopped: %v", err)
	}
}

func TestTerminalEmulator_Resize(t *testing.T) {
	emulator := NewTerminalEmulator(nil, nil, 80, 24)

	err := emulator.Resize(100, 30)
	if err != nil {
		t.Errorf("Resize() failed: %v", err)
	}

	state := emulator.GetState()
	if state.Width != 100 {
		t.Errorf("Width after resize = %d, want 100", state.Width)
	}

	if state.Height != 30 {
		t.Errorf("Height after resize = %d, want 30", state.Height)
	}

	// Test invalid dimensions
	err = emulator.Resize(0, 24)
	if err == nil {
		t.Error("Resize() should fail with zero width")
	}

	err = emulator.Resize(80, 0)
	if err == nil {
		t.Error("Resize() should fail with zero height")
	}

	err = emulator.Resize(-10, 24)
	if err == nil {
		t.Error("Resize() should fail with negative width")
	}
}

func TestTerminalEmulator_EnableMouse(t *testing.T) {
	emulator := NewTerminalEmulator(nil, nil, 80, 24)

	// Enable mouse
	err := emulator.EnableMouse(true)
	if err != nil {
		t.Errorf("EnableMouse(true) failed: %v", err)
	}

	state := emulator.GetState()
	if state.MouseMode == MouseModeOff {
		t.Error("Mouse mode should be enabled")
	}

	// Disable mouse
	err = emulator.EnableMouse(false)
	if err != nil {
		t.Errorf("EnableMouse(false) failed: %v", err)
	}

	state = emulator.GetState()
	if state.MouseMode != MouseModeOff {
		t.Error("Mouse mode should be disabled")
	}
}

func TestTerminalEmulator_SetState(t *testing.T) {
	emulator := NewTerminalEmulator(nil, nil, 80, 24)

	newState := DefaultTerminalState(80, 24)
	newState.CursorX = 10
	newState.CursorY = 5
	newState.Attributes.Bold = true

	err := emulator.SetState(newState)
	if err != nil {
		t.Errorf("SetState() failed: %v", err)
	}

	state := emulator.GetState()
	if state.CursorX != 10 {
		t.Errorf("CursorX = %d, want 10", state.CursorX)
	}

	if state.CursorY != 5 {
		t.Errorf("CursorY = %d, want 5", state.CursorY)
	}

	if !state.Attributes.Bold {
		t.Error("Bold attribute should be set")
	}

	// Test invalid state
	invalidState := DefaultTerminalState(80, 24)
	invalidState.CursorX = 100 // Out of bounds

	err = emulator.SetState(invalidState)
	if err == nil {
		t.Error("SetState() should fail with invalid state")
	}
}

func TestTerminalEmulator_PrintChar(t *testing.T) {
	emulator := NewTerminalEmulator(nil, nil, 80, 24)

	// Print a character
	emulator.printChar('A')

	// Check that character was placed in buffer
	cell := emulator.screen.Buffer[0][0]
	if cell.Char != 'A' {
		t.Errorf("Character at (0,0) = %c, want A", cell.Char)
	}

	// Check cursor moved
	state := emulator.GetState()
	if state.CursorX != 1 {
		t.Errorf("CursorX after print = %d, want 1", state.CursorX)
	}

	// Check screen is marked dirty
	if !emulator.screen.Dirty {
		t.Error("Screen should be marked dirty after printing")
	}
}

func TestTerminalEmulator_MoveCursor(t *testing.T) {
	emulator := NewTerminalEmulator(nil, nil, 80, 24)

	tests := []struct {
		move      CursorMove
		startX    int
		startY    int
		expectedX int
		expectedY int
	}{
		{CursorMove{Direction: "up", Count: 3}, 10, 10, 10, 7},
		{CursorMove{Direction: "down", Count: 2}, 10, 10, 10, 12},
		{CursorMove{Direction: "left", Count: 5}, 10, 10, 5, 10},
		{CursorMove{Direction: "right", Count: 3}, 10, 10, 13, 10},
		{CursorMove{Direction: "absolute", Row: 5, Col: 15}, 10, 10, 15, 5},
	}

	for _, tt := range tests {
		emulator.state.CursorX = tt.startX
		emulator.state.CursorY = tt.startY

		emulator.moveCursor(tt.move)

		state := emulator.GetState()
		if state.CursorX != tt.expectedX {
			t.Errorf("Move %v from (%d,%d): CursorX = %d, want %d", tt.move, tt.startX, tt.startY, state.CursorX, tt.expectedX)
		}

		if state.CursorY != tt.expectedY {
			t.Errorf("Move %v from (%d,%d): CursorY = %d, want %d", tt.move, tt.startX, tt.startY, state.CursorY, tt.expectedY)
		}
	}
}

func TestTerminalEmulator_ClearScreen(t *testing.T) {
	emulator := NewTerminalEmulator(nil, nil, 80, 24)

	// Fill screen with characters
	for y := 0; y < 24; y++ {
		for x := 0; x < 80; x++ {
			emulator.screen.Buffer[y][x] = Cell{Char: 'X', Attributes: DefaultTextAttributes()}
		}
	}

	// Set cursor position
	emulator.state.CursorX = 40
	emulator.state.CursorY = 12

	// Clear entire screen
	emulator.clearScreen(2)

	// Check that screen is cleared
	for y := 0; y < 24; y++ {
		for x := 0; x < 80; x++ {
			cell := emulator.screen.Buffer[y][x]
			if cell.Char != ' ' {
				t.Errorf("Character at (%d,%d) = %c, want space", x, y, cell.Char)
			}
		}
	}

	// Check screen is marked dirty
	if !emulator.screen.Dirty {
		t.Error("Screen should be marked dirty after clearing")
	}
}

func TestTerminalEmulator_SetAttribute(t *testing.T) {
	emulator := NewTerminalEmulator(nil, nil, 80, 24)

	// Test setting bold
	boldTrue := true
	change := AttributeChange{Bold: &boldTrue}
	emulator.setAttribute(change)

	if !emulator.state.Attributes.Bold {
		t.Error("Bold attribute should be set")
	}

	// Test setting color
	red := ColorRed
	change = AttributeChange{Foreground: &red}
	emulator.setAttribute(change)

	if emulator.state.Attributes.Foreground != ColorRed {
		t.Error("Foreground color should be red")
	}

	// Test reset
	change = AttributeChange{Reset: true}
	emulator.setAttribute(change)

	defaultAttrs := DefaultTextAttributes()
	if emulator.state.Attributes != defaultAttrs {
		t.Error("Attributes should be reset to default")
	}
}

func TestTerminalEmulator_CompleteANSIProcessing(t *testing.T) {
	emulator := NewTerminalEmulator(nil, nil, 80, 24)
	emulator.Start()

	tests := []struct {
		name   string
		output []byte
		verify func(*testing.T, *TerminalEmulator)
	}{
		{
			name:   "Text with color codes",
			output: []byte("\x1B[31mRed\x1B[0m Normal \x1B[1;32mBold Green\x1B[0m"),
			verify: func(t *testing.T, te *TerminalEmulator) {
				// After processing, verify the screen buffer contains the text
				expectedText := "Red Normal Bold Green"
				var actualText string
				for x := 0; x < len(expectedText) && x < te.state.Width; x++ {
					if te.screen.Buffer[0][x].Char != ' ' {
						actualText += string(te.screen.Buffer[0][x].Char)
					}
				}
				if len(actualText) == 0 {
					t.Error("No text found in screen buffer")
				}
			},
		},
		{
			name:   "Cursor positioning and clear",
			output: []byte("Hello\x1B[H\x1B[2JCleared"),
			verify: func(t *testing.T, te *TerminalEmulator) {
				// After clear screen, only "Cleared" should be visible
				if te.state.CursorX != 7 { // Length of "Cleared"
					t.Errorf("CursorX = %d, want 7", te.state.CursorX)
				}
				if te.state.CursorY != 0 {
					t.Errorf("CursorY = %d, want 0", te.state.CursorY)
				}
			},
		},
		{
			name:   "Tab and newline handling",
			output: []byte("A\tB\nC\rD"),
			verify: func(t *testing.T, te *TerminalEmulator) {
				// A should be at position 0
				if te.screen.Buffer[0][0].Char != 'A' {
					t.Errorf("Char at (0,0) = %c, want A", te.screen.Buffer[0][0].Char)
				}
				// B should be at tab position (8)
				if te.screen.Buffer[0][8].Char != 'B' {
					t.Errorf("Char at (8,0) = %c, want B", te.screen.Buffer[0][8].Char)
				}
				// D should be at position 0 of line 1 (after CR)
				if te.screen.Buffer[1][0].Char != 'D' {
					t.Errorf("Char at (0,1) = %c, want D", te.screen.Buffer[1][0].Char)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset emulator for each test
			emulator.screen = NewScreen(80, 24)
			emulator.state = DefaultTerminalState(80, 24)
			emulator.parser.Reset()

			err := emulator.ProcessOutput(tt.output)
			if err != nil {
				t.Errorf("ProcessOutput() failed: %v", err)
				return
			}

			tt.verify(t, emulator)
		})
	}
}

func TestTerminalEmulator_Scroll(t *testing.T) {
	emulator := NewTerminalEmulator(nil, nil, 80, 24)

	// Fill first line with characters
	for x := 0; x < 80; x++ {
		emulator.screen.Buffer[0][x] = Cell{Char: 'A', Attributes: DefaultTextAttributes()}
	}

	// Fill second line with different characters
	for x := 0; x < 80; x++ {
		emulator.screen.Buffer[1][x] = Cell{Char: 'B', Attributes: DefaultTextAttributes()}
	}

	// Scroll up
	emulator.scroll("up")

	// Check that first line now contains 'B'
	for x := 0; x < 80; x++ {
		cell := emulator.screen.Buffer[0][x]
		if cell.Char != 'B' {
			t.Errorf("Character at (0,%d) = %c, want B", x, cell.Char)
		}
	}

	// Check that last line is cleared
	for x := 0; x < 80; x++ {
		cell := emulator.screen.Buffer[23][x]
		if cell.Char != ' ' {
			t.Errorf("Character at (23,%d) = %c, want space", x, cell.Char)
		}
	}
}

func TestTerminalEmulator_Tab(t *testing.T) {
	emulator := NewTerminalEmulator(nil, nil, 80, 24)

	tests := []struct {
		startX    int
		expectedX int
	}{
		{0, 8},
		{1, 8},
		{7, 8},
		{8, 16},
		{15, 16},
		{16, 24},
		{72, 79}, // Should not exceed width
	}

	for _, tt := range tests {
		emulator.state.CursorX = tt.startX
		emulator.tab()

		if emulator.state.CursorX != tt.expectedX {
			t.Errorf("Tab from %d: CursorX = %d, want %d", tt.startX, emulator.state.CursorX, tt.expectedX)
		}
	}
}

func TestTerminalEmulator_NewlineAndCarriageReturn(t *testing.T) {
	emulator := NewTerminalEmulator(nil, nil, 80, 24)

	// Set cursor position
	emulator.state.CursorX = 10
	emulator.state.CursorY = 5

	// Test newline
	emulator.newline()

	if emulator.state.CursorY != 6 {
		t.Errorf("CursorY after newline = %d, want 6", emulator.state.CursorY)
	}

	// Test carriage return
	emulator.carriageReturn()

	if emulator.state.CursorX != 0 {
		t.Errorf("CursorX after carriage return = %d, want 0", emulator.state.CursorX)
	}
}

func TestTerminalEmulator_Backspace(t *testing.T) {
	emulator := NewTerminalEmulator(nil, nil, 80, 24)

	// Set cursor position
	emulator.state.CursorX = 10

	// Test backspace
	emulator.backspace()

	if emulator.state.CursorX != 9 {
		t.Errorf("CursorX after backspace = %d, want 9", emulator.state.CursorX)
	}

	// Test backspace at beginning of line
	emulator.state.CursorX = 0
	emulator.backspace()

	if emulator.state.CursorX != 0 {
		t.Errorf("CursorX should remain 0 when backspacing at beginning, got %d", emulator.state.CursorX)
	}
}

func TestMouseButton_String(t *testing.T) {
	tests := []struct {
		button   MouseButton
		expected string
	}{
		{MouseButtonNone, "none"},
		{MouseButtonLeft, "left"},
		{MouseButtonMiddle, "middle"},
		{MouseButtonRight, "right"},
		{MouseButtonWheelUp, "wheel_up"},
		{MouseButtonWheelDown, "wheel_down"},
		{MouseButton(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.button.String(); got != tt.expected {
				t.Errorf("MouseButton.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestMouseAction_String(t *testing.T) {
	tests := []struct {
		action   MouseAction
		expected string
	}{
		{MouseActionPress, "press"},
		{MouseActionRelease, "release"},
		{MouseActionMove, "move"},
		{MouseActionDrag, "drag"},
		{MouseAction(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.action.String(); got != tt.expected {
				t.Errorf("MouseAction.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNewMouseHandler(t *testing.T) {
	handler := NewMouseHandler()

	if handler == nil {
		t.Error("NewMouseHandler() returned nil")
	}

	if handler.GetMode() != MouseModeOff {
		t.Errorf("NewMouseHandler() mode = %v, want %v", handler.GetMode(), MouseModeOff)
	}

	if handler.buttonState == nil {
		t.Error("NewMouseHandler() buttonState should not be nil")
	}

	if handler.dragButton != MouseButtonNone {
		t.Errorf("NewMouseHandler() dragButton = %v, want %v", handler.dragButton, MouseButtonNone)
	}
}

func TestMouseHandler_SetMode(t *testing.T) {
	handler := NewMouseHandler()

	handler.SetMode(MouseModeX10)

	if handler.GetMode() != MouseModeX10 {
		t.Errorf("SetMode() mode = %v, want %v", handler.GetMode(), MouseModeX10)
	}
}

func TestMouseHandler_buttonToX10Code(t *testing.T) {
	handler := NewMouseHandler()

	tests := []struct {
		button   MouseButton
		expected int
	}{
		{MouseButtonLeft, 0},
		{MouseButtonMiddle, 1},
		{MouseButtonRight, 2},
		{MouseButtonWheelUp, -1},
		{MouseButtonNone, -1},
	}

	for _, tt := range tests {
		result := handler.buttonToX10Code(tt.button)
		if result != tt.expected {
			t.Errorf("buttonToX10Code(%v) = %d, want %d", tt.button, result, tt.expected)
		}
	}
}

func TestMouseHandler_buttonToVT200Code(t *testing.T) {
	handler := NewMouseHandler()

	tests := []struct {
		button   MouseButton
		action   MouseAction
		expected int
	}{
		{MouseButtonLeft, MouseActionPress, 0},
		{MouseButtonMiddle, MouseActionPress, 1},
		{MouseButtonRight, MouseActionPress, 2},
		{MouseButtonLeft, MouseActionRelease, 3},
		{MouseButtonMiddle, MouseActionRelease, 3},
		{MouseButtonRight, MouseActionRelease, 3},
		{MouseButtonWheelUp, MouseActionPress, -1},
	}

	for _, tt := range tests {
		result := handler.buttonToVT200Code(tt.button, tt.action)
		if result != tt.expected {
			t.Errorf("buttonToVT200Code(%v, %v) = %d, want %d", tt.button, tt.action, result, tt.expected)
		}
	}
}

func TestMouseHandler_buttonToBtnEventCode(t *testing.T) {
	handler := NewMouseHandler()

	tests := []struct {
		button   MouseButton
		action   MouseAction
		expected int
	}{
		{MouseButtonLeft, MouseActionPress, 0},
		{MouseButtonLeft, MouseActionDrag, 32},
		{MouseButtonMiddle, MouseActionPress, 1},
		{MouseButtonMiddle, MouseActionDrag, 33},
		{MouseButtonRight, MouseActionPress, 2},
		{MouseButtonRight, MouseActionDrag, 34},
		{MouseButtonWheelUp, MouseActionPress, 64},
		{MouseButtonWheelDown, MouseActionPress, 65},
		{MouseButtonNone, MouseActionRelease, 3},
	}

	for _, tt := range tests {
		result := handler.buttonToBtnEventCode(tt.button, tt.action)
		if result != tt.expected {
			t.Errorf("buttonToBtnEventCode(%v, %v) = %d, want %d", tt.button, tt.action, result, tt.expected)
		}
	}
}

func TestMouseHandler_generateX10Sequence(t *testing.T) {
	handler := NewMouseHandler()

	// Test left button press
	event := MouseEvent{
		X:      10,
		Y:      5,
		Button: MouseButtonLeft,
		Action: MouseActionPress,
	}

	sequence := handler.generateX10Sequence(event)

	expected := []byte{0x1B, '[', 'M', 32, 43, 38} // ESC[M + (0+32) + (10+33) + (5+33)

	if len(sequence) != len(expected) {
		t.Errorf("generateX10Sequence() length = %d, want %d", len(sequence), len(expected))
	}

	for i, b := range expected {
		if i < len(sequence) && sequence[i] != b {
			t.Errorf("generateX10Sequence()[%d] = %d, want %d", i, sequence[i], b)
		}
	}

	// Test release (should return nil in X10 mode)
	event.Action = MouseActionRelease
	sequence = handler.generateX10Sequence(event)

	if sequence != nil {
		t.Error("generateX10Sequence() should return nil for release events")
	}
}

func TestMouseHandler_generateVT200Sequence(t *testing.T) {
	handler := NewMouseHandler()

	// Test left button press
	event := MouseEvent{
		X:      5,
		Y:      3,
		Button: MouseButtonLeft,
		Action: MouseActionPress,
	}

	sequence := handler.generateVT200Sequence(event)

	expected := []byte{0x1B, '[', 'M', 32, 38, 36} // ESC[M + (0+32) + (5+33) + (3+33)

	if len(sequence) != len(expected) {
		t.Errorf("generateVT200Sequence() press length = %d, want %d", len(sequence), len(expected))
	}

	for i, b := range expected {
		if i < len(sequence) && sequence[i] != b {
			t.Errorf("generateVT200Sequence() press [%d] = %d, want %d", i, sequence[i], b)
		}
	}

	// Test release
	event.Action = MouseActionRelease
	sequence = handler.generateVT200Sequence(event)

	expected = []byte{0x1B, '[', 'M', 35, 38, 36} // ESC[M + (3+32) + (5+33) + (3+33)

	if len(sequence) != len(expected) {
		t.Errorf("generateVT200Sequence() release length = %d, want %d", len(sequence), len(expected))
	}

	for i, b := range expected {
		if i < len(sequence) && sequence[i] != b {
			t.Errorf("generateVT200Sequence() release [%d] = %d, want %d", i, sequence[i], b)
		}
	}
}

func TestMouseHandler_ProcessTcellEvent_ModeOff(t *testing.T) {
	handler := NewMouseHandler()
	handler.SetMode(MouseModeOff)

	// Create a mock tcell event (we can't easily create a real one)
	// For this test, we'll test the mode check
	sequence := handler.ProcessTcellEvent(nil)

	if sequence != nil {
		t.Error("ProcessTcellEvent() should return nil when mode is off")
	}
}

func TestMouseEvent_Structure(t *testing.T) {
	event := MouseEvent{
		X:      10,
		Y:      20,
		Button: MouseButtonLeft,
		Action: MouseActionPress,
	}

	if event.X != 10 {
		t.Errorf("MouseEvent.X = %d, want 10", event.X)
	}

	if event.Y != 20 {
		t.Errorf("MouseEvent.Y = %d, want 20", event.Y)
	}

	if event.Button != MouseButtonLeft {
		t.Errorf("MouseEvent.Button = %v, want %v", event.Button, MouseButtonLeft)
	}

	if event.Action != MouseActionPress {
		t.Errorf("MouseEvent.Action = %v, want %v", event.Action, MouseActionPress)
	}
}

func TestNewKeyHandler(t *testing.T) {
	handler := NewKeyHandler()

	if handler == nil {
		t.Error("NewKeyHandler() returned nil")
	}

	if handler.applicationMode {
		t.Error("NewKeyHandler() applicationMode should be false")
	}

	if handler.cursorKeyMode {
		t.Error("NewKeyHandler() cursorKeyMode should be false")
	}
}

func TestKeyHandler_SetApplicationMode(t *testing.T) {
	handler := NewKeyHandler()

	handler.SetApplicationMode(true)

	if !handler.applicationMode {
		t.Error("SetApplicationMode(true) should set applicationMode to true")
	}

	handler.SetApplicationMode(false)

	if handler.applicationMode {
		t.Error("SetApplicationMode(false) should set applicationMode to false")
	}
}

func TestKeyHandler_SetCursorKeyMode(t *testing.T) {
	handler := NewKeyHandler()

	handler.SetCursorKeyMode(true)

	if !handler.cursorKeyMode {
		t.Error("SetCursorKeyMode(true) should set cursorKeyMode to true")
	}

	handler.SetCursorKeyMode(false)

	if handler.cursorKeyMode {
		t.Error("SetCursorKeyMode(false) should set cursorKeyMode to false")
	}
}

func TestKeyHandler_handleSpecialKey(t *testing.T) {
	handler := NewKeyHandler()

	tests := []struct {
		key      tcell.Key
		mods     tcell.ModMask
		expected []byte
	}{
		{tcell.KeyEnter, 0, []byte{0x0D}},
		{tcell.KeyTab, 0, []byte{0x09}},
		{tcell.KeyTab, tcell.ModShift, []byte{0x1B, '[', 'Z'}},
		{tcell.KeyBackspace, 0, []byte{0x7F}},
		{tcell.KeyBackspace, tcell.ModAlt, []byte{0x1B, 0x7F}},
		{tcell.KeyDelete, 0, []byte{0x1B, '[', '3', '~'}},
		{tcell.KeyInsert, 0, []byte{0x1B, '[', '2', '~'}},
		{tcell.KeyEscape, 0, []byte{0x1B}},
	}

	for _, tt := range tests {
		result := handler.handleSpecialKey(tt.key, tt.mods)

		if len(result) != len(tt.expected) {
			t.Errorf("handleSpecialKey(%v, %v) length = %d, want %d", tt.key, tt.mods, len(result), len(tt.expected))
			continue
		}

		for i, b := range tt.expected {
			if i < len(result) && result[i] != b {
				t.Errorf("handleSpecialKey(%v, %v)[%d] = %d, want %d", tt.key, tt.mods, i, result[i], b)
			}
		}
	}
}

func TestKeyHandler_handleFunctionKey(t *testing.T) {
	handler := NewKeyHandler()

	tests := []struct {
		key      tcell.Key
		expected []byte
	}{
		{tcell.KeyF1, []byte{0x1B, 'O', 'P'}},
		{tcell.KeyF2, []byte{0x1B, 'O', 'Q'}},
		{tcell.KeyF3, []byte{0x1B, 'O', 'R'}},
		{tcell.KeyF4, []byte{0x1B, 'O', 'S'}},
		{tcell.KeyF5, []byte{0x1B, '[', '1', '5', '~'}},
		{tcell.KeyF12, []byte{0x1B, '[', '2', '4', '~'}},
	}

	for _, tt := range tests {
		result := handler.handleFunctionKey(tt.key, 0)

		if len(result) != len(tt.expected) {
			t.Errorf("handleFunctionKey(%v) length = %d, want %d", tt.key, len(result), len(tt.expected))
			continue
		}

		for i, b := range tt.expected {
			if i < len(result) && result[i] != b {
				t.Errorf("handleFunctionKey(%v)[%d] = %d, want %d", tt.key, i, result[i], b)
			}
		}
	}
}

func TestKeyHandler_handleCursorKey(t *testing.T) {
	handler := NewKeyHandler()

	// Test normal mode
	tests := []struct {
		key      tcell.Key
		expected []byte
	}{
		{tcell.KeyUp, []byte{0x1B, '[', 'A'}},
		{tcell.KeyDown, []byte{0x1B, '[', 'B'}},
		{tcell.KeyRight, []byte{0x1B, '[', 'C'}},
		{tcell.KeyLeft, []byte{0x1B, '[', 'D'}},
	}

	for _, tt := range tests {
		result := handler.handleCursorKey(tt.key, 0)

		if len(result) != len(tt.expected) {
			t.Errorf("handleCursorKey(%v) normal mode length = %d, want %d", tt.key, len(result), len(tt.expected))
			continue
		}

		for i, b := range tt.expected {
			if i < len(result) && result[i] != b {
				t.Errorf("handleCursorKey(%v) normal mode [%d] = %d, want %d", tt.key, i, result[i], b)
			}
		}
	}

	// Test application mode
	handler.SetCursorKeyMode(true)

	appTests := []struct {
		key      tcell.Key
		expected []byte
	}{
		{tcell.KeyUp, []byte{0x1B, 'O', 'A'}},
		{tcell.KeyDown, []byte{0x1B, 'O', 'B'}},
		{tcell.KeyRight, []byte{0x1B, 'O', 'C'}},
		{tcell.KeyLeft, []byte{0x1B, 'O', 'D'}},
	}

	for _, tt := range appTests {
		result := handler.handleCursorKey(tt.key, 0)

		if len(result) != len(tt.expected) {
			t.Errorf("handleCursorKey(%v) app mode length = %d, want %d", tt.key, len(result), len(tt.expected))
			continue
		}

		for i, b := range tt.expected {
			if i < len(result) && result[i] != b {
				t.Errorf("handleCursorKey(%v) app mode [%d] = %d, want %d", tt.key, i, result[i], b)
			}
		}
	}
}

func TestKeyHandler_handleKeypadKey(t *testing.T) {
	handler := NewKeyHandler()

	// Test normal mode
	tests := []struct {
		key      tcell.Key
		expected []byte
	}{
		{tcell.KeyHome, []byte{0x1B, '[', 'H'}},
		{tcell.KeyEnd, []byte{0x1B, '[', 'F'}},
		{tcell.KeyPgUp, []byte{0x1B, '[', '5', '~'}},
		{tcell.KeyPgDn, []byte{0x1B, '[', '6', '~'}},
	}

	for _, tt := range tests {
		result := handler.handleKeypadKey(tt.key, 0)

		if len(result) != len(tt.expected) {
			t.Errorf("handleKeypadKey(%v) normal mode length = %d, want %d", tt.key, len(result), len(tt.expected))
			continue
		}

		for i, b := range tt.expected {
			if i < len(result) && result[i] != b {
				t.Errorf("handleKeypadKey(%v) normal mode [%d] = %d, want %d", tt.key, i, result[i], b)
			}
		}
	}

	// Test application mode
	handler.SetApplicationMode(true)

	appTests := []struct {
		key      tcell.Key
		expected []byte
	}{
		{tcell.KeyHome, []byte{0x1B, 'O', 'H'}},
		{tcell.KeyEnd, []byte{0x1B, 'O', 'F'}},
	}

	for _, tt := range appTests {
		result := handler.handleKeypadKey(tt.key, 0)

		if len(result) != len(tt.expected) {
			t.Errorf("handleKeypadKey(%v) app mode length = %d, want %d", tt.key, len(result), len(tt.expected))
			continue
		}

		for i, b := range tt.expected {
			if i < len(result) && result[i] != b {
				t.Errorf("handleKeypadKey(%v) app mode [%d] = %d, want %d", tt.key, i, result[i], b)
			}
		}
	}
}

func TestKeyHandler_handleControlChar(t *testing.T) {
	handler := NewKeyHandler()

	tests := []struct {
		key      tcell.Key
		char     rune
		mods     tcell.ModMask
		expected []byte
	}{
		{tcell.KeyRune, 'a', tcell.ModCtrl, []byte{0x01}},     // Ctrl+A
		{tcell.KeyRune, 'c', tcell.ModCtrl, []byte{0x03}},     // Ctrl+C
		{tcell.KeyRune, 'z', tcell.ModCtrl, []byte{0x1A}},     // Ctrl+Z
		{tcell.KeyRune, 'A', tcell.ModCtrl, []byte{0x01}},     // Ctrl+Shift+A
		{tcell.KeyRune, ' ', tcell.ModCtrl, []byte{0x00}},     // Ctrl+Space
		{tcell.KeyRune, 'a', tcell.ModAlt, []byte{0x1B, 'a'}}, // Alt+A
		{tcell.KeyRune, '1', tcell.ModAlt, []byte{0x1B, '1'}}, // Alt+1
	}

	for _, tt := range tests {
		result := handler.handleControlChar(tt.key, tt.char, tt.mods)

		if len(result) != len(tt.expected) {
			t.Errorf("handleControlChar(%v, %c, %v) length = %d, want %d", tt.key, tt.char, tt.mods, len(result), len(tt.expected))
			continue
		}

		for i, b := range tt.expected {
			if i < len(result) && result[i] != b {
				t.Errorf("handleControlChar(%v, %c, %v)[%d] = %d, want %d", tt.key, tt.char, tt.mods, i, result[i], b)
			}
		}
	}
}

func TestKeyHandler_handleRegularChar(t *testing.T) {
	handler := NewKeyHandler()

	tests := []struct {
		char     rune
		mods     tcell.ModMask
		expected []byte
	}{
		{'a', 0, []byte{'a'}},
		{'Z', 0, []byte{'Z'}},
		{'1', 0, []byte{'1'}},
		{'!', 0, []byte{'!'}},
		{'a', tcell.ModAlt, []byte{0x1B, 'a'}}, // Alt+a
		{'€', 0, []byte{0xE2, 0x82, 0xAC}},     // Euro symbol (UTF-8)
	}

	for _, tt := range tests {
		result := handler.handleRegularChar(tt.char, tt.mods)

		if len(result) != len(tt.expected) {
			t.Errorf("handleRegularChar(%c, %v) length = %d, want %d", tt.char, tt.mods, len(result), len(tt.expected))
			continue
		}

		for i, b := range tt.expected {
			if i < len(result) && result[i] != b {
				t.Errorf("handleRegularChar(%c, %v)[%d] = %d, want %d", tt.char, tt.mods, i, result[i], b)
			}
		}
	}
}

func TestKeyHandler_addModifiers(t *testing.T) {
	handler := NewKeyHandler()

	base := []byte{0x1B, '[', 'A'} // Up arrow

	tests := []struct {
		mods     tcell.ModMask
		expected []byte
	}{
		{0, []byte{0x1B, '[', 'A'}},                                             // No modifiers
		{tcell.ModShift, []byte{0x1B, '[', '1', ';', '2', 'A'}},                 // Shift
		{tcell.ModCtrl, []byte{0x1B, '[', '1', ';', '5', 'A'}},                  // Ctrl
		{tcell.ModShift | tcell.ModCtrl, []byte{0x1B, '[', '1', ';', '6', 'A'}}, // Shift+Ctrl
	}

	for _, tt := range tests {
		result := handler.addModifiers(base, tt.mods)

		if len(result) != len(tt.expected) {
			t.Errorf("addModifiers(%v, %v) length = %d, want %d", base, tt.mods, len(result), len(tt.expected))
			continue
		}

		for i, b := range tt.expected {
			if i < len(result) && result[i] != b {
				t.Errorf("addModifiers(%v, %v)[%d] = %d, want %d", base, tt.mods, i, result[i], b)
			}
		}
	}
}

func TestNewInputProcessor(t *testing.T) {
	terminal := NewTerminalEmulator(nil, nil, 80, 24)
	processor := NewInputProcessor(terminal)

	if processor == nil {
		t.Error("NewInputProcessor() returned nil")
	}

	if processor.keyHandler == nil {
		t.Error("NewInputProcessor() keyHandler should not be nil")
	}

	if processor.mouseHandler == nil {
		t.Error("NewInputProcessor() mouseHandler should not be nil")
	}

	if processor.terminal != terminal {
		t.Error("NewInputProcessor() terminal should be set correctly")
	}
}

func TestInputProcessor_SetModes(t *testing.T) {
	terminal := NewTerminalEmulator(nil, nil, 80, 24)
	processor := NewInputProcessor(terminal)

	// Test keypad application mode
	processor.SetKeypadApplicationMode(true)

	if !processor.keyHandler.applicationMode {
		t.Error("SetKeypadApplicationMode(true) should set application mode")
	}

	// Test cursor key application mode
	processor.SetCursorKeyApplicationMode(true)

	if !processor.keyHandler.cursorKeyMode {
		t.Error("SetCursorKeyApplicationMode(true) should set cursor key mode")
	}

	// Test mouse mode
	processor.SetMouseMode(MouseModeX10)

	if processor.mouseHandler.GetMode() != MouseModeX10 {
		t.Error("SetMouseMode() should set mouse mode")
	}
}

func TestInputProcessor_GetHandlers(t *testing.T) {
	terminal := NewTerminalEmulator(nil, nil, 80, 24)
	processor := NewInputProcessor(terminal)

	keyHandler := processor.GetKeyHandler()
	if keyHandler == nil {
		t.Error("GetKeyHandler() should return key handler")
	}

	mouseHandler := processor.GetMouseHandler()
	if mouseHandler == nil {
		t.Error("GetMouseHandler() should return mouse handler")
	}
}

func TestGetKeySequenceByName(t *testing.T) {
	tests := []struct {
		name     string
		expected []byte
	}{
		{"Ctrl+C", []byte{0x03}},
		{"Ctrl+D", []byte{0x04}},
		{"Ctrl+Z", []byte{0x1A}},
		{"Alt+Enter", []byte{0x1B, 0x0D}},
		{"Shift+Tab", []byte{0x1B, '[', 'Z'}},
		{"NonExistent", nil},
	}

	for _, tt := range tests {
		result := GetKeySequenceByName(tt.name)

		if tt.expected == nil {
			if result != nil {
				t.Errorf("GetKeySequenceByName(%s) should return nil for non-existent key", tt.name)
			}
			continue
		}

		if len(result) != len(tt.expected) {
			t.Errorf("GetKeySequenceByName(%s) length = %d, want %d", tt.name, len(result), len(tt.expected))
			continue
		}

		for i, b := range tt.expected {
			if i < len(result) && result[i] != b {
				t.Errorf("GetKeySequenceByName(%s)[%d] = %d, want %d", tt.name, i, result[i], b)
			}
		}
	}
}

func TestKeySequence_Structure(t *testing.T) {
	seq := KeySequence{
		Name:     "Test",
		Sequence: []byte{0x1B, 'A'},
		Mods:     tcell.ModCtrl,
	}

	if seq.Name != "Test" {
		t.Errorf("KeySequence.Name = %s, want Test", seq.Name)
	}

	if len(seq.Sequence) != 2 {
		t.Errorf("KeySequence.Sequence length = %d, want 2", len(seq.Sequence))
	}

	if seq.Mods != tcell.ModCtrl {
		t.Errorf("KeySequence.Mods = %v, want %v", seq.Mods, tcell.ModCtrl)
	}
}

func TestShortcutAction_String(t *testing.T) {
	tests := []struct {
		action   ShortcutAction
		expected string
	}{
		{ActionExit, "exit"},
		{ActionSave, "save"},
		{ActionClear, "clear"},
		{ActionCopy, "copy"},
		{ActionPaste, "paste"},
		{ActionFind, "find"},
		{ActionHelp, "help"},
		{ActionSettings, "settings"},
		{ActionConnect, "connect"},
		{ActionDisconnect, "disconnect"},
		{ActionCustom, "custom"},
		{ShortcutAction(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.action.String(); got != tt.expected {
				t.Errorf("ShortcutAction.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestShortcut_Matches(t *testing.T) {
	shortcut := &Shortcut{
		Name:    "test",
		Key:     tcell.KeyRune,
		Char:    'C',
		Mods:    tcell.ModCtrl,
		Enabled: true,
	}

	tests := []struct {
		name     string
		key      tcell.Key
		char     rune
		mods     tcell.ModMask
		expected bool
	}{
		{"exact match", tcell.KeyRune, 'C', tcell.ModCtrl, true},
		{"wrong char", tcell.KeyRune, 'c', tcell.ModCtrl, false},
		{"wrong mods", tcell.KeyRune, 'C', 0, false},
		{"wrong key", tcell.KeyF1, 'C', tcell.ModCtrl, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shortcut.Matches(tt.key, tt.char, tt.mods)
			if result != tt.expected {
				t.Errorf("Shortcut.Matches(%v, %c, %v) = %v, want %v", tt.key, tt.char, tt.mods, result, tt.expected)
			}
		})
	}

	// Test disabled shortcut
	shortcut.Enabled = false
	result := shortcut.Matches(tcell.KeyRune, 'C', tcell.ModCtrl)
	if result {
		t.Error("Disabled shortcut should not match")
	}
}

func TestShortcut_Execute(t *testing.T) {
	executed := false
	shortcut := &Shortcut{
		Name:    "test",
		Enabled: true,
		Handler: func() error {
			executed = true
			return nil
		},
	}

	err := shortcut.Execute()
	if err != nil {
		t.Errorf("Shortcut.Execute() failed: %v", err)
	}

	if !executed {
		t.Error("Shortcut handler was not executed")
	}

	// Test disabled shortcut
	shortcut.Enabled = false
	executed = false

	err = shortcut.Execute()
	if err == nil {
		t.Error("Disabled shortcut should return error")
	}

	if executed {
		t.Error("Disabled shortcut handler should not be executed")
	}

	// Test shortcut without handler
	shortcut.Enabled = true
	shortcut.Handler = nil

	err = shortcut.Execute()
	if err == nil {
		t.Error("Shortcut without handler should return error")
	}
}

func TestNewShortcutManager(t *testing.T) {
	sm := NewShortcutManager()

	if sm == nil {
		t.Error("NewShortcutManager() returned nil")
	}

	if !sm.IsEnabled() {
		t.Error("NewShortcutManager() should be enabled by default")
	}

	if len(sm.shortcuts) == 0 {
		t.Error("NewShortcutManager() should have default shortcuts")
	}

	// Check for some default shortcuts
	if sm.GetShortcut("exit") == nil {
		t.Error("Default 'exit' shortcut should exist")
	}

	if sm.GetShortcut("save") == nil {
		t.Error("Default 'save' shortcut should exist")
	}

	if sm.GetShortcut("help") == nil {
		t.Error("Default 'help' shortcut should exist")
	}
}

func TestShortcutManager_AddRemoveShortcut(t *testing.T) {
	sm := NewShortcutManager()

	shortcut := &Shortcut{
		Name:        "test",
		Key:         tcell.KeyF5,
		Action:      ActionCustom,
		Description: "Test shortcut",
		Enabled:     true,
	}

	// Add shortcut
	sm.AddShortcut(shortcut)

	retrieved := sm.GetShortcut("test")
	if retrieved == nil {
		t.Error("Added shortcut should be retrievable")
	}

	if retrieved.Name != "test" {
		t.Errorf("Retrieved shortcut name = %s, want test", retrieved.Name)
	}

	// Remove shortcut
	sm.RemoveShortcut("test")

	retrieved = sm.GetShortcut("test")
	if retrieved != nil {
		t.Error("Removed shortcut should not be retrievable")
	}
}

func TestShortcutManager_EnableDisableShortcut(t *testing.T) {
	sm := NewShortcutManager()

	shortcut := &Shortcut{
		Name:    "test",
		Key:     tcell.KeyF5,
		Enabled: true,
	}

	sm.AddShortcut(shortcut)

	// Disable shortcut
	sm.DisableShortcut("test")

	if shortcut.Enabled {
		t.Error("DisableShortcut() should disable the shortcut")
	}

	// Enable shortcut
	sm.EnableShortcut("test")

	if !shortcut.Enabled {
		t.Error("EnableShortcut() should enable the shortcut")
	}

	// Test non-existent shortcut
	sm.DisableShortcut("non-existent")
	sm.EnableShortcut("non-existent")
	// Should not panic
}

func TestShortcutManager_SetEnabled(t *testing.T) {
	sm := NewShortcutManager()

	sm.SetEnabled(false)

	if sm.IsEnabled() {
		t.Error("SetEnabled(false) should disable the manager")
	}

	sm.SetEnabled(true)

	if !sm.IsEnabled() {
		t.Error("SetEnabled(true) should enable the manager")
	}
}

func TestShortcutManager_ProcessKeyEvent(t *testing.T) {
	sm := NewShortcutManager()

	executed := false
	shortcut := &Shortcut{
		Name:    "test",
		Key:     tcell.KeyRune,
		Char:    'T',
		Mods:    tcell.ModCtrl,
		Enabled: true,
		Handler: func() error {
			executed = true
			return nil
		},
	}

	sm.AddShortcut(shortcut)

	// Test matching key event
	handled, err := sm.ProcessKeyEvent(tcell.KeyRune, 'T', tcell.ModCtrl)

	if err != nil {
		t.Errorf("ProcessKeyEvent() failed: %v", err)
	}

	if !handled {
		t.Error("ProcessKeyEvent() should return true for handled shortcut")
	}

	if !executed {
		t.Error("Shortcut handler should be executed")
	}

	// Test non-matching key event
	executed = false
	handled, err = sm.ProcessKeyEvent(tcell.KeyRune, 'X', tcell.ModCtrl)

	if err != nil {
		t.Errorf("ProcessKeyEvent() failed: %v", err)
	}

	if handled {
		t.Error("ProcessKeyEvent() should return false for non-matching key")
	}

	if executed {
		t.Error("Shortcut handler should not be executed for non-matching key")
	}

	// Test disabled manager
	sm.SetEnabled(false)
	executed = false

	handled, err = sm.ProcessKeyEvent(tcell.KeyRune, 'T', tcell.ModCtrl)

	if err != nil {
		t.Errorf("ProcessKeyEvent() failed: %v", err)
	}

	if handled {
		t.Error("Disabled manager should not handle shortcuts")
	}

	if executed {
		t.Error("Shortcut handler should not be executed when manager is disabled")
	}
}

func TestShortcutManager_SetShortcutHandler(t *testing.T) {
	sm := NewShortcutManager()

	executed := false
	handler := func() error {
		executed = true
		return nil
	}

	// Test setting handler for existing shortcut
	err := sm.SetShortcutHandler("exit", handler)
	if err != nil {
		t.Errorf("SetShortcutHandler() failed: %v", err)
	}

	// Execute the shortcut to test handler
	shortcut := sm.GetShortcut("exit")
	if shortcut != nil {
		shortcut.Execute()

		if !executed {
			t.Error("Custom handler should be executed")
		}
	}

	// Test setting handler for non-existent shortcut
	err = sm.SetShortcutHandler("non-existent", handler)
	if err == nil {
		t.Error("SetShortcutHandler() should fail for non-existent shortcut")
	}
}

func TestShortcutManager_CustomShortcut(t *testing.T) {
	sm := NewShortcutManager()

	executed := false
	handler := func() error {
		executed = true
		return nil
	}

	sm.CustomShortcut("custom-test", "Custom test shortcut", tcell.KeyF10, 0, tcell.ModAlt, handler)

	shortcut := sm.GetShortcut("custom-test")
	if shortcut == nil {
		t.Error("Custom shortcut should be added")
	}

	if shortcut.Action != ActionCustom {
		t.Error("Custom shortcut should have ActionCustom")
	}

	if shortcut.Description != "Custom test shortcut" {
		t.Errorf("Custom shortcut description = %s, want 'Custom test shortcut'", shortcut.Description)
	}

	// Test execution
	handled, err := sm.ProcessKeyEvent(tcell.KeyF10, 0, tcell.ModAlt)

	if err != nil {
		t.Errorf("ProcessKeyEvent() failed: %v", err)
	}

	if !handled {
		t.Error("Custom shortcut should be handled")
	}

	if !executed {
		t.Error("Custom shortcut handler should be executed")
	}
}

func TestShortcutManager_ListShortcuts(t *testing.T) {
	sm := NewShortcutManager()

	shortcuts := sm.ListShortcuts()

	if len(shortcuts) == 0 {
		t.Error("ListShortcuts() should return default shortcuts")
	}

	// Add a custom shortcut
	sm.CustomShortcut("test", "Test", tcell.KeyF5, 0, 0, func() error { return nil })

	newShortcuts := sm.ListShortcuts()

	if len(newShortcuts) != len(shortcuts)+1 {
		t.Error("ListShortcuts() should include custom shortcut")
	}
}

func TestShortcutManager_GetShortcutHelp(t *testing.T) {
	sm := NewShortcutManager()

	help := sm.GetShortcutHelp()

	if help == "" {
		t.Error("GetShortcutHelp() should return non-empty help text")
	}

	// Should contain some default shortcuts
	if !containsString(help, "Exit application") {
		t.Error("Help should contain exit shortcut description")
	}

	if !containsString(help, "Show help") {
		t.Error("Help should contain help shortcut description")
	}
}

func TestShortcut_ToConfig(t *testing.T) {
	shortcut := &Shortcut{
		Name:        "test",
		Key:         tcell.KeyRune,
		Char:        'C',
		Mods:        tcell.ModCtrl | tcell.ModShift,
		Action:      ActionSave,
		Description: "Test shortcut",
		Enabled:     true,
	}

	config := shortcut.ToConfig()

	if config.Name != "test" {
		t.Errorf("ToConfig() Name = %s, want test", config.Name)
	}

	if config.Char != "C" {
		t.Errorf("ToConfig() Char = %s, want C", config.Char)
	}

	if config.Action != "save" {
		t.Errorf("ToConfig() Action = %s, want save", config.Action)
	}

	if !config.Enabled {
		t.Error("ToConfig() Enabled should be true")
	}

	if len(config.Mods) != 2 {
		t.Errorf("ToConfig() Mods length = %d, want 2", len(config.Mods))
	}
}

func TestShortcutFromConfig(t *testing.T) {
	config := ShortcutConfig{
		Name:        "test",
		Char:        "C",
		Mods:        []string{"ctrl", "shift"},
		Action:      "save",
		Description: "Test shortcut",
		Enabled:     true,
	}

	shortcut, err := ShortcutFromConfig(config)
	if err != nil {
		t.Errorf("ShortcutFromConfig() failed: %v", err)
	}

	if shortcut.Name != "test" {
		t.Errorf("ShortcutFromConfig() Name = %s, want test", shortcut.Name)
	}

	if shortcut.Char != 'C' {
		t.Errorf("ShortcutFromConfig() Char = %c, want C", shortcut.Char)
	}

	if shortcut.Action != ActionSave {
		t.Errorf("ShortcutFromConfig() Action = %v, want %v", shortcut.Action, ActionSave)
	}

	if shortcut.Mods != (tcell.ModCtrl | tcell.ModShift) {
		t.Errorf("ShortcutFromConfig() Mods = %v, want %v", shortcut.Mods, tcell.ModCtrl|tcell.ModShift)
	}

	// Test invalid action
	config.Action = "invalid"
	_, err = ShortcutFromConfig(config)
	if err == nil {
		t.Error("ShortcutFromConfig() should fail with invalid action")
	}
}

func TestStringToKey(t *testing.T) {
	tests := []struct {
		keyStr   string
		expected tcell.Key
		wantErr  bool
	}{
		{"F1", tcell.KeyF1, false},
		{"Enter", tcell.KeyEnter, false},
		{"Tab", tcell.KeyTab, false},
		{"Up", tcell.KeyUp, false},
		{"Invalid", tcell.KeyRune, true},
	}

	for _, tt := range tests {
		result, err := stringToKey(tt.keyStr)

		if tt.wantErr {
			if err == nil {
				t.Errorf("stringToKey(%s) should return error", tt.keyStr)
			}
		} else {
			if err != nil {
				t.Errorf("stringToKey(%s) failed: %v", tt.keyStr, err)
			}

			if result != tt.expected {
				t.Errorf("stringToKey(%s) = %v, want %v", tt.keyStr, result, tt.expected)
			}
		}
	}
}

// Helper function for string containment check
func containsString(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					indexOfSubstringHelper(s, substr) >= 0)))
}

func indexOfSubstringHelper(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
