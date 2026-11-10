package cmd

import (
	"fmt"
	"io"
	"os"
	"text/tabwriter"
	"time"

	"buf.build/go/protovalidate"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"within.website/x/cmd/iamd/pub/iam"
	iamv1 "within.website/x/gen/within/website/x/iam/v1"
	"within.website/x/iam/authz"
)

func init() {
	rootCmd.AddCommand(newPoliciesCmd())
}

func newPoliciesCmd() *cobra.Command {
	policies := &cobra.Command{
		Use:   "policies <command>",
		Short: "Manage IAM policies and user attachments",
		Long:  "Manage immutable IAM policies. Files contain a Policy document in protobuf JSON, not a CreatePolicy request. Remote operations require admin credentials.",
	}

	var validateFile string
	validate := &cobra.Command{
		Use:   "validate --file <path|->",
		Short: "Validate a policy file offline, without credentials",
		Long:  "Validate protobuf JSON, policy field rules, and CEL compilation without contacting iamd. Request-specific CEL fields are checked by the enforcing service, not by this command.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := loadPolicyFile(cmd, validateFile); err != nil {
				return err
			}
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "policy valid")
			return err
		},
	}
	validate.Flags().StringVarP(&validateFile, "file", "f", "", "Policy JSON file, or - for stdin")
	validate.MarkFlagRequired("file")

	var createFile string
	create := &cobra.Command{
		Use:   "create --file <path|->",
		Short: "Create a policy from a JSON file; print its ID",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			doc, err := loadPolicyFile(cmd, createFile)
			if err != nil {
				return err
			}
			cli, err := newPolicyClient(cmd)
			if err != nil {
				return err
			}
			resp, err := cli.CreatePolicy(cmd.Context(), &iamv1.CreatePolicyReq{Policy: doc})
			if err != nil {
				return fmt.Errorf("creating policy: %w", err)
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), resp.GetPolicy().GetId())
			return err
		},
	}
	create.Flags().StringVarP(&createFile, "file", "f", "", "Policy JSON file, or - for stdin")
	create.MarkFlagRequired("file")

	var getJSON bool
	get := &cobra.Command{
		Use:   "get <policy-id> [--json]",
		Short: "Show a policy; --json exports a reusable Policy document",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := newPolicyClient(cmd)
			if err != nil {
				return err
			}
			resp, err := cli.GetPolicy(cmd.Context(), &iamv1.GetPolicyReq{PolicyId: args[0]})
			if err != nil {
				return fmt.Errorf("getting policy: %w", err)
			}
			if getJSON {
				return writePolicyJSON(cmd, resp.GetPolicy())
			}
			return printPolicy(cmd.OutOrStdout(), resp.GetPolicy())
		},
	}
	get.Flags().BoolVarP(&getJSON, "json", "j", false, "Export the Policy as protobuf JSON")

	var listCount, listPage int32
	var listJSON bool
	var listUserID string
	list := &cobra.Command{
		Use:   "list [--user-id <user-id>] [--json]",
		Short: "List policy documents, optionally restricted to a user's attachments",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if listUserID == "" && (listCount <= 0 || listPage < 0) {
				return fmt.Errorf("--count must be positive and --page must be non-negative")
			}
			if listUserID != "" && (cmd.Flags().Changed("count") || cmd.Flags().Changed("page")) {
				return fmt.Errorf("--count and --page cannot be combined with --user-id; user attachments are not paginated")
			}
			cli, err := newPolicyClient(cmd)
			if err != nil {
				return err
			}
			var docs []*iamv1.Policy
			if listUserID != "" {
				resp, err := cli.ListPoliciesForUser(cmd.Context(), &iamv1.ListPoliciesForUserReq{UserId: listUserID})
				if err != nil {
					return fmt.Errorf("listing policies for user: %w", err)
				}
				docs = resp.GetPolicies()
			} else {
				resp, err := cli.ListPolicies(cmd.Context(), &iamv1.ListPoliciesReq{Count: listCount, Page: listPage})
				if err != nil {
					return fmt.Errorf("listing policies: %w", err)
				}
				docs = resp.GetPolicies()
			}
			if listJSON {
				return writePolicyJSON(cmd, &iamv1.ListPoliciesResp{Policies: docs})
			}
			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', tabwriter.StripEscape|tabwriter.TabIndent)
			fmt.Fprintln(tw, "Policy ID\tName\tStatements\tCreated\t")
			for _, p := range docs {
				fmt.Fprintf(tw, "%s\t%s\t%d\t%s\t\n", p.GetId(), p.GetName(), len(p.GetStatements()), p.GetCreatedAt().AsTime().Format(time.RFC3339))
			}
			return tw.Flush()
		},
	}
	list.Flags().Int32VarP(&listCount, "count", "c", 40, "Maximum policies per page (global list only)")
	list.Flags().Int32VarP(&listPage, "page", "p", 0, "Zero-based page (global list only)")
	list.Flags().BoolVarP(&listJSON, "json", "j", false, "Format the policy list as protobuf JSON")
	list.Flags().StringVar(&listUserID, "user-id", "", "List all policies attached to this user")

	var attachUserID string
	attach := &cobra.Command{
		Use:   "attach <policy-id> --user-id <user-id>",
		Short: "Attach a policy to a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := newPolicyClient(cmd)
			if err != nil {
				return err
			}
			if _, err := cli.AttachPolicy(cmd.Context(), &iamv1.AttachPolicyReq{PolicyId: args[0], UserId: attachUserID}); err != nil {
				return fmt.Errorf("attaching policy: %w", err)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "policy %s attached to user %s\n", args[0], attachUserID)
			return err
		},
	}
	attach.Flags().StringVar(&attachUserID, "user-id", "", "User to attach the policy to")
	attach.MarkFlagRequired("user-id")

	var detachUserID string
	detach := &cobra.Command{
		Use:   "detach <policy-id> --user-id <user-id>",
		Short: "Detach a policy from a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := newPolicyClient(cmd)
			if err != nil {
				return err
			}
			if _, err := cli.DetachPolicy(cmd.Context(), &iamv1.DetachPolicyReq{PolicyId: args[0], UserId: detachUserID}); err != nil {
				return fmt.Errorf("detaching policy: %w", err)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "policy %s detached from user %s\n", args[0], detachUserID)
			return err
		},
	}
	detach.Flags().StringVar(&detachUserID, "user-id", "", "User to detach the policy from")
	detach.MarkFlagRequired("user-id")

	delete := &cobra.Command{
		Use:   "delete <policy-id>",
		Short: "Delete an unattached policy document",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := newPolicyClient(cmd)
			if err != nil {
				return err
			}
			if _, err := cli.DeletePolicy(cmd.Context(), &iamv1.DeletePolicyReq{PolicyId: args[0]}); err != nil {
				return fmt.Errorf("deleting policy: %w", err)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "policy %s deleted\n", args[0])
			return err
		},
	}

	policies.AddCommand(validate, create, get, list, attach, detach, delete, newPolicyTemplatesCmd())
	return policies
}

