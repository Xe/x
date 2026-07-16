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
	usersCmd = &cobra.Command{
		Use:     "users <command>",
		Short:   "User management",
		Aliases: []string{"u"},
	}

	userCreateName string

	userCreateCmd = &cobra.Command{
		Use:     "create <flags>",
		Short:   "Create a new IAM user",
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

			resp, err := cli.Users.CreateUser(ctx, &iamv1.CreateUserReq{
				Name: userCreateName,
			})
			if err != nil {
				return err
			}

			u := resp.GetUser()
			fmt.Printf("ID:                %s\n", u.GetId())
			fmt.Printf("Name:              %s\n", u.GetName())
			fmt.Printf("Access key ID:     %s\n", resp.GetAccessKeyId())
			fmt.Printf("Secret access key: %s\n", resp.GetSecretAccessKey())

			return nil
		},
	}

	userDisableCmd = &cobra.Command{
		Use:   "disable <user-id> <reason>",
		Short: "Disable an IAM user",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				return fmt.Errorf("please supply a user ID and reason in args")
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

			id := args[0]
			reason := args[1]

			if _, err := cli.Users.DisableUser(ctx, &iamv1.DisableUserReq{
				Id:     id,
				Reason: reason,
			}); err != nil {
				return err
			}

			fmt.Println("user", args[0], "disabled, all keys are invalid")
			return nil
		},
	}

	userListCount, userListPage int32
	userListJSON                bool

	userListCmd = &cobra.Command{
		Use:   "list <--count=> <--page=> [--json]",
		Short: "List IAM users",
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

			list, err := cli.Users.ListUsers(ctx, &iamv1.ListUsersReq{
				Count: userListCount,
				Page:  userListPage,
			})
			if err != nil {
				return fmt.Errorf("can't list users: %w", err)
			}

			if userListJSON {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(list)
			}

			tw := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', tabwriter.StripEscape|tabwriter.TabIndent)
			fmt.Fprintln(tw, "ID\tName\tCreated\tUpdated\t")
			for _, u := range list.Users {
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t\n", u.GetId(), u.GetName(), u.GetCreatedAt().AsTime().Format(time.RFC3339), u.GetUpdatedAt().AsTime().Format(time.RFC3339))
			}
			tw.Flush()
			fmt.Fprintln(os.Stdout)

			return nil
		},
	}
)

func init() {
	rootCmd.AddCommand(usersCmd)
	usersCmd.AddCommand(userCreateCmd)
	usersCmd.AddCommand(userDisableCmd)
	usersCmd.AddCommand(userListCmd)

	ucf := userCreateCmd.Flags()
	ucf.StringVar(&userCreateName, "name", "", "Name to give the newly created user")
	userCreateCmd.MarkFlagRequired("username")

	ulf := userListCmd.Flags()
	ulf.Int32VarP(&userListCount, "count", "c", 40, "maximum users per page")
	ulf.Int32VarP(&userListPage, "page", "p", 0, "user list page to query")
	ulf.BoolVarP(&userListJSON, "json", "j", false, "if true, format result as JSON")
}
