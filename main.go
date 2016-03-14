package main

import (
	"bufio"
	"fmt"
	"github.com/gorilla/websocket"
	"gopkg.in/alecthomas/kingpin.v2"
	"net/http"
	"os"
	"os/signal"
	"strings"
)

var (
	url          = kingpin.Arg("url", "websocket url").Required().URL()
	extraHeaders = kingpin.Flag("header", "HTTP header").Short('H').Strings()
	oneOnly      = kingpin.Flag("one", "read only one message").Short('1').Bool()
)

func fail(msg string, o ...interface{}) {
	fmt.Fprintf(os.Stderr, msg, o...)
	os.Exit(1)
}

func main() {
	kingpin.UsageTemplate(kingpin.CompactUsageTemplate).Version("0.1.0")
	kingpin.Parse()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	headers := http.Header{}

	for _, header := range *extraHeaders {
		bits := strings.Split(header, ":")
		if len(bits) != 2 {
			fail("invalid header format: %v\n", header)
		}
		headers.Add(bits[0], bits[1])
	}

	conn, _, err := websocket.DefaultDialer.Dial((*url).String(), nil)
	if err != nil {
		fail("failed to connect to %q: %v\n", *url, err)
	}
	defer conn.Close()

	doneReading := make(chan bool)

	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure) {
					fail("unexpected read error %v\n", err)
				}
				break
			}
			fmt.Println(string(message))
			if *oneOnly {
				break
			}
		}
		doneReading <- true
	}()

	go func() {
		stdin := bufio.NewScanner(os.Stdin)
		for stdin.Scan() {
			conn.WriteMessage(websocket.TextMessage, []byte(stdin.Text()))
		}
	}()

	for {
		select {
		case <-doneReading:
			return
		case <-interrupt:
			err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure) {
					fail("unexpected close error %v\n", err)
				}
			}
			return
		}
	}
}
