package tracer

import (
	"io"

	"github.com/opentracing/opentracing-go"
	jaeger "github.com/uber/jaeger-client-go"
	config "github.com/uber/jaeger-client-go/config"
)

// Tr represent
var Tr opentracing.Tracer
var closer io.Closer

// Init returns an instance of Jaeger Tracer that samples 100% of traces and logs all spans to stdout.
func init() {
	cfg := &config.Configuration{
		Sampler: &config.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LogSpans: true,
		},
	}
	Tr, cl, err := cfg.New("dsiem", config.Logger(jaeger.StdLogger))
	if err != nil {
		panic(err)
	}
	closer = cl
	opentracing.SetGlobalTracer(Tr)
}

// CloseTracer stop tracing
func CloseTracer() {
	closer.Close()
}
