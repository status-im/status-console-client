package main

import (
	"bytes"
	"errors"
	"strings"

	"github.com/jroimartin/gocui"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol"
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

func chatAddCmdHandler(args []string) (chat protocol.Chat, err error) {
	if len(args) == 1 {
		name := args[0]
		chat = protocol.CreatePublicChat(name)
	} else if len(args) == 2 {
		publicKeyBytes, err := types.DecodeHex(args[0])
		if err != nil {
			return chat, err
		}
		publicKey, err := crypto.UnmarshalPubkey(publicKeyBytes)
		if err != nil {
			return chat, err
		}
		chat = protocol.CreateOneToOneChat(args[1], publicKey)
	} else {
		err = errors.New("/chat: incorrect arguments to add subcommand")
	}
	return
}

func ChatCmdFactory(chatsvc *ChatsViewController, chatvc *MessagesViewController) CmdHandler {
	return func(b []byte) error {
		args := bytesToArgs(b)[1:] // remove first item, i.e. "/chat"

		switch args[0] {
		case "add":
			chat, err := chatAddCmdHandler(args[1:])
			if err != nil {
				return err
			}
			if err := chatsvc.Add(chat); err != nil {
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
