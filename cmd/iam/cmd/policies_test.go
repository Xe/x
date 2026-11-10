package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	iamv1 "within.website/x/gen/within/website/x/iam/v1"
)

func TestPolicyValidateRejectsInvalidDocuments(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{
			name:  "unknown rule field",
			input: `{"name":"example","statements":[{"effect":"EFFECT_ALLOW","method":["*"]}]}`,
		},
		{
			name:  "unknown enum value",
			input: `{"name":"example","statements":[{"effect":99,"methods":["*"]}]}`,
		},
		{
			name:  "unspecified effect",
			input: `{"name":"example","statements":[{"methods":["*"]}]}`,
		},
		{
			name:  "empty name",
			input: `{"statements":[{"effect":"EFFECT_ALLOW","methods":["*"]}]}`,
		},
		{
			name:  "empty statements",
			input: `{"name":"example","statements":[]}`,
		},
		{
			name:  "empty methods",
			input: `{"name":"example","statements":[{"effect":"EFFECT_ALLOW"}]}`,
		},
		{
			name:  "empty method pattern",
			input: `{"name":"example","statements":[{"effect":"EFFECT_ALLOW","methods":[""]}]}`,
		},
		{
			name:  "duplicate method patterns",
			input: `{"name":"example","statements":[{"effect":"EFFECT_ALLOW","methods":["*","*"]}]}`,
		},
		{
			name:  "invalid CEL syntax",
			input: `{"name":"example","statements":[{"effect":"EFFECT_ALLOW","methods":["*"],"filter":"req.user_id =="}]}`,
		},
		{
			name:  "unknown principal field",
			input: `{"name":"example","statements":[{"effect":"EFFECT_ALLOW","methods":["*"],"filter":"principal.misspelled == 'x'"}]}`,
		},
		{
			name:  "RPC wrapper instead of document",
			input: `{"policy":{"name":"example","statements":[{"effect":"EFFECT_ALLOW","methods":["*"]}]}}`,
		},
		{
			name:  "duplicate document fields",
			input: `{"name":"first","name":"second","statements":[{"effect":"EFFECT_ALLOW","methods":["*"]}]}`,
		},
		{
			name:  "multiple JSON documents",
			input: `{"name":"example","statements":[{"effect":"EFFECT_ALLOW","methods":["*"]}]} {}`,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newPoliciesCmd()
			cmd.SetIn(strings.NewReader(tt.input))
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})
			cmd.SetArgs([]string{"validate", "--file", "-"})
			if err := cmd.Execute(); err == nil {
				t.Error("validate accepted an invalid document")
			}
		})
	}
}

func TestPolicyExportImportRoundTrip(t *testing.T) {
	original := &iamv1.Policy{
		Id:   "019d0000-0000-7000-8000-000000000001",
		Name: "own-keys",
		Statements: []*iamv1.PolicyStatement{
			{
				Effect:  iamv1.Effect_EFFECT_ALLOW,
				Methods: []string{"within.website.x.iam.v1.KeyService/ListKeys"},
				Filter:  `req.user_id == "" || req.user_id == principal.principal_id`,
			},
			{
				Effect:  iamv1.Effect_EFFECT_DENY,
				Methods: []string{"*/CreateKey"},
				Filter:  `req.user_id != principal.principal_id`,
			},
		},
		CreatedAt: timestamppb.New(time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)),
		UpdatedAt: timestamppb.New(time.Date(2026, 10, 16, 13, 0, 0, 0, time.UTC)),
	}
	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)
	if err := writePolicyJSON(cmd, original); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "exported.json")
	if err := os.WriteFile(path, out.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	decoded, err := loadPolicyFile(cmd, path)
	if err != nil {
		t.Fatalf("importing exported policy: %v", err)
	}
	if !proto.Equal(original, decoded) {
		t.Errorf("export/import changed the policy:\nwant %v\ngot  %v", original, decoded)
	}
}
