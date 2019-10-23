package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	MESSAGE_SEPARATOR = "\x1c\x0d"
	MESSAGE_HEADER    = '\x0b'
)

var httpClient = &http.Client{
	Timeout: time.Second * 10,
}

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
		mshSegments[11],
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
	var buff = make([]byte, 4096)

	for {
		bytesRead, err := reader.Read(buff)

		if err != nil {
			log.Printf("Client disconnected: %s", conn.RemoteAddr())
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
				log.Printf("WARNING: No header in the message!\n")
			} else {
				out <- message[1:]
			}
		}
	}
}

var headerSplitRegexp = regexp.MustCompile(`:\s*`)

func quoteJson(s string) string {
	bytes, _ := json.Marshal(s)

	return string(bytes)
}

func MessageToAidboxSender(in <-chan string, out chan string, aidboxUrl string, headers []string, configId string) {
	defer close(out)

	for msg := range in {
		log.Printf("\n%s", FormatMessage(msg))

		out <- MakeAck(msg)

		var body = `
{ "resourceType": "Hl7v2Message",
  "status": "received", "src": ` + quoteJson(msg) + `,
  "config": { "resourceType": "Hl7v2Config", "id": ` + quoteJson(configId) + ` }}`

		req, err := http.NewRequest("POST", aidboxUrl+"/Hl7v2Message", strings.NewReader(body))

		if err != nil {
			log.Printf("Unable to create new http request: %s\n", err)
			continue
		}

		req.Header.Add("Content-Type", "application/json")

		for _, header := range headers {
			parts := headerSplitRegexp.Split(header, 2)
			req.Header.Add(parts[0], parts[1])
		}

		response, err := httpClient.Do(req)

		if err != nil {
			log.Printf("Error during HTTP request: %s\n", err)
			continue
		}

		if response.StatusCode < 200 || response.StatusCode > 299 {
			var body []byte

			if response.Body != nil {
				defer response.Body.Close()
				body, err = ioutil.ReadAll(response.Body)

				if err != nil {
					log.Printf("Cannot read response body: %s\n", err)
				}
			}

			log.Printf("Received non-200 response: %v\n%s\n", response, body)
			continue
		} else {
			log.Printf("Sent to Aidbox: %s\n", response.Status)
		}
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

type flagsStringsArray []string

func (i *flagsStringsArray) String() string {
	var s = "(todo: correct string representation) "
	for _, f := range *i {
		s = s + f
	}

	return s
}

func (i *flagsStringsArray) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func main() {
	port := flag.Int("port", 5000, "HL7 port to listen")
	aidboxUrl := flag.String("url", "", "Aidbox URL to send messages to (i.e. 'https://foo.aidbox.app/')")
	configId := flag.String("config", "", "An ID of existing Hl7v2Config resource")

	var headers flagsStringsArray = make(flagsStringsArray, 0)
	flag.Var(&headers, "header", "Additional HTTP headers in format 'Header: value'")

	listenStr := fmt.Sprintf(":%d", *port)

	flag.Parse()

	psock, err := net.Listen("tcp", listenStr)
	log.Printf("Listening to %s\n", listenStr)

	if err != nil {
		log.Printf("Unable to open port: %s\n", err)
		return
	}

	for {
		conn, err := psock.Accept()
		log.Printf("Received connection from %s\n", conn.RemoteAddr())

		if err != nil {
			log.Printf("Unable to accept connection: %s\n", err)
			return
		}

		msgChan := make(chan string)
		ackChan := make(chan string)

		go ConnectionHandler(conn, msgChan)
		go MessageToAidboxSender(msgChan, ackChan, *aidboxUrl, headers, *configId)
		go AckSender(conn, ackChan)
	}
}
