package tracing

import "go.opencensus.io/trace"

func EndSpan(span *trace.Span, err *error) {
	RecordError(span, *err)
}

func RecordError(span *trace.Span, err error) {
	if err != nil {
		span.SetStatus(trace.Status{
			Code:    2,
			Message: err.Error(),
		})
		// For jaeger/opentracing: https://github.com/opentracing/specification/blob/master/semantic_conventions.md
		span.AddAttributes(trace.BoolAttribute("error", true))
	}
}
