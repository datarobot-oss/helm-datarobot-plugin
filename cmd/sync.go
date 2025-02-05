package cmd

import (
	"fmt"
	"strings"

	"github.com/datarobot-oss/helm-datarobot-plugin/pkg/image_uri"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
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
$ helm datarobot sync tests/charts/test-chart1/ -r registry.example.com
'''

'''sh
$ echo "reg_password" | helm datarobot sync tests/charts/test-chart1/ -r registry.example.com -u reg_username --password-stdin
'''

`, "'", "`", -1),
	RunE: func(cmd *cobra.Command, args []string) error {
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

			iUri.RegistryHost = syncReg
			iUri.Organization = iUri.Join([]string{syncImagePrefix, iUri.Organization}, "/")
			iUri.Project = iUri.Join([]string{iUri.Project, syncImageSuffix}, "/")
			if syncImageRepo != "" {
				iUri.Organization = syncImageRepo
				iUri.Project = ""
			}

			if len(syncSkipImageGroup) > 0 {
				_skipImage := false
				for _, group := range syncSkipImageGroup {
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
			if syncDryRun {
				cmd.Printf("[Dry-Run] Pulling image: %s\n", srcImage)
				cmd.Printf("[Dry-Run] Pushing image: %s\n\n", dstImage)
				continue
			}

			cmd.Printf("Pulling image: %s\n", srcImage)
			img, err := crane.Pull(srcImage)
			if err != nil {
				return fmt.Errorf("failed to pull image: %w", err)
			}

			transport, err := GetTransport(caCertPath, certPath, keyPath, skipTlsVerify)
			if err != nil {
				return fmt.Errorf("failed to GetTransport: %w", err)
			}

			cmd.Printf("Pushing image: %s\n\n", dstImage)

			auth := authn.Anonymous
			secretSyncToken := GetSecret(syncPasswordStdin, "REGISTRY_TOKEN", syncToken)
			if secretSyncToken != "" {
				auth = &authn.Bearer{
					Token: secretSyncToken,
				}
			}
			secretSyncUsername := GetSecret(false, "REGISTRY_USERNAME", syncUsername)
			secretSyncPassword := GetSecret(syncPasswordStdin, "REGISTRY_PASSWORD", syncPassword)
			if secretSyncUsername != "" && secretSyncPassword != "" {
				auth = &authn.Basic{
					Username: secretSyncUsername,
					Password: secretSyncPassword,
				}
			}

			if err := crane.Push(img, dstImage, crane.WithTransport(transport), crane.WithAuth(auth)); err != nil {
				return fmt.Errorf("failed to push image with authentication: %w", err)
			}

		}
		return nil
	},
}

var syncReg, syncUsername, syncPassword, syncToken, syncImagePrefix, syncImageSuffix, syncImageRepo, syncTransform, caCertPath, certPath, keyPath string
var syncDryRun, skipTlsVerify, syncPasswordStdin bool
var syncSkipImageGroup []string

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.Flags().StringVarP(&annotation, "annotation", "a", "datarobot.com/images", "annotation to lookup")
	syncCmd.Flags().StringVarP(&syncUsername, "username", "u", "", "username to auth")
	syncCmd.Flags().StringVarP(&syncPassword, "password", "p", "", "pass to auth")
	syncCmd.Flags().BoolVar(&syncPasswordStdin, "password-stdin", false, "Read password from stdin")
	syncCmd.Flags().StringVarP(&syncToken, "token", "t", "", "pass to auth")
	syncCmd.Flags().StringVarP(&syncReg, "registry", "r", "", "registry to auth")
	syncCmd.Flags().StringVarP(&syncImagePrefix, "prefix", "", "", "append prefix on repo name")
	syncCmd.Flags().StringVarP(&syncImageRepo, "repo", "", "", "rewrite the target repository name")
	syncCmd.Flags().StringVarP(&syncImageSuffix, "suffix", "", "", "append suffix on repo name")
	syncCmd.Flags().BoolVarP(&syncDryRun, "dry-run", "", false, "Perform a dry run without making changes")
	syncCmd.Flags().StringVarP(&caCertPath, "ca-cert", "c", "", "Path to the custom CA certificate")
	syncCmd.Flags().StringVarP(&certPath, "cert", "C", "", "Path to the client certificate")
	syncCmd.Flags().StringVarP(&keyPath, "key", "K", "", "Path to the client key")
	syncCmd.Flags().BoolVarP(&skipTlsVerify, "insecure", "i", false, "Skip server certificate verification")
	syncCmd.Flags().StringArrayVarP(&syncSkipImageGroup, "skip-group", "", []string{}, "Specify which image group should be skipped (can be used multiple times)")
	syncCmd.MarkFlagRequired("registry")
}
