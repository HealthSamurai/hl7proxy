#!/usr/bin/env sh

GOOS=linux GOARCH=amd64 BINSUFFIX=-linux-amd64 make
GOOS=linux GOARCH=386 BINSUFFIX=-linux-386 make
GOOS=windows GOARCH=amd64 BINSUFFIX=-windows-amd64.exe make
GOOS=windows GOARCH=386 BINSUFFIX=-windows-386.exe make
GOOS=darwin GOARCH=amd64 BINSUFFIX=-darwin-amd64 make
GOOS=darwin GOARCH=arm64 BINSUFFIX=-darwin-arm64 make
