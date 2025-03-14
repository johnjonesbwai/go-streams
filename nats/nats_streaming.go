package nats

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/johnjonesbwai/go-streams"
	"github.com/johnjonesbwai/go-streams/flow"
	stan "github.com/nats-io/stan.go"
)

// StreamingSource represents a NATS Streaming source connector.
// Deprecated: Use [JetStreamSource] instead.
type StreamingSource struct {
	conn             stan.Conn
	subscriptions    []stan.Subscription
	subscriptionType stan.SubscriptionOption
	topics           []string
	out              chan any

	logger *slog.Logger
}

var _ streams.Source = (*StreamingSource)(nil)

// NewStreamingSource returns a new [StreamingSource] connector.
func NewStreamingSource(ctx context.Context, conn stan.Conn,
	subscriptionType stan.SubscriptionOption,
	topics []string, logger *slog.Logger) *StreamingSource {
	if logger == nil {
		logger = slog.Default()
	}
	logger = logger.With(slog.Group("connector",
		slog.String("name", "nats.streaming"),
		slog.String("type", "source")))

	streamingSource := &StreamingSource{
		conn:             conn,
		subscriptions:    []stan.Subscription{},
		subscriptionType: subscriptionType,
		topics:           topics,
		out:              make(chan any),
		logger:           logger,
	}

	// asynchronously consume data and send it downstream
	go streamingSource.process(ctx)

	return streamingSource
}

func (ns *StreamingSource) process(ctx context.Context) {
	// bind all topic subscribers
	for _, topic := range ns.topics {
		subscription, err := ns.conn.Subscribe(topic, func(msg *stan.Msg) {
			ns.out <- msg
		}, ns.subscriptionType)
		if err != nil {
			ns.logger.Error("Failed to subscribe to topic",
				slog.String("topic", topic),
				slog.Any("error", err))
			continue
		}

		ns.logger.Info("Subscribed to topic", slog.String("topic", topic))
		ns.subscriptions = append(ns.subscriptions, subscription)
	}

	<-ctx.Done()

	ns.logger.Info("Closing connector")
	close(ns.out)
	ns.unsubscribe() // unbind all topic subscriptions

	if err := ns.conn.Close(); err != nil {
		ns.logger.Warn("Error in conn.Close", slog.Any("error", err))
	}
}

func (ns *StreamingSource) unsubscribe() {
	for _, subscription := range ns.subscriptions {
		if err := subscription.Unsubscribe(); err != nil {
			ns.logger.Warn("Failed to remove subscription",
				slog.Any("error", err))
		}
	}
}

// Via asynchronously streams data to the given Flow and returns it.
func (ns *StreamingSource) Via(operator streams.Flow) streams.Flow {
	flow.DoStream(ns, operator)
	return operator
}

// Out returns the output channel of the StreamingSource connector.
func (ns *StreamingSource) Out() <-chan any {
	return ns.out
}

// StreamingSink represents a NATS Streaming sink connector.
// Deprecated: Use [JetStreamSink] instead.
type StreamingSink struct {
	conn  stan.Conn
	topic string
	in    chan any

	done   chan struct{}
	logger *slog.Logger
}

var _ streams.Sink = (*StreamingSink)(nil)

// NewStreamingSink returns a new [StreamingSink] connector.
func NewStreamingSink(conn stan.Conn, topic string,
	logger *slog.Logger) *StreamingSink {
	if logger == nil {
		logger = slog.Default()
	}
	logger = logger.With(slog.Group("connector",
		slog.String("name", "nats.streaming"),
		slog.String("type", "sink")))

	streamingSink := &StreamingSink{
		conn:   conn,
		topic:  topic,
		in:     make(chan any),
		done:   make(chan struct{}),
		logger: logger,
	}

	// begin processing upstream data
	go streamingSink.process()

	return streamingSink
}

func (ns *StreamingSink) process() {
	defer close(ns.done) // signal data processing completion

	for msg := range ns.in {
		var err error
		switch message := msg.(type) {
		case *stan.Msg:
			err = ns.conn.Publish(ns.topic, message.Data)
		case []byte:
			err = ns.conn.Publish(ns.topic, message)
		default:
			ns.logger.Error("Unsupported message type",
				slog.String("type", fmt.Sprintf("%T", message)))
		}

		if err != nil {
			ns.logger.Error("Error processing message",
				slog.Any("error", err))
		}
	}

	ns.logger.Info("Closing connector")
	if err := ns.conn.Close(); err != nil {
		ns.logger.Warn("Error in conn.Close", slog.Any("error", err))
	}
}

// In returns the input channel of the StreamingSink connector.
func (ns *StreamingSink) In() chan<- any {
	return ns.in
}

// AwaitCompletion blocks until the StreamingSink connector has completed
// processing all the received data.
func (ns *StreamingSink) AwaitCompletion() {
	<-ns.done
}
