package main

import (
	"io"
	"log"
	"strings"

	"github.com/jroimartin/gocui"
	"github.com/pkg/errors"
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
		if cy == 0 && oy > 0 {
			if err := v.SetOrigin(ox, oy-1); err != nil {
				return err
			}
		}
		if cy > 0 {
			if err := v.SetCursor(cx, cy-1); err != nil {
				return err
			}
		}
	}
	return nil
}

func HomeHandler(g *gocui.Gui, v *gocui.View) error {
	log.Printf("[HomeHandler]")

	if v != nil {
		if err := v.SetCursor(0, 0); err != nil {
			return errors.Wrap(err, "invalid cursor position")
		}
		if err := v.SetOrigin(0, 0); err != nil {
			return errors.Wrap(err, "invalid origin position")
		}
	}
	return nil
}

func EndHandler(g *gocui.Gui, v *gocui.View) error {
	log.Printf("[EndHandler]")

	if v != nil {
		lines := strings.Count(v.ViewBuffer(), "\n")
		_, sy := v.Size()

		log.Printf("[EndHandler] lines=%d sy=%d", lines, sy)

		if lines < sy {
			if err := v.SetOrigin(0, 0); err != nil {
				return errors.Wrap(err, "invalid origin position")
			}
			if err := v.SetCursor(0, lines+1); err != nil {
				return errors.Wrap(err, "invalid cursor position")
			}
		} else {
			if err := v.SetOrigin(0, lines-sy-1); err != nil {
				return errors.Wrap(err, "invalid origin position")
			}
			if err := v.SetCursor(0, sy-1); err != nil {
				return errors.Wrap(err, "invalid cursor position")
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
