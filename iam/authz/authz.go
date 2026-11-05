// Package authz evaluates IAM policy filters over (method, request, principal).
package authz

import (
	"fmt"
	"strings"
	"time"

	"cel.dev/cel-go/cel"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/protobuf/proto"
	stsv1 "within.website/x/gen/within/website/x/iam/sts/v1"
	iamv1 "within.website/x/gen/within/website/x/iam/v1"
)

var (
	decisions = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "iam_authz_decisions_total",
		Help: "IAM authorization decisions by outcome.",
	}, []string{"decision"})
	filterSeconds = promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "iam_authz_filter_seconds",
		Help: "Time spent compiling and evaluating each IAM policy filter.",
	})
	filterErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "iam_authz_filter_errors_total",
		Help: "IAM policy filters that failed compilation or evaluation.",
	})
)

func decision(allowed bool, reason, outcome string) *Decision {
	decisions.WithLabelValues(outcome).Inc()
	return &Decision{Allowed: allowed, Reason: reason}
}

// Decision is the result of evaluating one policy set against a call.
type Decision struct {
	// Allowed is true when some applying ALLOW statement wins and no DENY
	// statement applies. An admin caller always yields Allowed=true.
	Allowed bool
	// Reason records the deciding rule: allow_admin, allow_policy, or the
	// denying statement's effect/method for denials.
	Reason string
}

// Evaluator registers the mounted request types at startup and checks policy
// filters against their concrete request and principal types on each call.
type Evaluator struct {
	envs map[string]*cel.Env
}

// New builds an evaluator over every method the mounted services accept.
// methodTypes maps a full method name to a sample protobuf request message.
// Filters are type-checked against that method's request at enforcement time.
func New(methodTypes map[string]interface{}) (*Evaluator, error) {
	e := &Evaluator{envs: make(map[string]*cel.Env, len(methodTypes))}
	for method, sample := range methodTypes {
		request, ok := sample.(proto.Message)
		if !ok || request == nil {
			return nil, fmt.Errorf("request type for %s must be a protobuf message", method)
		}
		env, err := cel.NewEnv(
			cel.Types(request, &stsv1.TokenIdentity{}),
			cel.Variable("method", cel.StringType),
			cel.Variable("req", cel.ObjectType(string(request.ProtoReflect().Descriptor().FullName()))),
			cel.Variable("principal", principalType()),
		)
		if err != nil {
			return nil, fmt.Errorf("CEL environment for %s: %w", method, err)
		}
		e.envs[method] = env
	}
	return e, nil
}

func principalType() *cel.Type {
	return cel.ObjectType(string((&stsv1.TokenIdentity{}).ProtoReflect().Descriptor().FullName()))
}

// MethodName builds the full method name <package>.<Service>/<Method> that
// policy statements match against.
func MethodName(service, method string) string {
	return strings.Join([]string{service, method}, "/")
}

// PrincipalFromUser builds the TokenIdentity principal used by policy
// filters from the caller's access key and resolved user.
func PrincipalFromUser(keyID string, u *iamv1.User) *stsv1.TokenIdentity {
	return &stsv1.TokenIdentity{
		AccessKeyId: keyID,
		PrincipalId: u.GetId(),
		IsAdmin:     u.GetIsAdmin(),
	}
}

// TryCompile checks a boolean CEL filter with a concrete principal type.
// req remains dynamic because policies can target other services' messages;
// the evaluator checks request fields against the method's concrete type.
// An empty filter applies unconditionally.
func TryCompile(src string) error {
	if src == "" {
		return nil
	}
	env, err := cel.NewEnv(
		cel.Types(&stsv1.TokenIdentity{}),
		cel.Variable("method", cel.StringType),
		cel.Variable("req", cel.DynType),
		cel.Variable("principal", principalType()),
	)
	if err != nil {
		return err
	}
	_, err = compileFilter(env, src)
	return err
}

func compileFilter(env *cel.Env, src string) (*cel.Ast, error) {
	ast, issues := env.Compile(src)
	if err := issues.Err(); err != nil {
		return nil, err
	}
	if !ast.OutputType().IsExactType(cel.BoolType) {
		return nil, fmt.Errorf("filter must return a boolean, got %s", ast.OutputType())
	}
	return ast, nil
}

