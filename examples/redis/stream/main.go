package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/johnjonesbwai/go-streams/flow"
	rs "github.com/johnjonesbwai/go-streams/redis"
	"github.com/redis/go-redis/v9"
)

// XADD stream1 * key1 a key2 b key3 c
// XLEN stream2
// XREAD COUNT 1 BLOCK 100 STREAMS stream2 0
func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	config := &redis.Options{
		Addr:     "localhost:6379", // use default Addr
		Password: "",               // no password set
		DB:       0,                // use default DB
	}

	redisClient := redis.NewClient(config)

	readGroupArgs := &redis.XReadGroupArgs{
		Group:    "group1",
		Consumer: "consumer1",
		Streams:  []string{"stream1", ">"},
	}
	// groupCreateArgs := &rs.XGroupCreateArgs{
	// 	Stream:   "stream1",
	// 	Group:    "group1",
	// 	StartID:  "$",
	// 	MkStream: true,
	// }
	source, err := rs.NewStreamSource(ctx, redisClient, readGroupArgs,
		nil, nil)
	if err != nil {
		log.Fatal(err)
	}

	toUpperMapFlow := flow.NewMap(toUpper, 1)
	sink := rs.NewStreamSink(ctx, redisClient, "stream2", nil)

	source.
		Via(toUpperMapFlow).
		To(sink)
}

var toUpper = func(msg *redis.XMessage) *redis.XMessage {
	log.Printf("Got: %v", msg.Values)
	values := make(map[string]any, len(msg.Values))
	for key, element := range msg.Values {
		values[key] = strings.ToUpper(fmt.Sprintf("%v", element))
	}
	msg.Values = values
	return msg
}
