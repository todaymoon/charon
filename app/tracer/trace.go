// Copyright © 2022-2025 Obol Labs Inc. Licensed under the terms of a Business Source License 1.1

// Package tracer provides a global OpenTelemetry tracer.
package tracer

import (
	"context"
	"io"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/obolnetwork/charon/app/errors"
)

// tracer is the global app level tracer, it defaults to a noop tracer.
var tracer = noop.NewTracerProvider().Tracer("")

// Start creates a span and a context.Context containing the newly-created span from the global tracer.
// See go.opentelemetry.io/otel/trace#Start for more details.
func Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return tracer.Start(ctx, spanName, opts...) //nolint:spancheck // we defer end span outside of this function
}

// RootedCtx returns a copy of the parent context containing a tracing span context
// rooted to the trace ID. All spans started from the context will be rooted to the trace ID.
func RootedCtx(ctx context.Context, traceID trace.TraceID) context.Context {
	return trace.ContextWithSpanContext(ctx, trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceID,
	}))
}

// Init initialises the global tracer via the option(s) defaulting to a noop tracer. It returns a shutdown function.
func Init(opts ...func(*options)) (func(context.Context) error, error) {
	var o options
	for _, opt := range opts {
		opt(&o)
	}

	if o.expFunc == nil {
		return func(context.Context) error {
			return nil
		}, nil
	}

	exp, err := o.expFunc()
	if err != nil {
		return nil, err
	}

	tp := newTraceProvider(exp, "")

	// Set globals
	otel.SetTracerProvider(tp)
	tracer = tp.Tracer("")

	return tp.Shutdown, nil
}

type options struct {
	expFunc func() (sdktrace.SpanExporter, error)
}

// WithStdOut returns an option to configure an OpenTelemetry exporter for tracing
// telemetry to be written to an output destination as JSON.
func WithStdOut(w io.Writer) func(*options) {
	return func(o *options) {
		o.expFunc = func() (sdktrace.SpanExporter, error) {
			ex, err := stdouttrace.New(stdouttrace.WithWriter(w))
			if err != nil {
				return nil, errors.Wrap(err, "stdouttrace error")
			}

			return ex, nil
		}
	}
}

func newTraceProvider(exp sdktrace.SpanExporter, service string) *sdktrace.TracerProvider {
	r := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(service),
	)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(r),
	)

	return tp
}
