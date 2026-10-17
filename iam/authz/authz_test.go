package authz

import (
	"strings"
	"testing"

	stsv1 "within.website/x/gen/within/website/x/iam/sts/v1"
	iamv1 "within.website/x/gen/within/website/x/iam/v1"
)

func TestTryCompile(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name   string
		filter string
		valid  bool
	}{
		{"empty is unconditional", "", true},
		{"boolean literal", "true", true},
		{"principal fields", `principal.principal_id == "alice" && !principal.is_admin`, true},
		{"request fields deferred", `req.user_id == principal.principal_id`, true},
		{"unknown principal field", `principal.misspelled == "alice"`, false},
		{"wrong principal field type", `principal.is_admin == "true"`, false},
		{"syntax error", `req.user_id ==`, false},
		{"string result", `"true"`, false},
		{"integer result", `1`, false},
		{"principal result", `principal`, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := TryCompile(tt.filter)
			if (err == nil) != tt.valid {
				t.Fatalf("TryCompile(%q) = %v; valid = %v", tt.filter, err, tt.valid)
			}
		})
	}
}

func TestCheck(t *testing.T) {
	t.Parallel()
	method := MethodName(iamv1.KeyService_ServiceDesc.ServiceName, "ListKeys")
	eval, err := New(map[string]interface{}{method: &iamv1.ListKeysReq{}})
	if err != nil {
		t.Fatal(err)
	}
	allow := func(filter string) *iamv1.PolicyStatement {
		return &iamv1.PolicyStatement{Effect: iamv1.Effect_EFFECT_ALLOW, Methods: []string{"*"}, Filter: filter}
	}
	deny := func(filter string) *iamv1.PolicyStatement {
		return &iamv1.PolicyStatement{Effect: iamv1.Effect_EFFECT_DENY, Methods: []string{"*"}, Filter: filter}
	}
	for _, tt := range []struct {
		name       string
		statements []*iamv1.PolicyStatement
		userID     string
		admin      bool
		allowed    bool
		reason     string
	}{
		{name: "implicit deny", reason: "deny_implicit"},
		{name: "admin bypasses deny", statements: []*iamv1.PolicyStatement{deny("")}, admin: true, allowed: true, reason: "allow_admin"},
		{name: "empty allow", statements: []*iamv1.PolicyStatement{allow("")}, allowed: true},
		{name: "true allow", statements: []*iamv1.PolicyStatement{allow("true")}, allowed: true},
		{name: "false allow", statements: []*iamv1.PolicyStatement{allow("false")}, reason: "deny_implicit"},
		{name: "own request", statements: []*iamv1.PolicyStatement{allow(`req.user_id == principal.principal_id`)}, userID: "alice", allowed: true},
		{name: "other request", statements: []*iamv1.PolicyStatement{allow(`req.user_id == principal.principal_id`)}, userID: "bob"},
		{name: "default request", statements: []*iamv1.PolicyStatement{allow(`req.user_id == "" || req.user_id == principal.principal_id`)}, allowed: true},
		{name: "method variable", statements: []*iamv1.PolicyStatement{allow(`method.endsWith("/ListKeys")`)}, allowed: true},
		{name: "deny after allow", statements: []*iamv1.PolicyStatement{allow(""), deny("")}, reason: "deny by policy"},
		{name: "deny before allow", statements: []*iamv1.PolicyStatement{deny(""), allow("")}, reason: "deny by policy"},
		{name: "false deny falls through", statements: []*iamv1.PolicyStatement{deny("false"), allow("")}, allowed: true},
		{name: "conditional deny applies", statements: []*iamv1.PolicyStatement{allow(""), deny(`req.user_id == "bob"`)}, userID: "bob"},
		{name: "conditional deny does not apply", statements: []*iamv1.PolicyStatement{allow(""), deny(`req.user_id == "bob"`)}, userID: "alice", allowed: true},
		{name: "unknown request field allow fails closed", statements: []*iamv1.PolicyStatement{allow(`req.misspelled == "alice"`)}},
		{name: "unknown request field deny fails closed", statements: []*iamv1.PolicyStatement{allow(""), deny(`req.misspelled == "alice"`)}, reason: "undefined field"},
		{name: "unknown principal field allow fails closed", statements: []*iamv1.PolicyStatement{allow(`principal.misspelled == "alice"`)}},
		{name: "nonboolean allow fails closed", statements: []*iamv1.PolicyStatement{allow(`"true"`)}},
		{name: "nonboolean deny fails closed", statements: []*iamv1.PolicyStatement{allow(""), deny(`"false"`)}, reason: "must return a boolean"},
		{name: "runtime allow fails closed", statements: []*iamv1.PolicyStatement{allow(`1 / size(req.user_id) > 0`)}},
		{name: "runtime deny fails closed", statements: []*iamv1.PolicyStatement{allow(""), deny(`1 / size(req.user_id) > 0`)}, reason: "division by zero"},
		{name: "unmatched deny ignored", statements: []*iamv1.PolicyStatement{allow(""), {Effect: iamv1.Effect_EFFECT_DENY, Methods: []string{"Other/*"}}}, allowed: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// Each statement is in its own policy to exercise precedence across attachments.
			var policies []*iamv1.Policy
			for _, statement := range tt.statements {
				policies = append(policies, &iamv1.Policy{Statements: []*iamv1.PolicyStatement{statement}})
			}
			got := eval.Check(policies, method, &iamv1.ListKeysReq{UserId: tt.userID}, &stsv1.TokenIdentity{PrincipalId: "alice", IsAdmin: tt.admin})
			if got.Allowed != tt.allowed || !strings.Contains(got.Reason, tt.reason) {
				t.Fatalf("Check() = %+v; want allowed=%v, reason containing %q", got, tt.allowed, tt.reason)
			}
		})
	}
	if got := eval.Check(nil, "unknown", &iamv1.ListKeysReq{}, nil); got.Allowed {
		t.Fatalf("unknown method allowed: %+v", got)
	}
	if got := eval.Check(nil, "unknown", nil, &stsv1.TokenIdentity{IsAdmin: true}); !got.Allowed {
		t.Fatalf("admin should bypass environment lookup: %+v", got)
	}
}

func TestNewRejectsNonProtoRequest(t *testing.T) {
	if _, err := New(map[string]interface{}{"service/method": struct{}{}}); err == nil {
		t.Fatal("New accepted a non-protobuf request")
	}
}

func TestMatch(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name, pattern, value string
		want                 bool
	}{
		{"empty", "", "", true},
		{"star empty", "*", "", true},
		{"trailing star empty", "service/Read*", "service/Read", true},
		{"repeated stars", "service/Read***", "service/Read", true},
		{"global spans slash", "*", "service/Read", true},
		{"service wildcard", "service/*", "service/Read", true},
		{"prefix wildcard", "*/Read", "service/Read", true},
		{"middle wildcard", "s*ce/R*d", "service/Read", true},
		{"retry wildcard", "*ab", "aaab", true},
		{"multiple wildcard", "*a*b*c", "xxaayybbzzc", true},
		{"one character", "service/Re?d", "service/Read", true},
		{"one unicode character", "?", "é", true},
		{"exact", "service/Read", "service/Read", true},
		{"prefix insufficient", "service", "service/Read", false},
		{"empty pattern", "", "service/Read", false},
		{"question requires character", "?", "", false},
		{"wrong suffix", "*Write", "service/Read", false},
		{"literal brackets", "[ab]", "a", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := match(tt.pattern, tt.value); got != tt.want {
				t.Fatalf("match(%q, %q) = %v; want %v", tt.pattern, tt.value, got, tt.want)
			}
		})
	}
}
