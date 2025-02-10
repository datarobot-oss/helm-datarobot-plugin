package cmd

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/datarobot-oss/helm-datarobot-plugin/pkg/image_uri"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/sethvargo/go-envconfig"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:          "sync",
	Aliases:      []string{"sy"},
	Short:        "sync",
	SilenceUsage: true,
	Long: strings.Replace(`

This command is designed to sync directly all images as part of the release manifest to a registry

Example:
'''sh
$ helm datarobot sync tests/charts/test-chart1/ -r registry.example.com -u reg_username -p reg_password

Pulling image: docker.io/datarobot/test-image1:1.0.0
Pushing image: registry.example.com/datarobot/test-image1:1.0.0
'''

Authentication can be provided in various ways, including:

'''sh
export REGISTRY_USERNAME=reg_username
export REGISTRY_PASSWORD=reg_password
export REGISTRY_HOST=registry.example.com
$ helm datarobot sync tests/charts/test-chart1/
'''
`, "'", "`", -1),
	RunE: func(cmd *cobra.Command, args []string) error {

		if syncCfg.RegistryHost == "" {
			return fmt.Errorf("Registry Host not set")
		}

		images, err := ExtractImagesFromCharts(args)
		if err != nil {
			return fmt.Errorf("Error ExtractImagesFromCharts: %v", err)
		}

		for _, image := range images {
			iUri, err := image_uri.NewDockerUri(image.Image)
			if err != nil {
				return err
			}
			srcImage := iUri.String()
			if image.Tag != "" {
				iUri.Tag = image.Tag
			}

			iUri.RegistryHost = syncCfg.RegistryHost
			iUri.Organization = iUri.Join([]string{syncCfg.ImagePrefix, iUri.Organization}, "/")
			iUri.Project = iUri.Join([]string{iUri.Project, syncCfg.ImageSuffix}, "/")
			if syncCfg.ImageRepo != "" {
				iUri.Organization = syncCfg.ImageRepo
				iUri.Project = ""
			}

			if len(syncCfg.ImageSkipGroup) > 0 {
				_skipImage := false
				for _, group := range syncCfg.ImageSkipGroup {
					if image.Group == group {
						cmd.Printf("Skipping image: %s\n\n", srcImage)
						_skipImage = true
						continue
					}
				}
				if _skipImage {
					continue
				}
			}

			dstImage := iUri.String()
			if syncCfg.DryRun {
				cmd.Printf("[Dry-Run] Pulling image: %s\n", srcImage)
				cmd.Printf("[Dry-Run] Pushing image: %s\n\n", dstImage)
				continue
			}

			cmd.Printf("Pulling image: %s\n", srcImage)
			img, err := crane.Pull(srcImage)
			if err != nil {
				return fmt.Errorf("failed to pull image: %w", err)
			}

			transport, err := GetTransport(syncCfg.CaCertPath, syncCfg.CertPath, syncCfg.KeyPath, syncCfg.SkipTlsVerify)
			if err != nil {
				return fmt.Errorf("failed to GetTransport: %w", err)
			}

			cmd.Printf("Pushing image: %s\n\n", dstImage)

			auth := authn.Anonymous
			if syncCfg.Token != "" {
				auth = &authn.Bearer{
					Token: syncCfg.Token,
				}
			}
			if syncCfg.Username != "" && syncCfg.Password != "" {
				auth = &authn.Basic{
					Username: syncCfg.Username,
					Password: syncCfg.Password,
				}
			}

			if err := crane.Push(img, dstImage, crane.WithTransport(transport), crane.WithAuth(auth)); err != nil {
				return fmt.Errorf("failed to push image with authentication: %w", err)
			}

		}
		return nil
	},
}

type syncConfig struct {
	Username       string   `env:"REGISTRY_USERNAME"`
	Password       string   `env:"REGISTRY_PASSWORD"`
	Token          string   `env:"REGISTRY_TOKEN"`
	RegistryHost   string   `env:"REGISTRY_HOST"`
	ImagePrefix    string   `env:"IMAGE_PREFIX"`
	ImageSuffix    string   `env:"IMAGE_SUFFIX"`
	ImageRepo      string   `env:"IMAGE_REPO"`
	Transform      string   `env:"TRANSFORM"`
	CaCertPath     string   `env:"CA_CERT_PATH"`
	CertPath       string   `env:"CERT_PATH"`
	KeyPath        string   `env:"KEY_PATH"`
	SkipTlsVerify  bool     `env:"SKIP_TLS_VERIFY"`
	ImageSkipGroup []string `env:"IMAGE_SKIP_GROUP"`
	DryRun         bool     `env:"DRY_RUN"`
}

var syncCfg syncConfig

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.Flags().StringVarP(&annotation, "annotation", "a", "datarobot.com/images", "annotation to lookup")
	syncCmd.Flags().StringVarP(&syncCfg.Username, "username", "u", "", "username to auth")
	syncCmd.Flags().StringVarP(&syncCfg.Password, "password", "p", "", "pass to auth")
	syncCmd.Flags().StringVarP(&syncCfg.Token, "token", "t", "", "pass to auth")
	syncCmd.Flags().StringVarP(&syncCfg.RegistryHost, "registry", "r", "", "registry to auth")
	syncCmd.Flags().StringVarP(&syncCfg.ImagePrefix, "prefix", "", "", "append prefix on repo name")
	syncCmd.Flags().StringVarP(&syncCfg.ImageRepo, "repo", "", "", "rewrite the target repository name")
	syncCmd.Flags().StringVarP(&syncCfg.ImageSuffix, "suffix", "", "", "append suffix on repo name")
	syncCmd.Flags().BoolVarP(&syncCfg.DryRun, "dry-run", "", false, "Perform a dry run without making changes")
	syncCmd.Flags().StringVarP(&syncCfg.CaCertPath, "ca-cert", "c", "", "Path to the custom CA certificate")
	syncCmd.Flags().StringVarP(&syncCfg.CertPath, "cert", "C", "", "Path to the client certificate")
	syncCmd.Flags().StringVarP(&syncCfg.KeyPath, "key", "K", "", "Path to the client key")
	syncCmd.Flags().BoolVarP(&syncCfg.SkipTlsVerify, "insecure", "i", false, "Skip server certificate verification")
	syncCmd.Flags().StringArrayVarP(&syncCfg.ImageSkipGroup, "skip-group", "", []string{}, "Specify which image group should be skipped (can be used multiple times)")
	ctx := context.Background()
	if err := envconfig.Process(ctx, &syncCfg); err != nil {
		log.Fatal(err)
	}
}
