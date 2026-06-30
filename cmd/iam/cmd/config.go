package cmd

import (
	"fmt"
	"os"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/spf13/cobra"
)

type Config struct {
	Endpoint        string `hcl:"endpoint"`
	Region          string `hcl:"region"`
	AccessKeyID     string `hcl:"access_key_id"`
	SecretAccessKey string `hcl:"secret_access_key"`
}

func loadConfig(fname string) (*Config, error) {
	var result Config

	if err := hclsimple.DecodeFile(fname, nil, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

var (
	configCmd = &cobra.Command{
		Use:     "config <save|show>",
		Short:   "Configuration management",
		Aliases: []string{"c"},
	}

	configSaveCmd = &cobra.Command{
		Use:   "save <--endpoint=> <--region=> <--access-key-id=> <--secret-access-key=>",
		Short: "Save configuration values to disk",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := Config{
				Endpoint:        configSaveEndpoint,
				Region:          configSaveRegion,
				AccessKeyID:     configSaveAccessKeyID,
				SecretAccessKey: configSaveSecretAccessKey,
			}

			f := hclwrite.NewEmptyFile()
			gohcl.EncodeIntoBody(&c, f.Body())

			if err := os.WriteFile(cfgFile, f.Bytes(), 0600); err != nil {
				return fmt.Errorf("writing config to %s: %w", cfgFile, err)
			}

			return nil
		},
	}

	configSaveEndpoint, configSaveRegion, configSaveAccessKeyID, configSaveSecretAccessKey string

	configShowCmd = &cobra.Command{
		Use:   "show",
		Short: "Show active configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := loadConfig(cfgFile)
			if err != nil {
				return fmt.Errorf("loading config from %s: %w", cfgFile, err)
			}

			fmt.Printf("endpoint:          %s\n", c.Endpoint)
			fmt.Printf("region:            %s\n", c.Region)
			fmt.Printf("access_key_id:     %s\n", c.AccessKeyID)
			fmt.Printf("secret_access_key: %s\n", c.SecretAccessKey)
			return nil
		},
	}
)

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configSaveCmd)
	configCmd.AddCommand(configShowCmd)

	// config save
	csf := configSaveCmd.Flags()
	csf.StringVar(&configSaveEndpoint, "endpoint", "", "endpoint URL for the IAM service")
	csf.StringVar(&configSaveRegion, "region", "yow", "region for the IAM service")
	csf.StringVar(&configSaveAccessKeyID, "access-key-id", "", "access key id")
	csf.StringVar(&configSaveSecretAccessKey, "secret-access-key", "", "secret access key")
	configSaveCmd.MarkFlagRequired("endpoint")
	configSaveCmd.MarkFlagRequired("region")
	configSaveCmd.MarkFlagRequired("access-key-id")
	configSaveCmd.MarkFlagRequired("secret-access-key")
}
