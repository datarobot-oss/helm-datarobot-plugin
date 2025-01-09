package cmd

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/datarobot-oss/helm-datarobot-plugin/pkg/image_uri"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/spf13/cobra"
)

var loadCmd = &cobra.Command{
	Use:          "load",
	Short:        "load",
	SilenceUsage: true,
	Long: strings.Replace(`

This command is designed to load all images from a tgz file to a specific registry

Example:
'''sh
$ helm datarobot load images.tgz -r registry.example.com -u reg_username -p reg_password
Successfully pushed image: registry.example.com/test-image1:1.0.0

'''`, "'", "`", -1),
	Args: cobra.MinimumNArgs(1), // Requires at least one argument (file path)
	RunE: func(cmd *cobra.Command, args []string) error {

		tgzPath := args[0]
		// Open the tgz file
		file, err := os.Open(tgzPath)
		if err != nil {
			return fmt.Errorf("failed to open tgz file %q: %v", tgzPath, err)
		}
		defer file.Close()

		// Create a gzip reader
		gzipReader, err := gzip.NewReader(file)
		if err != nil {
			return fmt.Errorf("failed to create gzip reader: %v", err)
		}
		defer gzipReader.Close()

		// Create a tar reader
		tarReader := tar.NewReader(gzipReader)

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

			// Push the image to the specified registry
			tag := filepath.Base(header.Name) // Use the tarball name as the image tag
			tag = strings.TrimSuffix(tag, ".tgz")

			imageName := loadReg + "/" + loadImagePrefix + "/" + tag
			iUri, err := image_uri.NewDockerUri(imageName)
			if err != nil {
				return err
			}

			iUri.Project = iUri.Join([]string{iUri.Project, loadImageSuffix}, "/")
			if loadImageRepo != "" {
				iUri.Organization = loadImageRepo
				iUri.Project = ""
			}

			if loadDryRun {
				cmd.Printf("[Dry-Run] Pushing image: %s\n", iUri.String())
				continue
			}

			ref, err := name.NewTag(iUri.String())
			if err != nil {
				return fmt.Errorf("failed to create image reference: %v", err)
			}

			transport, err := GetTransport(caCertPath, certPath, keyPath, skipTlsVerify)
			if err != nil {
				return fmt.Errorf("failed to GetTransport: %w", err)
			}

			var auth authn.Authenticator

			if loadToken != "" {
				auth = &authn.Bearer{
					Token: loadToken,
				}

			} else if loadUsername != "" && loadPassword != "" {
				auth = &authn.Basic{
					Username: loadUsername,
					Password: loadPassword,
				}

			} else {
				auth = authn.Anonymous
			}

			err = remote.Write(ref, image, remote.WithTransport(transport), remote.WithAuth(auth))
			if err != nil {
				return fmt.Errorf("failed to push Docker image to registry: %v", err)
			}
			cmd.Printf("Successfully pushed image %s\n", ref.Name())
		}

		return nil
	},
}

var loadReg, loadUsername, loadPassword, loadToken, loadImagePrefix, loadImageSuffix, loadImageRepo string
var loadDryRun bool

func init() {
	rootCmd.AddCommand(loadCmd)
	loadCmd.Flags().StringVarP(&loadUsername, "username", "u", "", "username to auth")
	loadCmd.Flags().StringVarP(&loadPassword, "password", "p", "", "pass to auth")
	loadCmd.Flags().StringVarP(&loadToken, "token", "t", "", "pass to auth")
	loadCmd.Flags().StringVarP(&loadReg, "registry", "r", "", "registry to auth")
	loadCmd.Flags().StringVarP(&loadImagePrefix, "prefix", "", "", "append prefix on repo name")
	loadCmd.Flags().StringVarP(&loadImageRepo, "repo", "", "", "rewrite the target repository name")
	loadCmd.Flags().StringVarP(&loadImageSuffix, "suffix", "", "", "append suffix on repo name")
	loadCmd.Flags().StringVarP(&caCertPath, "ca-cert", "c", "", "Path to the custom CA certificate")
	loadCmd.Flags().StringVarP(&certPath, "cert", "C", "", "Path to the client certificate")
	loadCmd.Flags().StringVarP(&keyPath, "key", "K", "", "Path to the client key")
	loadCmd.Flags().BoolVarP(&skipTlsVerify, "insecure", "i", false, "Skip server certificate verification")
	loadCmd.Flags().BoolVarP(&loadDryRun, "dry-run", "", false, "Perform a dry run without making changes")
	loadCmd.MarkFlagRequired("registry")
}
