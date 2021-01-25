package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"google.golang.org/grpc"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpgrpc"
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/propagation"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv"
	"go.opentelemetry.io/otel/trace"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

func initProvider() func() {
	ctx := context.Background()
	collectorAddr, ok := os.LookupEnv("OTEL_ENDPOINT")
	if !ok {
		collectorAddr = "otel:55680"
	}

	driver := otlpgrpc.NewDriver(
		otlpgrpc.WithInsecure(),
		otlpgrpc.WithEndpoint(collectorAddr),
		otlpgrpc.WithDialOption(grpc.WithBlock()), // useful for testing
	)
	exp, err := otlp.NewExporter(ctx, driver)
	handleErr(err, "failed to create exporter")

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String("api-goservice"),
		),
	)
	handleErr(err, "failed to create resource")

	bsp := sdktrace.NewBatchSpanProcessor(exp)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithConfig(sdktrace.Config{DefaultSampler: sdktrace.AlwaysSample()}),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	cont := controller.New(
		processor.New(
			simple.NewWithExactDistribution(),
			exp,
		),
		controller.WithPusher(exp),
		controller.WithCollectPeriod(2*time.Second),
	)

	// set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.TraceContext{})
	otel.SetTracerProvider(tracerProvider)
	otel.SetMeterProvider(cont.MeterProvider())
	handleErr(cont.Start(context.Background()), "failed to start controller")

	return func() {
		// Shutdown will flush any remaining spans.
		handleErr(tracerProvider.Shutdown(ctx), "failed to shutdown TracerProvider")

		// Push any last metric events to the exporter.
		handleErr(cont.Stop(context.Background()), "failed to stop controller")
	}
}

func handleErr(err error, message string) {
	if err != nil {
		log.Fatalf("%s: %v", message, err)
	}
}

func main() {
	closeOtel := initProvider()
	defer closeOtel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	msgs := consumerLoop(&ctx)
	data := make([]stargazerEvent, 0)
	go func() {
		for msg := range msgs {
			data = append(data, msg)
		}
	}()
	app := fiber.New()
	app.Use(logger.New())
	app.Get("/", func(c *fiber.Ctx) error {
		return c.Status(200).JSON(data)
	})

	app.Listen(":3100")
}

func spanFromHeaders(topic string, headers []kafka.Header) {
	var propagatedId string
	for _, h := range headers {
		if h.Key == "traceparent" {
			propagatedId = string(h.Value)
			break
		}
	}

	if propagatedId == "" {
		return
	}

	tracer := otel.Tracer("api-goservice")
	attWithTopic := make([]label.KeyValue, 10)
	attWithTopic = append(
		attWithTopic,
		label.String("messaging.destination_kind", "topic"),
		label.String("span.otel.kind", "CONSUMER"),
		label.String("messaging.system", "kafka"),
		label.String("net.transport", "IP.TCP"),
		label.String("messaging.url", "localhost:9092"),
		label.String("messaging.operation", "receive"),
		label.String("messaging.destination", "stargazers-results"),
		//		label.String("messaging.message_id", spanContext.SpanID.String()
	)

	traceID, spanID, err := parseParentTrace(propagatedId)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing parent trace %v \n", err)
		return
	}
	ctx := trace.ContextWithRemoteSpanContext(context.Background(),
		trace.SpanContext{
			TraceID: *traceID,
			SpanID:  *spanID,
		})
	ctx, span := tracer.Start(ctx, "consumer-go-service", trace.WithAttributes(attWithTopic...))
	defer span.End()
}
func parseParentTrace(ptrace string) (*trace.TraceID, *trace.SpanID, error) {
	tokens := strings.Split(ptrace, "-")
	traceID, err := trace.TraceIDFromHex(tokens[1])
	if err != nil {
		return nil, nil, err
	}
	spanID, err := trace.SpanIDFromHex(tokens[2])
	if err != nil {
		return nil, nil, err
	}
	return &traceID, &spanID, nil
}

func consumerLoop(ctx *context.Context) <-chan stargazerEvent {
	stargazers := make(chan stargazerEvent, 1)

	go func() {
		sigchan := make(chan os.Signal, 1)
		signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
		broker, ok := os.LookupEnv("BOOTSTRAP_SERVERS")
		if !ok {
			broker = "localhost:9092"
		}
		c, err := kafka.NewConsumer(&kafka.ConfigMap{
			"bootstrap.servers":        broker,
			"group.id":                 uuid.New().String(),
			"enable.auto.commit":       "false",
			"debug":                    "consumer",
			"auto.offset.reset":        "earliest",
			"go.events.channel.enable": true,
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "error creating consumer %v \n", err)
		}
		handleErr(c.Subscribe("stargazers-results", nil), "error subscribing to topic")
		defer c.Close()
		for {
			select {
			case sig := <-sigchan:
				fmt.Printf("Caught signal %v: terminating\n", sig)
				os.Exit(0)
			case <-(*ctx).Done():
				fmt.Printf("exiting consumer loop due to context cancellation")
				return
			case ev := <-c.Events():
				switch e := ev.(type) {

				case *kafka.Message:
					fmt.Printf("%% Message on %s:\n%s\n",
						e.TopicPartition, string(e.Value))
					var evt stargazerEvent
					if err := json.Unmarshal(e.Value, &evt); err != nil {
						fmt.Printf(" Error: %v\n", err)
						c.Close()
					}
					spanFromHeaders("stargazer-results", e.Headers)
					stargazers <- evt

				case kafka.Error:
					fmt.Fprintf(os.Stderr, "%% Error: %v\n", e)
					c.Close()
				}
			}
		}
	}()
	return stargazers
}

type stargazerEvent struct {
	Login string `json:"LOGIN"`
	Type  string `json:"TYPE"`
}
