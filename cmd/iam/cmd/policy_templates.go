package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"
	"within.website/x/cmd/iam/examples"
	iamv1 "within.website/x/gen/within/website/x/iam/v1"
)

func newPolicyTemplatesCmd() *cobra.Command {
	templates := &cobra.Command{
		Use:   "templates <command>",
		Short: "List and render built-in policy templates offline",
		Long:  "List and render the policy examples bundled with this binary. No credentials, server connection, or source checkout are required. Rendering does not create or attach a policy.",
	}
	list := &cobra.Command{
		Use:   "list",
		Short: "List available template names",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			entries, err := examples.Policies.ReadDir(".")
			if err != nil {
				return fmt.Errorf("listing policy templates: %w", err)
			}
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSuffix(entry.Name(), ".json")); err != nil {
					return err
				}
			}
			return nil
		},
	}

	var name, region, service string
	render := &cobra.Command{
		Use:   "render <template-name>",
		Short: "Write a policy document as protobuf JSON to stdout",
		Long: "Render a built-in template, optionally changing the policy name. Save stdout to a file or pipe it to policies create --file -. " +
			"For sts-sigv4-verifier only, --region and --service together replace the example's yow/iam scope. " +
			"That template grants forgery-capable symmetric signing keys; prefer sts-sigv4a-verifier for verification-only access.",
		Example: "  iam policies templates render own-keys --name alice-keys > alice-keys.json\n" +
			"  iam policies templates render sts-sigv4-verifier --region us-east-1 --service mi --name mi-verifier",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			template := args[0]
			data, err := examples.Policies.ReadFile(template + ".json")
			if err != nil {
				return fmt.Errorf("loading policy template %q (see policies templates list): %w", template, err)
			}
			var doc iamv1.Policy
			if err := protojson.Unmarshal(data, &doc); err != nil {
				return fmt.Errorf("decoding policy template %q: %w", template, err)
			}
			if cmd.Flags().Changed("region") || cmd.Flags().Changed("service") {
				if template != "sts-sigv4-verifier" {
					return fmt.Errorf("--region and --service apply only to sts-sigv4-verifier")
				}
				if region == "" || service == "" {
					return fmt.Errorf("--region and --service must be non-empty")
				}
				if len(doc.GetStatements()) != 1 {
					return fmt.Errorf("sts-sigv4-verifier must contain exactly one scoped statement")
				}
				// Quote scope values as CEL string literals so CLI arguments
				// cannot inject expressions or widen the template's grant.
				doc.Statements[0].Filter = fmt.Sprintf("req.region == %s && req.service == %s", strconv.Quote(region), strconv.Quote(service))
				doc.Name = fmt.Sprintf("sts-sigv4-verifier-%s-%s", region, service)
			}
			if cmd.Flags().Changed("name") {
				doc.Name = name
			}
			if err := validatePolicy(&doc, "template "+template); err != nil {
				return err
			}
			return writePolicyJSON(cmd, &doc)
		},
	}
	render.Flags().StringVar(&name, "name", "", "Override the policy document's name")
	render.Flags().StringVar(&region, "region", "", "Override the classic SigV4 region; requires --service")
	render.Flags().StringVar(&service, "service", "", "Override the classic SigV4 service; requires --region")
	render.MarkFlagsRequiredTogether("region", "service")

	templates.AddCommand(list, render)
	return templates
}
