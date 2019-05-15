package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/status-im/status-console-client/protocol/adapters"
	"github.com/status-im/status-console-client/protocol/client"
)

var (
	publicTopic  = flag.String("topic-public", "", "print public topic")
	privateTopic = flag.Bool("topic-private", false, "print private topic")
	output       = flag.String("o", "hex", "output format (hex, text, go)")
)

func main() {
	flag.Parse()

	log.Println("flags:", *publicTopic, *privateTopic, *output)

	if *publicTopic != "" {
		topic, err := adapters.ToTopic(*publicTopic)
		exitErr(err)

		printOutput(topic)
	} else if *privateTopic {
		topic, err := adapters.ToTopic(client.DefaultPrivateTopic())
		exitErr(err)

		printOutput(topic)
	}
}

func printOutput(data interface{}) {
	var err error

	if *output == "hex" {
		_, err = fmt.Fprintf(os.Stdout, "0x%x\n", data)
	} else if *output == "text" {
		_, err = fmt.Fprintf(os.Stdout, "%s\n", data)
	} else {
		_, err = fmt.Fprintf(os.Stdout, "%+v\n", data)
	}

	exitErr(err)
}

func exitErr(err error) {
	if err == nil {
		return
	}

	fmt.Println(err)
	os.Exit(1)
}
