package main

import (
	"context"
	"log"
	"strings"
	"time"

	ext "github.com/johnjonesbwai/go-streams/extension"
	"github.com/johnjonesbwai/go-streams/flow"
)

// Test producer: nc -u 127.0.0.1 3434
// Test consumer: nc -u -l 3535
func main() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	source, err := ext.NewNetSource(ctx, ext.UDP, "127.0.0.1:3434")
	if err != nil {
		log.Fatal(err)
	}

	toUpperMapFlow := flow.NewMap(toUpper, 1)
	sink, err := ext.NewNetSink(ext.UDP, "127.0.0.1:3535")
	if err != nil {
		log.Fatal(err)
	}

	source.
		Via(toUpperMapFlow).
		To(sink)
}

var toUpper = func(msg string) string {
	log.Printf("Got: %s", msg)
	return strings.ToUpper(msg)
}