// Check evaluates every attached policy in order. Admin callers bypass
// everything. A DENY statement that applies wins; otherwise the first
// applying ALLOW allows. Failure is closed: a filter that fails to compile
// or errors at evaluation is treated as not applying on ALLOW and as
// applying on DENY.
func (e *Evaluator) Check(policies []*iamv1.Policy, method string, msg interface{}, principal *stsv1.TokenIdentity) *Decision {
	if principal.GetIsAdmin() {
		return decision(true, "allow_admin", "allow_admin")
	}

	env, ok := e.envs[method]
	if !ok {
		// No environment registered for this method: fail closed.
		return decision(false, "deny: no policy environment", "deny_filter_error")
	}

	var allow *Decision
	var filterErr error
	for _, p := range policies {
		for _, st := range p.GetStatements() {
			if !applies(st, method) {
				continue
			}
			started := time.Now()
			matches, err := evalFilter(env, st.GetFilter(), method, msg, principal)
			filterSeconds.Observe(time.Since(started).Seconds())
			if err != nil {
				filterErr = err
				filterErrors.Inc()
			}
			switch st.GetEffect() {
			case iamv1.Effect_EFFECT_DENY:
				if err != nil {
					return decision(false, fmt.Sprintf("deny: policy filter on %s: %v", method, err), "deny_filter_error")
				}
				if matches {
					return decision(false, "deny by policy: "+st.GetEffect().String()+" on "+method, "deny_explicit")
				}
			case iamv1.Effect_EFFECT_ALLOW:
				if err == nil && matches && allow == nil {
					allow = &Decision{Allowed: true, Reason: "allow by policy: " + st.GetEffect().String() + " on " + method}
				}
			}
		}
	}
	if allow != nil {
		return decision(true, allow.Reason, "allow_policy")
	}
	if filterErr != nil {
		return decision(false, fmt.Sprintf("deny: policy filter on %s: %v", method, filterErr), "deny_filter_error")
	}
	return decision(false, "deny_implicit", "deny_implicit")
}

// applies reports whether a statement's methods patterns match the method.
func applies(st *iamv1.PolicyStatement, method string) bool {
	for _, pat := range st.GetMethods() {
		if match(pat, method) {
			return true
		}
	}
	return false
}

// match reports whether a glob pattern matches the whole string. '*' spans
// any characters including '/'; '?' matches one character.
func match(pattern, s string) bool {
	pat, value := []rune(pattern), []rune(s)
	pi, si, star, retry := 0, 0, -1, 0
	for si < len(value) {
		switch {
		case pi < len(pat) && (pat[pi] == '?' || pat[pi] == value[si]):
			pi++
			si++
		case pi < len(pat) && pat[pi] == '*':
			star, retry = pi, si
			pi++
		case star >= 0:
			retry++
			si, pi = retry, star+1
		default:
			return false
		}
	}
	for pi < len(pat) && pat[pi] == '*' {
		pi++
	}
	return pi == len(pat)
}

// evalFilter compiles and evaluates a CEL filter over (method, req,
// principal). It returns true when the filter evaluates to true. Errors
// (including type errors) are surfaced to the caller so fail-closed logic
// can decide.
func evalFilter(env *cel.Env, filter string, method string, msg interface{}, principal *stsv1.TokenIdentity) (bool, error) {
	if filter == "" {
		return true, nil
	}
	ast, err := compileFilter(env, filter)
	if err != nil {
		return false, err
	}
	program, err := env.Program(ast, cel.EvalOptions(
		cel.OptOptimize,
	))
	if err != nil {
		return false, fmt.Errorf("program: %w", err)
	}
	out, _, err := program.Eval(map[string]any{
		"method":    method,
		"req":       msg,
		"principal": principal,
	})
	if err != nil {
		return false, fmt.Errorf("eval: %w", err)
	}
	v := out.Value()
	b, ok := v.(bool)
	if !ok {
		return false, fmt.Errorf("filter did not return a boolean")
	}
	return b, nil
}
