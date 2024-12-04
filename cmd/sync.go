package cmd

import (
	"crypto/tls"
	"fmt"
	"net/http"
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
$ helm datarobot sync testdata/test-chart1/ -r registry.example.com -u reg_username -p reg_password

Pulling image: docker.io/datarobot/test-image1:1.0.0
Pushing image: registry.example.com/datarobot/test-image1:1.0.0

'''`, "'", "`", -1),
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
			if syncRefPrefix != "" {
				if iUri.Organization == "" {
					iUri.Organization = syncRefPrefix
				} else {
					iUri.Organization = strings.Join([]string{syncRefPrefix, iUri.Organization}, "/")
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

			transport := http.DefaultTransport
			if skipTlsVerify {
				transport = &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
				}
			}

			cmd.Printf("Pushing image: %s\n\n", dstImage)
			var auth authn.Authenticator

			if syncToken != "" {
				auth = &authn.Bearer{
					Token: syncToken,
				}

			} else if syncUsername != "" && syncPassword != "" {
				auth = &authn.Basic{
					Username: syncUsername,
					Password: syncPassword,
				}

			} else {
				auth = authn.Anonymous
			}

			if err := crane.Push(img, dstImage, crane.WithTransport(transport), crane.WithAuth(auth)); err != nil {
				return fmt.Errorf("failed to push image with authentication: %w", err)
			}

		}
		return nil
	},
}

var syncReg, syncUsername, syncPassword, syncToken, syncRefPrefix string
var syncDryRun, skipTlsVerify bool

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.Flags().StringVarP(&syncUsername, "username", "u", "", "username to auth")
	syncCmd.Flags().StringVarP(&syncPassword, "password", "p", "", "pass to auth")
	syncCmd.Flags().StringVarP(&syncToken, "token", "t", "", "pass to auth")
	syncCmd.Flags().StringVarP(&syncReg, "registry", "r", "", "registry to auth")
	syncCmd.Flags().StringVarP(&syncRefPrefix, "prefix", "", "", "append prefix on repo name")
	syncCmd.Flags().BoolVarP(&syncDryRun, "dry-run", "", false, "Perform a dry run without making changes")
	syncCmd.Flags().BoolVarP(&skipTlsVerify, "skip-tls-verify", "", false, "Ignore SSL certificate verification (optional)")
	syncCmd.MarkFlagRequired("registry")
}
