package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

const (
	MESSAGE_SEPARATOR = "\x1c\x0d"
	MESSAGE_HEADER    = '\x0b'
)

func FormatMessage(msg string) string {
	return strings.Replace(msg, "\r", "\n", -1)
}

func HL7TS(t time.Time) string {
	return fmt.Sprintf(
		"%d%d%d%d%d%d",
		t.Year(),
		t.Month(),
		t.Day(),
		t.Hour(),
		t.Minute(),
		t.Second())
}

func MakeAck(msg string) string {
	msh := msg[0:strings.Index(msg, "\r")]
	mshSegments := strings.Split(msh, "|")

	ackMsh := []string{
		"MSH",
		"^~\\&",
		mshSegments[4],
		mshSegments[5],
		mshSegments[2],
		mshSegments[3],
		HL7TS(time.Now()),
		"",
		"ACK",
		"",
		mshSegments[10],
		"2.4",
	}

	ackMsa := []string{
		"MSA",
		"AA",
		mshSegments[9],
		"",
		"",
		"",
		"",
	}

	return strings.Join(ackMsh, "|") + "\r" + strings.Join(ackMsa, "|") + "\r"
}

func ConnectionHandler(conn net.Conn, out chan string) {
	defer close(out)
	var reader = bufio.NewReader(conn)
	var tail = ""

	for {
		var buff = make([]byte, 4096)

		bytesRead, err := reader.Read(buff)
		if err != nil {
			fmt.Printf("Error: %v", err)
			// return
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

func MessageSender(in <-chan string, out chan string, aidboxUrl string) {
	for msg := range in {
		log.Printf("\n%s", FormatMessage(msg))
		out <- MakeAck(msg)
	}
}

func AckSender(conn net.Conn, in <-chan string) {
	for ack := range in {
		log.Printf("Sending ACK: %s\n\n", FormatMessage(ack))
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
		log.Printf("Received connection from %s", conn.RemoteAddr())

		if err != nil {
			return
		}

		msgChan := make(chan string)
		ackChan := make(chan string)

		go ConnectionHandler(conn, msgChan)
		go MessageSender(msgChan, ackChan, *aidboxUrl)
		go AckSender(conn, ackChan)
	}
}
