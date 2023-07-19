package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	START_BLOCK_CHAR = '\x0b'
	END_BLOCK_CHAR = '\x1c'
	CR_CHAR = '\x0d'
)

var httpClient = &http.Client{
	Timeout: time.Second * 10,
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

func makeAckAndIdentity(msg string) (string, string, error) {
	if !strings.HasPrefix(msg, "MSH") {
		return "", "", fmt.Errorf("Ivalid message, message should start with 'MSH'")
	}

	mshrbound := strings.IndexByte(msg, '\n')
	if mshrbound == -1 {
		mshrbound = strings.IndexByte(msg, '\r')
	}
	if mshrbound == -1 {
		return "", "", fmt.Errorf("Ivalid message, can't detect end of MSH segment, no '\n' or '\r' byte found")
	}

	msh := msg[0:mshrbound]
	mshSegments := strings.Split(msh, "|")

	if len(mshSegments) < 12 {
		return "", "", fmt.Errorf("Ivalid message, 'MSH' segment doesn't has enough fields")
	}

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

	identity := fmt.Sprintf("type: %s, ctlid: %s, from: %s, to: %s", mshSegments[8], mshSegments[9], mshSegments[3],  mshSegments[5])

	return strings.Join(ackMsh, "|") + "\r" + strings.Join(ackMsa, "|") + "\r", identity, nil
}


func ScanHL7Msgs(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, END_BLOCK_CHAR); i >= 0 && len(data) >= (i + 2) {
		// We have a full END_BLOCK-terminated msg.
		if data[0] != START_BLOCK_CHAR {
			return 0, nil, fmt.Errorf("HL7 content should be preffixed with 'Start Block character (1 byte). ASCII <VT>, i.e., <0x0B>'.")
		} else if data[i + 1] != CR_CHAR {
			return 0, nil, fmt.Errorf("HL7 content should be suffixed with 'End Block character (1 byte). ASCII <FS>, i.e., <0x1C>' and 'Carriage Return (1 byte). ASCII <CR> character, i.e., <0x0D>'.")
		}
		return i + 2, data[1:i], nil
	}
	if atEOF {
		return 0, nil, fmt.Errorf("Unexpected EOF, HL7 content should be enclosed by special characters to form a Block. The Block format is as follows: <SB>dddd<EB><CR>")
	}
	// Request more data.
	return 0, nil, nil
}

func ConnectionHandler(conn net.Conn, opts options) {
	defer conn.Close()
	defer log.Printf("INFO: Client %s disconnected", conn.RemoteAddr())
	var scanner = bufio.NewScanner(conn)
	scanner.Split(ScanHL7Msgs)

	for scanner.Scan() {
		hl7msg := scanner.Text()
		log.Printf("INFO: New message received. Length: %d bytes", len(hl7msg))

		var ackmsg, identity, err = makeAckAndIdentity(hl7msg)
		if err != nil {
			log.Printf("ERROR: Message processing fail: %s", err)
			m := 50
			if m >= len(hl7msg) {
				m = len(hl7msg)
			}
			log.Printf("INFO: Message: %s...", hl7msg[0:m])
			return
		}
		log.Printf("INFO: Message identity: %s", identity)

		// Remove null characters from a string if present
		hl7msgCln := strings.ReplaceAll(hl7msg, "\u0000", "")
		if hl7msg != hl7msgCln {
			log.Printf("WARN: Null character detected and replaced in message")
		}
		err = deliverMessageToAidbox(hl7msgCln, opts)
		if err != nil {
			log.Printf("Error: Unable to deliver message to aidbox: %s", err)
			return
		}
		err = ack(conn, ackmsg)
		if err != nil {
			log.Printf("Error: Unable to send ACK: %s", err)
			return
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("ERROR: Invalid data received: %s", err)
	}
}

var headerSplitRegexp = regexp.MustCompile(`:\s*`)

type message struct {
    Name string
    Age int
    Active bool
    lastLoginAt string
}

type Hl7v2Message struct {
    ResourceType string    `json:"resourceType"`
    Status       string    `json:"status"`
    Src          string    `json:"src"`
    Config       Hl7v2Config `json:"config"`
}

type Hl7v2Config struct {
    ResourceType string `json:"resourceType"`
    ID           string `json:"id"`
}

func deliverMessageToAidbox(hl7msg string, opts options) error {
	msg := &Hl7v2Message{
		ResourceType: "Hl7v2Message",
		Status:       "received",
		Src:          hl7msg,
		Config: Hl7v2Config{
			ResourceType: "Hl7v2Config",
			ID:           opts.configId,
		},
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("unable to serialize request body: %s", err)
	}

	req, err := http.NewRequest("POST", opts.aidboxUrl+"/Hl7v2Message", bytes.NewReader(body))

	if err != nil {
		return fmt.Errorf("unable to create http request: %s\n", err)
	}

	req.Header.Add("Content-Type", "application/json")

	for _, header := range opts.headers {
		parts := headerSplitRegexp.Split(header, 2)
		req.Header.Add(parts[0], parts[1])
	}

	response, err := httpClient.Do(req)

	if err != nil {
		return fmt.Errorf("error during HTTP request: %s\n", err)
	}

	if response.StatusCode < 200 || response.StatusCode > 299 {
		var body []byte
		var scoderr = fmt.Sprintf("response code: %d", response.StatusCode)

		if response.Body != nil {
			defer response.Body.Close()
			body, err = ioutil.ReadAll(response.Body)

			if err != nil {
				return fmt.Errorf("%s, cannot read response body: %s", scoderr, err)
			}
		}
		return fmt.Errorf("%s, aidbox response: \"%s\"", scoderr, body)
	} else {
		duration := response.Header.Get("x-duration")
		log.Printf("INFO: Message delivered to Aidbox: %s (%sms)", response.Status, duration)
	}
	return nil
}

func ack(conn net.Conn, ack string) error {
	log.Printf("INFO: Sending ACK")
	_, err := fmt.Fprint(conn, string(START_BLOCK_CHAR), ack, string(END_BLOCK_CHAR), string(CR_CHAR))
	return err;
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

type options struct {
	port int;
	host string;
	aidboxUrl string;
	configId string;
	headers flagsStringsArray;
}

func main() {

	var opts options;
	opts.headers = make(flagsStringsArray, 0);

	flag.IntVar(&opts.port, "port", 5000, "HL7 port to listen")
	flag.StringVar(&opts.aidboxUrl, "url", "", "Aidbox URL to send messages to (i.e. 'https://foo.aidbox.app/') (required)")
	flag.StringVar(&opts.configId, "config", "", "An ID of existing Hl7v2Config resource (required)")
	flag.StringVar(&opts.host, "host", "", "host to listen")

	flag.Var(&opts.headers, "header", "Additional HTTP headers in format 'Header: value'")
	
	flag.Parse()

	listenStr := fmt.Sprintf("%s:%d", opts.host, opts.port)

	if opts.configId == "" {
		fmt.Printf("No required -config flag is provided. Usage:\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if opts.aidboxUrl == "" {
		fmt.Printf("No required -url flag is provided. Usage:\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	psock, err := net.Listen("tcp", listenStr)
	log.Printf("INFO: Listening to %s", listenStr)

	if err != nil {
		log.Printf("ERROR: Unable to open port: %s", err)
		return
	}

	for {
		conn, err := psock.Accept()
		log.Printf("INFO: New connection from %s", conn.RemoteAddr())

		if err != nil {
			log.Printf("Unable to accept connection: %s", err)
			return
		}

		go ConnectionHandler(conn, opts)
	}
}
