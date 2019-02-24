package main

import (
	"io"

	"github.com/jroimartin/gocui"
)

// GocuiHandler is a key binding handler.
type GocuiHandler func(*gocui.Gui, *gocui.View) error

// Binding describes a binding.
type Binding struct {
	Key     gocui.Key
	Mod     gocui.Modifier
	Handler GocuiHandler
}

// QuitHandler handles quit.
func QuitHandler(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

// NextViewHandler handles switching between views.
func NextViewHandler(m *ViewManager) GocuiHandler {
	return func(g *gocui.Gui, v *gocui.View) error {
		return m.NextView()
	}
}

// CursorDownHandler handles moving cursor one line down.
func CursorDownHandler(g *gocui.Gui, v *gocui.View) error {
	if v != nil {
		cx, cy := v.Cursor()
		if err := v.SetCursor(cx, cy+1); err != nil {
			ox, oy := v.Origin()
			if err := v.SetOrigin(ox, oy+1); err != nil {
				return err
			}
		}
	}
	return nil
}

// CursorUpHandler handles moving cursor one line up.
func CursorUpHandler(g *gocui.Gui, v *gocui.View) error {
	if v != nil {
		ox, oy := v.Origin()
		cx, cy := v.Cursor()
		if err := v.SetCursor(cx, cy-1); err != nil && oy > 0 {
			if err := v.SetOrigin(ox, oy-1); err != nil {
				return err
			}
		}
	}
	return nil
}

// EnterHandler handles striking Enter.
// To be as general as possible, it accepts a writer
// which gets the content of the current line.
func EnterHandler(w io.Writer) GocuiHandler {
	return func(g *gocui.Gui, v *gocui.View) error {
		if v == nil {
			return nil
		}

		if _, err := io.Copy(w, v); err != nil {
			return err
		}

		// Clear and move cursor at the beginning.
		v.Clear()
		if err := v.SetOrigin(0, 0); err != nil {
			return err
		}
		if err := v.SetCursor(0, 0); err != nil {
			return err
		}

		return nil
	}
}

// MoveToNewLineHandler allows to enter multiline text.
func MoveToNewLineHandler(g *gocui.Gui, v *gocui.View) error {
	if v == nil {
		return nil
	}

	v.MoveCursor(0, 1, true)
	_, cy := v.Cursor()
	if err := v.SetCursor(0, cy); err != nil {
		return err
	}

	return nil
}

// GetLineHandler passes the concent of the current line
// to the provided callback.
func GetLineHandler(cb func(int, string) error) GocuiHandler {
	return func(g *gocui.Gui, v *gocui.View) error {
		_, cy := v.Cursor()
		line, err := v.Line(cy)
		if err != nil {
			return err
		}
		return cb(cy, line)
	}
}