// loadPolicyFile accepts the same document get --json exports. Server-owned
// metadata may be present; CreatePolicy assigns a new ID and timestamps.
func loadPolicyFile(cmd *cobra.Command, path string) (*iamv1.Policy, error) {
	var data []byte
	var err error
	if path == "-" {
		data, err = io.ReadAll(cmd.InOrStdin())
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, fmt.Errorf("reading policy %q: %w", path, err)
	}
	var doc iamv1.Policy
	// Reject unknown and duplicate fields rather than silently discarding a
	// misspelled rule. A file is a Policy, never a CreatePolicyReq wrapper.
	if err := protojson.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("decoding policy %q: %w", path, err)
	}
	if err := validatePolicy(&doc, path); err != nil {
		return nil, err
	}
	return &doc, nil
}

func validatePolicy(doc *iamv1.Policy, source string) error {
	if err := protovalidate.Validate(doc); err != nil {
		return fmt.Errorf("validating policy %q: %w", source, err)
	}
	for i, st := range doc.GetStatements() {
		if err := authz.TryCompile(st.GetFilter()); err != nil {
			return fmt.Errorf("policy %q statement %d filter: %w", source, i+1, err)
		}
	}
	return nil
}

func newPolicyClient(cmd *cobra.Command) (iamv1.PolicyService, error) {
	cfg, err := loadConfig(cfgFile)
	if err != nil {
		return nil, fmt.Errorf("loading config from %s: %w", cfgFile, err)
	}
	cli, err := iam.New(cmd.Context(), cfg.Endpoint, cfg.Region, cfg.AccessKeyID, cfg.SecretAccessKey)
	if err != nil {
		return nil, fmt.Errorf("creating IAM client: %w", err)
	}
	return cli.Policies, nil
}

func writePolicyJSON(cmd *cobra.Command, msg proto.Message) error {
	data, err := (protojson.MarshalOptions{Indent: "  ", UseProtoNames: true}).Marshal(msg)
	if err != nil {
		return fmt.Errorf("encoding policy JSON: %w", err)
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return err
}

func printPolicy(w io.Writer, p *iamv1.Policy) error {
	if _, err := fmt.Fprintf(w, "Policy ID: %s\nName:      %s\nCreated:   %s\nUpdated:   %s\n",
		p.GetId(), p.GetName(), p.GetCreatedAt().AsTime().Format(time.RFC3339), p.GetUpdatedAt().AsTime().Format(time.RFC3339)); err != nil {
		return err
	}
	for i, st := range p.GetStatements() {
		if _, err := fmt.Fprintf(w, "\nStatement %d: %s\n", i+1, st.GetEffect()); err != nil {
			return err
		}
		for _, method := range st.GetMethods() {
			if _, err := fmt.Fprintf(w, "  Method: %s\n", method); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(w, "  Filter: %s\n", st.GetFilter()); err != nil {
			return err
		}
	}
	return nil
}
