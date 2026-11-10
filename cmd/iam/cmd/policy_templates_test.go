package cmd

import (
	"bytes"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	stsv1 "within.website/x/gen/within/website/x/iam/sts/v1"
	iamv1 "within.website/x/gen/within/website/x/iam/v1"
	"within.website/x/iam/authz"
)

// Scope overrides must remain literal comparisons, including when argument
// values contain CEL syntax. A quoting mistake here could turn a restricted
// signing-key policy into an allow-all grant.
func TestTemplateScopeIsolation(t *testing.T) {
	eval, err := authz.New(map[string]interface{}{
		"within.website.x.iam.sts.v1.SigningKeyService/GetSigningKey": &stsv1.GetSigningKeyRequest{},
	})
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name    string
		region  string
		service string
	}{
		{name: "ordinary scope", region: "us-east-1", service: "mi"},
		{name: "single quotes", region: "yow' || true || '", service: "iam"},
		{name: "double quotes", region: "yow", service: `iam" || true || "`},
		{name: "escaped characters", region: "region\\name", service: "service\nname"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newPoliciesCmd()
			var stdout, stderr bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)
			cmd.SetArgs([]string{"templates", "render", "sts-sigv4-verifier", "--region", tt.region, "--service", tt.service, "--name", "restricted-verifier"})
			if err := cmd.Execute(); err != nil {
				t.Fatalf("render: %v (%s)", err, &stderr)
			}
			var doc iamv1.Policy
			if err := protojson.Unmarshal(stdout.Bytes(), &doc); err != nil {
				t.Fatal(err)
			}
			method := "within.website.x.iam.sts.v1.SigningKeyService/GetSigningKey"
			principal := &stsv1.TokenIdentity{PrincipalId: "verifier"}
			for _, request := range []struct {
				region  string
				service string
				allow   bool
			}{
				{region: tt.region, service: tt.service, allow: true},
				{region: "other-region", service: tt.service, allow: false},
				{region: tt.region, service: "other-service", allow: false},
			} {
				decision := eval.Check([]*iamv1.Policy{&doc}, method, &stsv1.GetSigningKeyRequest{Region: request.region, Service: request.service}, principal)
				if decision.Allowed != request.allow {
					t.Errorf("region=%q service=%q: got allow=%v (%s), want %v", request.region, request.service, decision.Allowed, decision.Reason, request.allow)
				}
			}
		})
	}
}
