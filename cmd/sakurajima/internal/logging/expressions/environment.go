package expressions

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/ext"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	filterInvocations = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "techaro",
		Subsystem: "osiris",
		Name:      "slog_filter_invocations",
	}, []string{"name"})

	filterExecutionTime = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "techaro",
		Subsystem: "osiris",
		Name:      "slog_filter_execution_time",
		Buckets:   []float64{10, 50, 100, 200, 500, 1000, 2000, 5000, 10000}, // 10 microseconds to 10 milliseconds
	}, []string{"name"})
)

func New(opts ...cel.EnvOption) (*cel.Env, error) {
	args := []cel.EnvOption{
		ext.Strings(
			ext.StringsLocale("en_US"),
			ext.StringsValidateFormatCalls(true),
		),

		// default all timestamps to UTC
		cel.DefaultUTCTimeZone(true),

		// Functions exposed to all CEL programs:
		cel.Function("randInt",
			cel.Overload("randInt_int",
				[]*cel.Type{cel.IntType},
				cel.IntType,
				cel.UnaryBinding(func(val ref.Val) ref.Val {
					n, ok := val.(types.Int)
					if !ok {
						return types.ValOrErr(val, "value is not an integer, but is %T", val)
					}

					return types.Int(rand.IntN(int(n)))
				}),
			),
		),

		// Variables exposed to CEL programs:
		cel.Variable("time", cel.TimestampType),
		cel.Variable("msg", cel.StringType),
		cel.Variable("level", cel.StringType),
		cel.Variable("attrs", cel.MapType(cel.StringType, cel.StringType)),
	}

	args = append(args, opts...)
	return cel.NewEnv(args...)
}

// Compile takes CEL environment and syntax tree then emits an optimized
// Program for execution.
func Compile(env *cel.Env, src string) (cel.Program, error) {
	intermediate, iss := env.Compile(src)
	if iss != nil {
		return nil, iss.Err()
	}

	ast, iss := env.Check(intermediate)
	if iss != nil {
		return nil, iss.Err()
	}

	return env.Program(
		ast,
		cel.EvalOptions(
			// optimize regular expressions right now instead of on the fly
			cel.OptOptimize,
		),
	)
}

func NewFilter(lg *slog.Logger, name, src string) (*Filter, error) {
	env, err := New()
	if err != nil {
		return nil, fmt.Errorf("logging: can't create CEL env: %w", err)
	}

	program, err := Compile(env, src)
	if err != nil {
		return nil, fmt.Errorf("logging: can't compile expression: Compile(%q): %w", src, err)
	}

	return &Filter{
		program: program,
		name:    name,
		src:     src,
		log:     lg.With("filter", name),
	}, nil
}

func TryCompile(src string) error {
	env, err := New()
	if err != nil {
		return err
	}

	_, err = Compile(env, src)
	return err
}

type Filter struct {
	program cel.Program
	name    string
	src     string
	log     *slog.Logger
}

func (f Filter) Filter(ctx context.Context, r slog.Record) bool {
	t0 := time.Now()

	result, _, err := f.program.ContextEval(ctx, &Record{
		Record: r,
	})
	if err != nil {
		f.log.Error("error executing log filter", "err", err, "src", f.src)
		return false
	}
	dur := time.Since(t0)
	filterExecutionTime.WithLabelValues(f.name).Observe(float64(dur.Microseconds()))
	filterInvocations.WithLabelValues(f.name).Inc()
	f.log.Debug("filter execution", "dur", dur.Microseconds())

	if val, ok := result.(types.Bool); ok {
		return !bool(val)
	}

	return false
}

type Record struct {
	slog.Record
	attrs map[string]string
}

func (r *Record) Parent() cel.Activation { return nil }

func (r *Record) ResolveName(name string) (any, bool) {
	switch name {
	case "time":
		return &timestamp.Timestamp{Seconds: r.Time.Unix()}, true
	case "msg":
		return r.Message, true
	case "level":
		return r.Level.String(), true
	case "attrs":
		if r.attrs == nil {
			attrs := map[string]string{}

			r.Attrs(func(attr slog.Attr) bool {
				attrs[attr.Key] = attr.Value.String()
				return true
			})

			r.attrs = attrs
			return attrs, true
		}
		return r.attrs, true
	default:
		return nil, false
	}
}
