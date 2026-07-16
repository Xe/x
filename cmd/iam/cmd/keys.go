package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"within.website/x/cmd/iamd/pub/iam"
	iamv1 "within.website/x/gen/within/website/x/iam/v1"
)

var (
	keysCmd = &cobra.Command{
		Use:     "keys <command>",
		Short:   "Signing key management",
		Aliases: []string{"k"},
	}

	keyCreateComment string
	keyCreateUserID  string

	keyCreateCmd = &cobra.Command{
		Use:     "create <flags>",
		Short:   "Create a new IAM signing key",
		Aliases: []string{"new"},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(cfgFile)
			if err != nil {
				return fmt.Errorf("loading config from %s: %w", cfgFile, err)
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			cli, err := iam.New(ctx, cfg.Endpoint, cfg.Region, cfg.AccessKeyID, cfg.SecretAccessKey)
			if err != nil {
				return fmt.Errorf("can't create IAM client: %w", err)
			}

			resp, err := cli.Keys.CreateKey(ctx, &iamv1.CreateKeyReq{
				Comment: keyCreateComment,
				UserId:  keyCreateUserID,
			})
			if err != nil {
				return err
			}

			k := resp.GetKey()
			fmt.Printf("Access key ID:     %s\n", k.GetAccessKeyId())
			fmt.Printf("Comment:           %s\n", k.GetComment())
			fmt.Printf("Secret access key: %s\n", resp.GetSecretAccessKey())

			return nil
		},
	}

	keyDisableUserID string

	keyDisableCmd = &cobra.Command{
		Use:   "disable <key-id> <reason>",
		Short: "Disable an IAM signing key",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				return fmt.Errorf("please supply a key ID and reason in args")
			}

			cfg, err := loadConfig(cfgFile)
			if err != nil {
				return fmt.Errorf("loading config from %s: %w", cfgFile, err)
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			cli, err := iam.New(ctx, cfg.Endpoint, cfg.Region, cfg.AccessKeyID, cfg.SecretAccessKey)
			if err != nil {
				return fmt.Errorf("can't create IAM client: %w", err)
			}

			keyID := args[0]
			reason := args[1]

			if _, err := cli.Keys.DisableKey(ctx, &iamv1.DisableKeyReq{
				KeyId:  keyID,
				Reason: reason,
				UserId: keyDisableUserID,
			}); err != nil {
				return fmt.Errorf("can't disable key: %w", err)
			}

			fmt.Println("key", args[0], "disabled")
			return nil
		},
	}

	keyListCount, keyListPage int32
	keyListJSON               bool
	keyListUserID             string

	keyListCmd = &cobra.Command{
		Use:   "list <--count=> <--page=> [--json] [--user-id=]",
		Short: "List IAM signing keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(cfgFile)
			if err != nil {
				return fmt.Errorf("loading config from %s: %w", cfgFile, err)
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			cli, err := iam.New(ctx, cfg.Endpoint, cfg.Region, cfg.AccessKeyID, cfg.SecretAccessKey)
			if err != nil {
				return fmt.Errorf("can't create IAM client: %w", err)
			}

			list, err := cli.Keys.ListKeys(ctx, &iamv1.ListKeysReq{
				Count:  keyListCount,
				Page:   keyListPage,
				UserId: keyListUserID,
			})
			if err != nil {
				return fmt.Errorf("can't list keys: %w", err)
			}

			if keyListJSON {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(list)
			}

			tw := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', tabwriter.StripEscape|tabwriter.TabIndent)
			fmt.Fprintln(tw, "Access Key ID\tComment\tCreated\tUpdated\tDisabled\t")
			for _, k := range list.Keys {
				created := k.GetCreatedAt().AsTime().Format(time.RFC3339)
				updated := k.GetUpdatedAt().AsTime().Format(time.RFC3339)
				disabled := ""
				if d := k.GetDisabledAt(); d != nil {
					disabled = d.AsTime().Format(time.RFC3339)
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t\n", k.GetAccessKeyId(), k.GetComment(), created, updated, disabled)
			}
			tw.Flush()
			fmt.Fprintln(os.Stdout)

			return nil
		},
	}
)

func init() {
	rootCmd.AddCommand(keysCmd)
	keysCmd.AddCommand(keyCreateCmd)
	keysCmd.AddCommand(keyDisableCmd)
	keysCmd.AddCommand(keyListCmd)

	kcf := keyCreateCmd.Flags()
	kcf.StringVar(&keyCreateComment, "comment", "", "Comment describing the newly created key")
	kcf.StringVar(&keyCreateUserID, "user-id", "", "Optional user ID to create the key for")
	keyCreateCmd.MarkFlagRequired("comment")

	kdf := keyDisableCmd.Flags()
	kdf.StringVar(&keyDisableUserID, "user-id", "", "Optional user ID to scope the disable to")

	klf := keyListCmd.Flags()
	klf.Int32VarP(&keyListCount, "count", "c", 40, "maximum keys per page")
	klf.Int32VarP(&keyListPage, "page", "p", 0, "key list page to query")
	klf.BoolVarP(&keyListJSON, "json", "j", false, "if true, format result as JSON")
	klf.StringVar(&keyListUserID, "user-id", "", "Optional user ID to scope the list to")
}
