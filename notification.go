package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/jroimartin/gocui"
)

type NotificationViewController struct {
	*ViewController
}

func NewNotificationViewController(vc *ViewController) *NotificationViewController {
	return &NotificationViewController{
		ViewController: vc,
	}
}

func (n *NotificationViewController) Debug(title, message string) error {
	if err := n.vm.EnableView(n.viewName); err != nil {
		return err
	}

	str := fmt.Sprintf(
		"%s | %s | %s",
		title,
		time.Now().Format(time.RFC822),
		strings.TrimSpace(message),
	)

	n.g.Update(func(*gocui.Gui) error {
		_, err := color.New(color.FgYellow).Fprintln(n.ViewController, str)
		return err
	})

	return nil
}

func (n *NotificationViewController) Error(title, message string) error {
	if err := n.vm.EnableView(n.viewName); err != nil {
		return err
	}

	str := fmt.Sprintf(
		"%s | %s | %s",
		title,
		time.Now().Format(time.RFC822),
		strings.TrimSpace(message),
	)

	n.g.Update(func(*gocui.Gui) error {
		_, err := color.New(color.FgRed).Fprintln(n.ViewController, str)
		return err
	})

	return nil
}
