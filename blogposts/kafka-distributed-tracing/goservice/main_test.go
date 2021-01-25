package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/trace"
)

func TestTraceWithParentID(t *testing.T) {
	shutdown := initProvider()
	defer shutdown()
	tracer := otel.Tracer("test-tracer")
	attWithTopic := make([]label.KeyValue, 10)
	attWithTopic = append(
		attWithTopic,
		label.String("messaging.destination_kind", "topic"),
		label.String("span.otel.kind", "CONSUMER"),
		label.String("messaging.system", "kafka"),
		label.String("net.transport", "IP.TCP"),
		label.String("messaging.url", "broker:29092"),
		label.String("messaging.operation", "receive"),
		label.String("messaging.destination", "stargazers-results"),
		//		label.String("messaging.message_id", spanContext.SpanID.String()
	)

	ctx := baggage.ContextWithValues(context.Background(),
		label.String("traceparent", "00-ad9dd3e072a6770c05b4fa117b3c50b7-dbde5131c90c0207-01"))
	traceID, _ := trace.TraceIDFromHex("bf25b10d0357dd3c5df5b80799611229")
	spanID, _ := trace.SpanIDFromHex("5e80edd6a64e9542")
	ctx = trace.ContextWithRemoteSpanContext(ctx,
		trace.SpanContext{
			TraceID: traceID,
			SpanID:  spanID,
		})

	ctx, span := tracer.Start(ctx, "consumer-go-service", trace.WithAttributes(attWithTopic...))
	<-time.After(1 * time.Second)
	span.End()

}

func TestParentRetrieve(t *testing.T) {
	propagatedtrace := "00-ad9dd3e072a6770c05b4fa117b3c50b7-dbde5131c90c0207-01"
	spandID, err := trace.SpanIDFromHex(propagatedtrace)
	assert.NoError(t, err)
	assert.Equal(t, spandID, "ad9dd3e072a6770c05b4fa117b3c50b7")
}

func TestSendingSnapshot(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	pipe := make(chan []int, 10)
	go func() {
		shared := make([]int, 10)
		stay := true
		for stay {
			select {
			case <-time.After(1 * time.Second):
				shared = append(shared, 1)
				pipe <- shared
			case <-ctx.Done():
				close(pipe)
				stay = false
			}
		}
	}()

	counter := 0
	for i := range pipe {
		counter++
		if counter == 10 {
			cancel()
		}
	}

}
