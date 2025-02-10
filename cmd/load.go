package cmd

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/datarobot-oss/helm-datarobot-plugin/pkg/image_uri"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/klauspost/compress/zstd"
	"github.com/sethvargo/go-envconfig"
	"github.com/spf13/cobra"
)

var loadCmd = &cobra.Command{
	Use:          "load",
	Short:        "load all images from a tgz file to a specific registry",
	SilenceUsage: true,
	Long: strings.Replace(`

This command is designed to load all images from a tgz file to a specific registry

Example:
'''sh
$ helm datarobot load images.tgz -r registry.example.com -u reg_username -p reg_password
Successfully pushed image: registry.example.com/test-image1:1.0.0

'''

Authentication can be provided in various ways, including:

'''sh
export REGISTRY_USERNAME=reg_username
export REGISTRY_PASSWORD=reg_password
export REGISTRY_HOST=registry.example.com
$ helm datarobot load images.tgz
'''
`, "'", "`", -1),
	Args: cobra.MinimumNArgs(1), // Requires at least one argument (file path)
	RunE: func(cmd *cobra.Command, args []string) error {

		zstFile := args[0]
		// Open the tgz file
		file, err := os.Open(zstFile)
		if err != nil {
			return fmt.Errorf("failed to open tgz file %q: %v", zstFile, err)
		}
		defer file.Close()

		// Create a Zstandard reader
		zstdReader, err := zstd.NewReader(file)
		if err != nil {
			return fmt.Errorf("failed to create gzip reader: %v", err)
		}
		defer zstdReader.Close()

		// Create a new tar reader
		tarReader := tar.NewReader(zstdReader)

		for {
			// Read the next header from the tar archive
			header, err := tarReader.Next()
			if err == io.EOF {
				break // End of archive
			}
			if err != nil {
				return fmt.Errorf("failed to read tar header: %v", err)
			}

			// Create a temporary file to store the extracted tarball
			tempFile, err := os.CreateTemp("", "image-*.tar")
			if err != nil {
				return fmt.Errorf("failed to create temp file: %v", err)
			}
			defer os.Remove(tempFile.Name()) // Clean up temp file

			// Copy the tarball content to the temp file
			_, err = io.Copy(tempFile, tarReader)
			if err != nil {
				tempFile.Close()
				return fmt.Errorf("failed to copy tarball content: %v", err)
			}
			tempFile.Close()

			// Load the Docker image from the tarball
			image, err := tarball.ImageFromPath(tempFile.Name(), nil)
			if err != nil {
				return fmt.Errorf("failed to load Docker image from tarball: %v", err)
			}

			imageName := loadCfg.RegistryHost + "/" + strings.TrimSuffix(header.Name, ".tgz")
			iUri, err := image_uri.NewDockerUri(imageName)
			if err != nil {
				return err
			}

			iUri.Organization = iUri.Join([]string{loadCfg.ImagePrefix, iUri.Organization}, "/")
			iUri.Project = iUri.Join([]string{iUri.Project, loadCfg.ImageSuffix}, "/")
			if loadCfg.ImageRepo != "" {
				iUri.Organization = loadCfg.ImageRepo
				iUri.Project = ""
			}

			if loadCfg.DryRun {
				cmd.Printf("[Dry-Run] Pushing image: %s\n", iUri.String())
				continue
			}

			ref, err := name.NewTag(iUri.String())
			if err != nil {
				return fmt.Errorf("failed to create image reference: %v", err)
			}

			transport, err := GetTransport(loadCfg.CaCertPath, loadCfg.CertPath, loadCfg.KeyPath, loadCfg.SkipTlsVerify)
			if err != nil {
				return fmt.Errorf("failed to GetTransport: %w", err)
			}

			auth := authn.Anonymous
			if loadCfg.Token != "" {
				auth = &authn.Bearer{
					Token: loadCfg.Token,
				}
			}
			if loadCfg.Username != "" && loadCfg.Password != "" {
				auth = &authn.Basic{
					Username: loadCfg.Username,
					Password: loadCfg.Password,
				}
			}

			err = remote.Write(ref, image, remote.WithTransport(transport), remote.WithAuth(auth))
			if err != nil {
				return fmt.Errorf("failed to push Docker image to registry: %v", err)
			} else {
				cmd.Printf("Successfully pushed image %s\n", ref.Name())
			}
		}

		return nil
	},
}

type loadConfig struct {
	Username      string `env:"REGISTRY_USERNAME"`
	Password      string `env:"REGISTRY_PASSWORD"`
	Token         string `env:"REGISTRY_TOKEN"`
	RegistryHost  string `env:"REGISTRY_HOST"`
	ImagePrefix   string `env:"IMAGE_PREFIX"`
	ImageSuffix   string `env:"IMAGE_SUFFIX"`
	ImageRepo     string `env:"IMAGE_REPO"`
	CaCertPath    string `env:"CA_CERT_PATH"`
	CertPath      string `env:"CERT_PATH"`
	KeyPath       string `env:"KEY_PATH"`
	SkipTlsVerify bool   `env:"SKIP_TLS_VERIFY"`
	DryRun        bool   `env:"DRY_RUN"`
}

var loadCfg loadConfig

func init() {
	rootCmd.AddCommand(loadCmd)
	loadCmd.Flags().StringVarP(&loadCfg.Username, "username", "u", "", "username to auth")
	loadCmd.Flags().StringVarP(&loadCfg.Password, "password", "p", "", "pass to auth")
	loadCmd.Flags().StringVarP(&loadCfg.Token, "token", "t", "", "pass to auth")
	loadCmd.Flags().StringVarP(&loadCfg.RegistryHost, "registry", "r", "", "registry to auth")
	loadCmd.Flags().StringVarP(&loadCfg.ImagePrefix, "prefix", "", "", "append prefix on repo name")
	loadCmd.Flags().StringVarP(&loadCfg.ImageRepo, "repo", "", "", "rewrite the target repository name")
	loadCmd.Flags().StringVarP(&loadCfg.ImageSuffix, "suffix", "", "", "append suffix on repo name")
	loadCmd.Flags().StringVarP(&loadCfg.CaCertPath, "ca-cert", "c", "", "Path to the custom CA certificate")
	loadCmd.Flags().StringVarP(&loadCfg.CertPath, "cert", "C", "", "Path to the client certificate")
	loadCmd.Flags().StringVarP(&loadCfg.KeyPath, "key", "K", "", "Path to the client key")
	loadCmd.Flags().BoolVarP(&loadCfg.SkipTlsVerify, "insecure", "i", false, "Skip server certificate verification")
	loadCmd.Flags().BoolVarP(&loadCfg.DryRun, "dry-run", "", false, "Perform a dry run without making changes")
	ctx := context.Background()
	if err := envconfig.Process(ctx, &loadCfg); err != nil {
		log.Fatal(err)
	}
}
