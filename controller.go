package main

import (
	"github.com/jroimartin/gocui"
)

// ViewController is a minimal view controller struct.
type ViewController struct {
	vm       *ViewManager
	g        *gocui.Gui
	viewName string
}

func (c *ViewController) view() (*gocui.View, error) {
	return c.vm.RawView(c.viewName)
}

// Write writes a payload to the view.
func (c *ViewController) Write(p []byte) (n int, err error) {
	v, err := c.view()
	if err != nil {
		return 0, err
	}
	return v.Write(p)
}

// Clear removes all content and movoes the cursor to the beginning.
func (c *ViewController) Clear() error {
	v, err := c.view()
	if err != nil {
		return err
	}

	v.Clear()

	if err := v.SetOrigin(0, 0); err != nil {
		return err
	}

	if err := v.SetCursor(0, 0); err != nil {
		return err
	}

	return nil
}
