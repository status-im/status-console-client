package main

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/fatih/color"
)

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
