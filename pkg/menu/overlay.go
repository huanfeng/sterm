package menu

import (
	"github.com/gdamore/tcell/v2"
)

// OverlayManager manages screen overlays for menus
type OverlayManager struct {
	screen       tcell.Screen
	savedContent [][]SavedCell
	width        int
	height       int
}

// SavedCell represents a saved screen cell
type SavedCell struct {
	Char  rune
	Style tcell.Style
}

// NewOverlayManager creates a new overlay manager
func NewOverlayManager(screen tcell.Screen) *OverlayManager {
	width, height := screen.Size()
	return &OverlayManager{
		screen: screen,
		width:  width,
		height: height,
	}
}

// SaveScreen saves the current screen content
func (om *OverlayManager) SaveScreen() {
	om.width, om.height = om.screen.Size()
	om.savedContent = make([][]SavedCell, om.height)
	
	for y := 0; y < om.height; y++ {
		om.savedContent[y] = make([]SavedCell, om.width)
		for x := 0; x < om.width; x++ {
			mainc, _, style, _ := om.screen.GetContent(x, y)
			om.savedContent[y][x] = SavedCell{
				Char:  mainc,
				Style: style,
			}
		}
	}
}

// RestoreScreen restores the saved screen content
func (om *OverlayManager) RestoreScreen() {
	if om.savedContent == nil {
		return
	}
	
	for y := 0; y < len(om.savedContent); y++ {
		for x := 0; x < len(om.savedContent[y]); x++ {
			cell := om.savedContent[y][x]
			om.screen.SetContent(x, y, cell.Char, nil, cell.Style)
		}
	}
	
	om.screen.Show()
}

// Clear clears the saved content
func (om *OverlayManager) Clear() {
	om.savedContent = nil
}