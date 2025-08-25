package menu

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
)

// Menu represents a menu system
type Menu struct {
	items    []MenuItem
	selected int
	visible  bool
	screen   tcell.Screen
	x, y     int
	width    int
	height   int
	parent   *Menu
	title    string
	
	// Callbacks
	onClose func()
}

// MenuItem represents a single menu item
type MenuItem struct {
	Label     string
	Shortcut  string
	Action    func() error
	Submenu   *Menu
	Enabled   bool
	Separator bool
}

// NewMenu creates a new menu
func NewMenu(title string, screen tcell.Screen) *Menu {
	return &Menu{
		title:    title,
		screen:   screen,
		items:    make([]MenuItem, 0),
		selected: 0,
		visible:  false,
		width:    40,
		height:   10,
	}
}

// AddItem adds a menu item
func (m *Menu) AddItem(label, shortcut string, action func() error) {
	m.items = append(m.items, MenuItem{
		Label:    label,
		Shortcut: shortcut,
		Action:   action,
		Enabled:  true,
	})
	// Adjust menu dimensions
	m.updateDimensions()
}

// AddSeparator adds a separator line
func (m *Menu) AddSeparator() {
	m.items = append(m.items, MenuItem{
		Separator: true,
	})
}

// AddSubmenu adds a submenu item
func (m *Menu) AddSubmenu(label string, submenu *Menu) {
	submenu.parent = m
	m.items = append(m.items, MenuItem{
		Label:   label,
		Submenu: submenu,
		Enabled: true,
	})
	m.updateDimensions()
}

// Show displays the menu
func (m *Menu) Show() {
	m.visible = true
	// Center the menu on screen
	screenWidth, screenHeight := m.screen.Size()
	m.x = (screenWidth - m.width) / 2
	m.y = (screenHeight - m.height) / 2
	m.Draw()
}

// Hide hides the menu
func (m *Menu) Hide() {
	m.visible = false
	if m.onClose != nil {
		m.onClose()
	}
}

// IsVisible returns whether the menu is visible
func (m *Menu) IsVisible() bool {
	return m.visible
}

// Draw renders the menu on screen
func (m *Menu) Draw() {
	if !m.visible {
		return
	}

	style := tcell.StyleDefault.Background(tcell.ColorDarkBlue).Foreground(tcell.ColorWhite)
	selectedStyle := tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack)
	disabledStyle := tcell.StyleDefault.Background(tcell.ColorDarkBlue).Foreground(tcell.ColorGray)

	// Draw menu border and background
	m.drawBorder()

	// Draw title if present
	titleY := m.y + 1
	if m.title != "" {
		titleX := m.x + (m.width-len(m.title))/2
		m.drawText(titleX, titleY, m.title, style.Bold(true))
		titleY++
		// Draw separator under title
		for x := m.x + 1; x < m.x+m.width-1; x++ {
			m.screen.SetContent(x, titleY, '─', nil, style)
		}
		titleY++
	}

	// Draw menu items
	itemY := titleY
	for i, item := range m.items {
		if item.Separator {
			// Draw separator line
			for x := m.x + 1; x < m.x+m.width-1; x++ {
				m.screen.SetContent(x, itemY, '─', nil, style)
			}
		} else {
			// Determine style
			itemStyle := style
			if !item.Enabled {
				itemStyle = disabledStyle
			} else if i == m.selected {
				itemStyle = selectedStyle
			}

			// Clear line first
			for x := m.x + 1; x < m.x+m.width-1; x++ {
				m.screen.SetContent(x, itemY, ' ', nil, itemStyle)
			}

			// Draw item label
			label := item.Label
			if item.Submenu != nil {
				label = label + " >"
			}
			m.drawText(m.x+2, itemY, label, itemStyle)

			// Draw shortcut if present
			if item.Shortcut != "" && item.Submenu == nil {
				shortcutX := m.x + m.width - len(item.Shortcut) - 2
				m.drawText(shortcutX, itemY, item.Shortcut, itemStyle)
			}
		}
		itemY++
	}

	m.screen.Show()
}

