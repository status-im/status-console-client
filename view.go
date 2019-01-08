package main

import (
	"errors"
	"strings"

	"github.com/jroimartin/gocui"
)

// Type of views.
const (
	ViewContacts = "contacts"
	ViewChat     = "chat"
	ViewInput    = "input"
)

// View describes a single terminal view.
type View struct {
	Name                   string
	Title                  string
	TopLeft                func(int, int) (int, int)
	BottomRight            func(int, int) (int, int)
	Autoscroll             bool
	Cursor                 bool
	Editable               bool
	Wrap                   bool
	Highlight              bool
	SelBgColor, SelFgColor gocui.Attribute

	Keybindings []Binding

	OnActivate   func(*View)
	OnDeactivate func(*View)
}

// ViewManager manages a set of views.
type ViewManager struct {
	g      *gocui.Gui
	views  []*View
	active int
}

// NewViewManager returns a new view manager.
func NewViewManager(views []*View, g *gocui.Gui) *ViewManager {
	m := ViewManager{
		g:     g,
		views: views,
	}

	m.g.Highlight = true
	m.g.Cursor = true
	m.g.SelFgColor = gocui.ColorGreen

	m.g.SetManagerFunc(m.Layout)

	return &m
}

// SetViews replaces the existing views with the new ones.
func (m *ViewManager) SetViews(views []*View) error {
	for _, v := range m.views {
		if err := m.g.DeleteView(v.Name); err != nil {
			return err
		}
		m.g.DeleteKeybindings(v.Name)
	}

	m.views = views

	return nil
}

func (m *ViewManager) setKeybindings(viewName string, bindings []Binding) error {
	for _, b := range bindings {
		if err := m.g.SetKeybinding(viewName, b.Key, b.Mod, b.Handler); err != nil {
			return err
		}
	}
	return nil
}

// SetGlobalKeybindings sets a global key bindings which work from each view.
func (m *ViewManager) SetGlobalKeybindings(bindings []Binding) error {
	globalKeybindingsViewName := ""

	m.g.DeleteKeybindings(globalKeybindingsViewName)
	return m.setKeybindings(globalKeybindingsViewName, bindings)
}

// View returns a underlying low level view struct.
func (m *ViewManager) View(name string) (*gocui.View, error) {
	return m.g.View(name)
}

// ViewIndex returns an index of a view by name.
func (m *ViewManager) ViewIndex(name string) int {
	for i, v := range m.views {
		if v.Name == name {
			return i
		}
	}
	return -1
}

// Layout puts the views in the terminal window.
func (m *ViewManager) Layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	for idx, config := range m.views {
		x0, y0 := config.TopLeft(maxX, maxY)
		x1, y1 := config.BottomRight(maxX, maxY)

		v, err := m.g.SetView(config.Name, x0, y0, x1, y1)
		if err != nil && err != gocui.ErrUnknownView {
			return err
		}
		// add bindings only once
		if err == gocui.ErrUnknownView {
			if err := m.setKeybindings(config.Name, config.Keybindings); err != nil {
				return err
			}
		}

		if config.Title != "" {
			v.Title = config.Title
		} else {
			v.Title = strings.ToUpper(config.Name)
		}
		v.Autoscroll = config.Autoscroll
		v.Editable = config.Editable
		v.Wrap = config.Wrap
		v.Highlight = config.Highlight
		v.SelFgColor = config.SelFgColor
		v.SelBgColor = config.SelBgColor

		if m.active == idx {
			if _, err := m.setCurrentView(config.Name); err != nil {
				return err
			}

			m.g.Highlight = config.Highlight
			m.g.Cursor = config.Cursor
		}
	}

	return nil
}

func (m *ViewManager) setCurrentView(name string) (*gocui.View, error) {
	return m.g.SetCurrentView(name)
}

// SelectView selects a view to be active by name.
func (m *ViewManager) SelectView(name string) (*gocui.View, error) {
	idx := m.ViewIndex(name)
	if idx == -1 {
		return nil, errors.New("view does not exist")
	}

	nextView := m.views[idx]

	gocuiView, err := m.setCurrentView(nextView.Name)
	if err != nil {
		return nil, err
	}

	// deactivate callback
	v := m.views[m.active]
	if v.OnDeactivate != nil {
		v.OnDeactivate(v)
	}

	m.active = idx

	// activate callback
	v = m.views[m.active]
	if v.OnActivate != nil {
		v.OnActivate(v)
	}

	m.g.Cursor = nextView.Cursor
	m.g.Highlight = nextView.Highlight

	return gocuiView, nil
}

// NextView selects a next view clockwise.
func (m *ViewManager) NextView() error {
	nextActive := (m.active + 1) % len(m.views)
	nextView := m.views[nextActive]
	_, err := m.SelectView(nextView.Name)
	return err
}
