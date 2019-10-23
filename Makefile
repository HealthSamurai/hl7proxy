GOFILES=$(wildcard *.go)

hl7proxy: $(GOFILES)
	go build -o "bin/hl7proxy${BINSUFFIX}" .
