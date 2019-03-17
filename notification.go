package main

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/fatih/color"
)

// TODO: it's not enough. The gocui view should be updated,
// i.e. wrapped in a callback.
type Notifications struct {
	writer io.Writer
}

func (n *Notifications) Debug(title, message string) error {
	str := fmt.Sprintf(
		"%s | %s | %s",
		title,
		time.Now().Format(time.RFC822),
		strings.TrimSpace(message),
	)
	_, err := color.New(color.FgYellow).Fprintln(n.writer, str)
	return err
}

func (n *Notifications) Error(title, message string) error {
	str := fmt.Sprintf(
		"%s | %s | %s",
		title,
		time.Now().Format(time.RFC822),
		strings.TrimSpace(message),
	)
	_, err := color.New(color.FgRed).Fprintln(n.writer, str)
	return err
}
