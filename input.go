package main

import (
	"bytes"
	"errors"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/jroimartin/gocui"

	"github.com/status-im/status-console-client/protocol/client"
	"github.com/status-im/status-console-client/protocol/v1"
)

const DefaultMultiplexerPrefix = "default"

type CmdHandler func([]byte) error

type InputMultiplexer struct {
	handlers map[string]CmdHandler // map string prefix to handler
}

func NewInputMultiplexer() *InputMultiplexer {
	return &InputMultiplexer{
		handlers: make(map[string]CmdHandler),
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

func (m *InputMultiplexer) AddHandler(prefix string, h CmdHandler) {
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

func chatAddCmdHandler(args []string) (c client.Chat, err error) {
	if len(args) == 1 {
		name := args[0]
		c = client.Chat{
			Name: name,
			Type: client.PublicChat,
		}
	} else if len(args) == 2 {
		c, err = client.CreateOneToOneChat(args[1], args[0])
	} else {
		err = errors.New("/chat: incorect arguments to add subcommand")
	}

	return
}

func ChatCmdFactory(c *ChatsViewController) CmdHandler {
	return func(b []byte) error {
		args := bytesToArgs(b)[1:] // remove first item, i.e. "/chat"

		log.Printf("handle /chat command: %s", b)

		switch args[0] {
		case "add":
			chat, err := chatAddCmdHandler(args[1:])
			if err != nil {
				return err
			}
			if err := c.Add(chat); err != nil {
				return err
			}
			// TODO: fix removing chats
			// case "remove":
			//	if len(args) == 2 {
			//		if err := c.Remove(args[1]); err != nil {
			//			return err
			//		}
			//		c.Refresh()
			//		return nil
			//	}
			//	return errors.New("/chat: incorect arguments to remove subcommand")
		}

		return nil
	}
}

func RequestCmdFactory(chat *ChatViewController) CmdHandler {
	return func(b []byte) error {
		args := bytesToArgs(b)[1:] // remove first item, i.e. "/request"

		log.Printf("handle /request command: %s", b)

		params := protocol.DefaultRequestOptions()

		switch len(args) {
		case 3:
			limit, err := strconv.Atoi(args[2])
			if err != nil {
				log.Printf("failed to parse Limit param: %v", err)
			} else {
				params.Limit = limit
			}
			fallthrough
		case 2:
			from, err := time.ParseDuration(args[1])
			if err != nil {
				log.Printf("failed to parse From param: %v", err)
			} else {
				params.From = time.Now().Add(-from).Unix()
			}
			fallthrough
		case 1:
			to, err := time.ParseDuration(args[0])
			if err != nil {
				log.Printf("failed to parse To param: %v", err)
			} else {
				params.To = time.Now().Add(-to).Unix()
			}
		}

		return chat.RequestMessages(params)
	}
}
