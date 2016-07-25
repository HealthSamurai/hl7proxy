package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"strings"
)

var MESSAGE_SEPARATOR = fmt.Sprintf("%c%c", rune(0x1c), rune(0x0d))
var MESSAGE_HEADER = rune(0x0b)

func connectionHandler(conn net.Conn, out chan string) {
	defer close(out)
	var reader = bufio.NewReader(conn)
	var tail = ""

	for {
		var buff = make([]byte, 4096)

		bytesRead, err := reader.Read(buff)
		if err != nil {
			return
		}

		var messages = strings.Split(string(buff[:bytesRead]), MESSAGE_SEPARATOR)

		if len(messages) > 0 {
			messages[0] = tail + messages[0]

			// last message will be new tail
			tail = messages[len(messages)-1]
			messages = messages[:len(messages)-1]
		}

		for _, message := range messages {
			if rune(message[0]) != MESSAGE_HEADER {
				log.Printf("WARNING: No header in message!\n")
			} else {
				out <- message[1:]
			}
		}
	}
}

func messageSender(in <-chan string, out chan string, aidboxUrl string) {
	for msg := range in {
		log.Printf("\n%s", msg)
		out <- "ACK|||||||||\r"
	}
}

func ackSender(conn net.Conn, in <-chan string) {
	for ack := range in {
		log.Printf("Sending ACK: %s\n\n", ack)
		conn.Write([]byte(string(MESSAGE_HEADER)))
		conn.Write([]byte(ack))
		conn.Write([]byte(MESSAGE_SEPARATOR))
	}
}

func main() {
	port := flag.Int("port", 5000, "HL7 port to listen")
	aidboxUrl := flag.String("url", "", "URL of Aidbox HL7 handler")
	listenStr := fmt.Sprintf(":%d", *port)

	flag.Parse()

	psock, err := net.Listen("tcp", listenStr)
	log.Printf("Listening to %s\n", listenStr)

	if err != nil {
		return
	}

	for {
		conn, err := psock.Accept()
		if err != nil {
			return
		}

		msgChan := make(chan string)
		ackChan := make(chan string)

		go connectionHandler(conn, msgChan)
		go messageSender(msgChan, ackChan, *aidboxUrl)
		go ackSender(conn, ackChan)
	}
}