// HandleKey processes keyboard input
func (m *Menu) HandleKey(ev *tcell.EventKey) bool {
	if !m.visible {
		return false
	}

	switch ev.Key() {
	case tcell.KeyEscape:
		m.Hide()
		return true

	case tcell.KeyUp:
		m.moveSelection(-1)
		m.Draw()
		return true

	case tcell.KeyDown:
		m.moveSelection(1)
		m.Draw()
		return true

	case tcell.KeyEnter:
		return m.activateSelected()

	case tcell.KeyRight:
		// Enter submenu if available
		if m.selected >= 0 && m.selected < len(m.items) {
			item := m.items[m.selected]
			if item.Submenu != nil && item.Enabled {
				item.Submenu.Show()
				return true
			}
		}
		return true

	case tcell.KeyLeft:
		// Return to parent menu
		if m.parent != nil {
			m.Hide()
			return true
		}
		return true
	}

	// Check for shortcut keys
	if ev.Key() == tcell.KeyRune {
		char := ev.Rune()
		for _, item := range m.items {
			if item.Shortcut != "" && !item.Separator && item.Enabled {
				// Simple shortcut matching (could be improved)
				if string(char) == item.Shortcut {
					if item.Action != nil {
						item.Action()
						m.Hide()
					}
					return true
				}
			}
		}
	}

	return false
}

// moveSelection moves the selection up or down
func (m *Menu) moveSelection(direction int) {
	newSelected := m.selected
	itemCount := len(m.items)

	for {
		newSelected += direction
		if newSelected < 0 {
			newSelected = itemCount - 1
		} else if newSelected >= itemCount {
			newSelected = 0
		}

		// Skip separators and disabled items
		if !m.items[newSelected].Separator && m.items[newSelected].Enabled {
			m.selected = newSelected
			break
		}

		// Prevent infinite loop if all items are disabled
		if newSelected == m.selected {
			break
		}
	}
}

// activateSelected activates the currently selected item
func (m *Menu) activateSelected() bool {
	if m.selected < 0 || m.selected >= len(m.items) {
		return false
	}

	item := m.items[m.selected]
	if !item.Enabled || item.Separator {
		return false
	}

	if item.Submenu != nil {
		item.Submenu.Show()
		return true
	}

	if item.Action != nil {
		err := item.Action()
		if err != nil {
			// Could show error in status bar
			fmt.Printf("Menu action error: %v\n", err)
		}
		m.Hide()
		return true
	}

	return false
}

// drawBorder draws the menu border
func (m *Menu) drawBorder() {
	style := tcell.StyleDefault.Background(tcell.ColorDarkBlue).Foreground(tcell.ColorWhite)

	// Top border
	m.screen.SetContent(m.x, m.y, '┌', nil, style)
	m.screen.SetContent(m.x+m.width-1, m.y, '┐', nil, style)
	for x := m.x + 1; x < m.x+m.width-1; x++ {
		m.screen.SetContent(x, m.y, '─', nil, style)
	}

	// Side borders and fill
	for y := m.y + 1; y < m.y+m.height-1; y++ {
		m.screen.SetContent(m.x, y, '│', nil, style)
		m.screen.SetContent(m.x+m.width-1, y, '│', nil, style)
		// Fill background
		for x := m.x + 1; x < m.x+m.width-1; x++ {
			m.screen.SetContent(x, y, ' ', nil, style)
		}
	}

	// Bottom border
	m.screen.SetContent(m.x, m.y+m.height-1, '└', nil, style)
	m.screen.SetContent(m.x+m.width-1, m.y+m.height-1, '┘', nil, style)
	for x := m.x + 1; x < m.x+m.width-1; x++ {
		m.screen.SetContent(x, m.y+m.height-1, '─', nil, style)
	}
}

// drawText draws text at the specified position
func (m *Menu) drawText(x, y int, text string, style tcell.Style) {
	for i, ch := range text {
		m.screen.SetContent(x+i, y, ch, nil, style)
	}
}

// updateDimensions updates menu dimensions based on items
func (m *Menu) updateDimensions() {
	maxWidth := len(m.title) + 4
	
	for _, item := range m.items {
		if !item.Separator {
			width := len(item.Label) + len(item.Shortcut) + 8
			if item.Submenu != nil {
				width += 2 // Space for submenu indicator
			}
			if width > maxWidth {
				maxWidth = width
			}
		}
	}
	
	m.width = maxWidth
	m.height = len(m.items) + 4 // Items + borders + title
	if m.title != "" {
		m.height += 2 // Title and separator
	}
}

// SetOnClose sets the callback for when menu closes
func (m *Menu) SetOnClose(callback func()) {
	m.onClose = callback
}

// EnableItem enables or disables a menu item
func (m *Menu) EnableItem(index int, enabled bool) {
	if index >= 0 && index < len(m.items) {
		m.items[index].Enabled = enabled
	}
}

// Clear removes all menu items
func (m *Menu) Clear() {
	m.items = []MenuItem{}
	m.selected = 0
	m.updateDimensions()
}