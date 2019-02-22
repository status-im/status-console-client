package main

import (
	"bytes"
	"errors"
	"log"
	"strings"

	"github.com/jroimartin/gocui"
)

const DefaultMultiplexerPrefix = "default"

type InputMultiplexer struct {
	handlers map[string]func([]byte) error // map string prefix to handler
}

func NewInputMultiplexer() *InputMultiplexer {
	return &InputMultiplexer{
		handlers: make(map[string]func([]byte) error),
	}
}

func (m *InputMultiplexer) BindingHandler(g *gocui.Gui, v *gocui.View) error {
	if v == nil {
		return nil
	}

	var buf bytes.Buffer
	if err := EnterHandler(&buf)(g, v); err != nil {
		return err
	}

	inputStr := buf.String()
	inputBytes := bytes.TrimSpace(buf.Bytes())

	for prefix, h := range m.handlers {
		if strings.HasPrefix(inputStr, prefix) {
			return h(inputBytes)
		}
	}

	if h, ok := m.handlers[DefaultMultiplexerPrefix]; ok {
		return h(inputBytes)
	}

	return nil
}

func (m *InputMultiplexer) AddHandler(prefix string, h func([]byte) error) {
	m.handlers[prefix] = h
}

func bytesToArgs(b []byte) []string {
	args := bytes.Split(b, []byte(" "))
	argsStr := make([]string, len(args))
	for i, arg := range args {
		argsStr[i] = string(arg)
	}
	return argsStr
}

// ContactCmdHandler handles /contact command.
//
// Usage:
//   * /contact add public-chat-name
//   * /contact add 0xabc name
func ContactCmdHandler(b []byte) (c Contact, err error) {
	args := bytesToArgs(b)[1:] // remove first item, i.e. "/contact"

	log.Printf("ContactCmdHandler arguments: %s", args)

	switch args[0] {
	case "add":
		if len(args[1:]) == 1 {
			c = Contact{Name: args[1], Type: ContactPublicChat}
		} else if len(args[1:]) == 2 {
			c, err = NewContactWithPublicKey(args[2], args[1])
		} else {
			err = errors.New("/contact cmd: incorect arguments to add subcommand")
		}
		return
	default:
		err = errors.New("/contact cmd: invalid subcommand")
	}

	return
}

func RequestCmdHandler(b []byte) error {
	return nil
}
