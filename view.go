package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/jroimartin/gocui"
	"github.com/pkg/errors"
)

// Type of views.
const (
	ViewChats        = "chats"
	ViewChat         = "chat"
	ViewInput        = "input"
	ViewNotification = "notification"
)

// View describes a single terminal view.
type View struct {
	Name                   string
	Title                  string
	Enabled                bool
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
	g            *gocui.Gui
	views        []*View
	activeView   string
	previousView string
}

// NewViewManager returns a new view manager.
func NewViewManager(views []*View, g *gocui.Gui) *ViewManager {
	m := ViewManager{
		g:     g,
		views: views,
	}

	if len(views) > 0 {
		m.previousView = views[0].Name
		m.activeView = m.previousView
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
		if err := m.DeleteView(v.Name); err != nil {
			return errors.Wrap(err, "failed to set views")
		}
	}

	m.views = views

	if len(views) > 0 {
		m.previousView = views[0].Name
		m.activeView = m.previousView
	}

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
	if err := m.setKeybindings(globalKeybindingsViewName, bindings); err != nil {
		return errors.Wrap(err, "failed to set global keybindings")
	}
	return nil
}

// RawView returns a underlying low level view struct.
func (m *ViewManager) RawView(name string) (*gocui.View, error) {
	return m.g.View(name)
}

// ViewByName returns a View by name.
func (m *ViewManager) ViewByName(name string) *View {
	for _, v := range m.views {
		if v.Name == name {
			return v
		}
	}
	return nil
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

	for _, config := range m.views {
		if !config.Enabled {
			continue
		}

		x0, y0 := config.TopLeft(maxX, maxY)
		x1, y1 := config.BottomRight(maxX, maxY)

		v, err := m.g.SetView(config.Name, x0, y0, x1, y1)
		if err != nil && err != gocui.ErrUnknownView {
			return errors.Wrap(err, "failed to do layout")
		}
		// add bindings only once
		if err == gocui.ErrUnknownView {
			if err := m.setKeybindings(config.Name, config.Keybindings); err != nil {
				return errors.Wrap(err, "failed to do layout")
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

		if m.activeView == config.Name {
			if _, err := m.setCurrentView(config.Name); err != nil {
				return errors.Wrap(err, "failed to do layout")
			}

			m.g.Highlight = config.Highlight
			m.g.Cursor = config.Cursor
		}
	}

	return nil
}

// DeleteView removes a view and its bindings.
func (m *ViewManager) DeleteView(name string) error {
	if err := m.g.DeleteView(name); err != nil {
		return err
	}

	m.g.DeleteKeybindings(name)

	return nil
}

func (m *ViewManager) setCurrentView(name string) (*gocui.View, error) {
	return m.g.SetCurrentView(name)
}

// EnableView makes the view enable and selects it immediately.
func (m *ViewManager) EnableView(name string) error {
	log.Printf("[ViewManager::EnableView] enabling '%s'", name)

	view := m.ViewByName(name)
	if view == nil {
		return fmt.Errorf("failed to enable non-existing view '%s'", name)
	}

	view.Enabled = true

	if err := m.Layout(m.g); err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to enable view '%s'", name))
	}

	_, err := m.SelectView(name)
	return errors.Wrap(err, "failed to enable view")
}

// DisableView disables the view and selects the previous active one.
func (m *ViewManager) DisableView(name string) error {
	log.Printf("[ViewManager::DisableView] disabling '%s'", name)

	view := m.ViewByName(name)
	if view == nil {
		return fmt.Errorf("failed to disable non-existing view '%s'", name)
	}

	view.Enabled = false

	if err := m.Layout(m.g); err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to disable view '%s'", name))
	}

	_, err := m.SelectView(m.previousView)
	return errors.Wrap(err, "failed to disable view")
}

// SelectView selects a view to be active by name.
func (m *ViewManager) SelectView(name string) (*gocui.View, error) {
	log.Printf("[ViewManager::SelectView] selecting %s", name)

	if m.activeView == name {
		return m.RawView(name)
	}

	nextView := m.ViewByName(name)
	if nextView == nil {
		return nil, fmt.Errorf("failed to select non-existing '%s' view", name)
	}

	gocuiView, err := m.setCurrentView(nextView.Name)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed to select view '%s'", nextView.Name))
	}

	// deactivate callback
	currentView := m.ViewByName(m.activeView)
	if currentView == nil {
		return nil, fmt.Errorf("failed to view a view by name '%s'", m.activeView)
	} else if currentView.OnDeactivate != nil {
		currentView.OnDeactivate(currentView)
	}

	// activate callback
	if nextView.OnActivate != nil {
		nextView.OnActivate(nextView)
	}

	m.previousView = m.activeView
	m.activeView = name

	m.g.Cursor = nextView.Cursor
	m.g.Highlight = nextView.Highlight

	return gocuiView, nil
}

// NextView selects a next view clockwise.
func (m *ViewManager) NextView() error {
	nextActive := m.ViewIndex(m.activeView)
	for {
		nextActive = (nextActive + 1) % len(m.views)

		nextView := m.views[nextActive]
		if !nextView.Enabled {
			continue
		}

		_, err := m.SelectView(nextView.Name)
		return err
	}
}
