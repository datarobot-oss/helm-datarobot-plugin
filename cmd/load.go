package cmd

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/datarobot-oss/helm-datarobot-plugin/pkg/image_uri"
	"github.com/datarobot-oss/helm-datarobot-plugin/pkg/logger"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
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
		ctx := context.Background()
		if err := envconfig.Process(ctx, &loadCfg); err != nil {
			return fmt.Errorf("%v", err)
		}

		if loadCfg.Debug {
			logger.SetLevel(logger.DEBUG)
		}

		if loadCfg.DryRun {
			logger.SetPrefix("[Dry-Run] ")
		}

		tarballPath := args[0]
		// Step 1: Extract Tarball
		err := extractTarball(tarballPath, loadCfg.OutputDir)
		if err != nil {
			fmt.Printf("Error extracting tarball: %v\n", err)
			return nil
		}

		// Step 2: Read Manifest
		manifestPath := filepath.Join(loadCfg.OutputDir, "manifest.json")
		manifests, err := readManifest(manifestPath)
		if err != nil {
			fmt.Printf("Error reading manifest: %v\n", err)
			os.RemoveAll(loadCfg.OutputDir)
			return nil
		}

		// Step 3: Rebuild and Push Images
		for _, manifest := range manifests {
			if len(loadCfg.ImageSkip) > 0 {
				logger.Debug("Checking if image %s should be skipped", manifest.ImageName)
				_skipImage := false
				for _, imageSkip := range loadCfg.ImageSkip {
					if manifest.ImageName == imageSkip {
						logger.Info("Skipping image: %s\n", manifest.ImageName)
						_skipImage = true
						break
					}
				}
				if _skipImage {
					continue
				}
			}
			imageUri, err := rebuildAndPushImage(manifest, loadCfg, cmd)
			if err != nil {
				return fmt.Errorf("Error processing image %s: %v\n", manifest.OriginalImage, err)
			} else {
				logger.Info("Successfully pushed image: %s\n", imageUri)
			}
		}

		err = os.RemoveAll(loadCfg.OutputDir)
		if err != nil {
			return fmt.Errorf("Error Tmp Folder: %v\n", err)
		}
		return nil
	},
}

type loadConfig struct {
	Username      string   `env:"REGISTRY_USERNAME"`
	Password      string   `env:"REGISTRY_PASSWORD"`
	Token         string   `env:"REGISTRY_TOKEN"`
	RegistryHost  string   `env:"REGISTRY_HOST"`
	ImagePrefix   string   `env:"IMAGE_PREFIX"`
	ImageSuffix   string   `env:"IMAGE_SUFFIX"`
	ImageRepo     string   `env:"IMAGE_REPO"`
	CaCertPath    string   `env:"CA_CERT_PATH"`
	CertPath      string   `env:"CERT_PATH"`
	KeyPath       string   `env:"KEY_PATH"`
	OutputDir     string   `env:"OUTPUT_DIR"`
	ImageSkip     []string `env:"IMAGE_SKIP"`
	SkipTlsVerify bool     `env:"SKIP_TLS_VERIFY"`
	Overwrite     bool     `env:"OVERWRITE"`
	DryRun        bool     `env:"DRY_RUN"`
	Debug         bool     `env:"DEBUG"`
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
	loadCmd.Flags().StringVarP(&loadCfg.OutputDir, "output-dir", "", "export", "file to save")
	loadCmd.Flags().StringVarP(&loadCfg.CaCertPath, "ca-cert", "c", "", "Path to the custom CA certificate")
	loadCmd.Flags().StringVarP(&loadCfg.CertPath, "cert", "C", "", "Path to the client certificate")
	loadCmd.Flags().StringVarP(&loadCfg.KeyPath, "key", "K", "", "Path to the client key")
	loadCmd.Flags().BoolVarP(&loadCfg.SkipTlsVerify, "insecure", "i", false, "Skip server certificate verification")
	loadCmd.Flags().BoolVarP(&loadCfg.Overwrite, "overwrite", "", false, "Overwrite existing images")
	loadCmd.Flags().BoolVarP(&loadCfg.DryRun, "dry-run", "", false, "Perform a dry run without making changes")
	loadCmd.Flags().StringArrayVarP(&loadCfg.ImageSkip, "skip-image", "", []string{}, "Specify which image should be skipped (can be used multiple times)")
	loadCmd.Flags().BoolVarP(&loadCfg.Debug, "debug", "", false, "Enable debug mode")
}

func extractTarball(tarballPath, outputDir string) error {
	logger.Debug("Extracting tarball %s to %s", tarballPath, outputDir)

	// Open the tarball
	file, err := os.Open(tarballPath)
	if err != nil {
		return fmt.Errorf("error opening tarball: %v", err)
	}
	defer file.Close()

	zstdReader, err := zstd.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create zstd reader: %v", err)
	}
	defer zstdReader.Close()

	// Create a tar reader
	tarReader := tar.NewReader(zstdReader)

	// Extract files
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return fmt.Errorf("error reading tar header: %v", err)
		}

		// Determine the output path
		cleanName := filepath.Clean(header.Name)
		if strings.Contains(cleanName, "..") {
			return fmt.Errorf("invalid file path: %s", header.Name)
		}
		outputPath := filepath.Join(outputDir, cleanName)

		// Handle directories
		if header.Typeflag == tar.TypeDir {
			os.MkdirAll(outputPath, 0755)
			continue
		}

		// Handle files
		if header.Typeflag == tar.TypeReg {
			os.MkdirAll(filepath.Dir(outputPath), 0755)
			outFile, err := os.Create(outputPath)
			if err != nil {
				return fmt.Errorf("error creating file %s: %v", outputPath, err)
			}
			defer outFile.Close()

			_, err = io.Copy(outFile, tarReader)
			if err != nil {
				return fmt.Errorf("error writing file %s: %v", outputPath, err)
			}
		}
	}

	// fmt.Println("Extraction complete.")
	return nil
}

func readManifest(manifestPath string) ([]ImageManifest, error) {
	logger.Debug("Reading manifest from %s", manifestPath)

	file, err := os.Open(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("error opening manifest file: %v", err)
	}
	defer file.Close()

	var manifests []ImageManifest
	err = json.NewDecoder(file).Decode(&manifests)
	if err != nil {
		return nil, fmt.Errorf("error decoding manifest: %v", err)
	}

	// fmt.Println("Manifest successfully read.")
	return manifests, nil
}

func rebuildAndPushImage(manifest ImageManifest, c loadConfig, cmd *cobra.Command) (string, error) {
	logger.Debug("Rebuilding and pushing image: %s", manifest.OriginalImage)

	targetRef := fmt.Sprintf("%s/%s", c.RegistryHost, manifest.ImageName)
	iUri, err := image_uri.NewDockerUri(targetRef)
	if err != nil {
		return "", err
	}

	iUri.Organization = iUri.Join([]string{c.ImagePrefix, iUri.Organization}, "/")
	iUri.Project = iUri.Join([]string{iUri.Project, c.ImageSuffix}, "/")
	if c.ImageRepo != "" {
		iUri.Organization = c.ImageRepo
		iUri.Project = ""
	}

	transport, err := GetTransport(c.CaCertPath, c.CertPath, c.KeyPath, c.SkipTlsVerify)
	if err != nil {
		return "", fmt.Errorf("failed to GetTransport: %w", err)
	}

	auth := authn.Anonymous
	if c.Token != "" {
		auth = &authn.Bearer{
			Token: c.Token,
		}
	}
	if c.Username != "" && c.Password != "" {
		auth = &authn.Basic{
			Username: c.Username,
			Password: c.Password,
		}
	}
	if c.DryRun {
		return iUri.String(), nil
	}

	if !c.Overwrite {
		mfs, _ := crane.Manifest(iUri.String(), crane.WithTransport(transport), crane.WithAuth(auth))
		if len(mfs) > 0 {
			logger.Info("image %s already exists in the registry", iUri.String())
			return iUri.String(), nil
		}
	}

	// Step 1: Load Config File
	configPath := filepath.Join(c.OutputDir, manifest.ConfigFile)
	configFile, err := loadConfigFile(configPath)
	if err != nil {
		return "", fmt.Errorf("error loading config file %s: %v", configPath, err)
	}

	// Step 2: Load Layers
	var layers []v1.Layer
	for _, layerDigest := range manifest.Layers {
		layerPath := filepath.Join(c.OutputDir, "layers", layerDigest+".tar.gz")
		layer, err := tarball.LayerFromFile(layerPath)
		if err != nil {
			return "", fmt.Errorf("error loading layer %s: %v", layerPath, err)
		}
		layers = append(layers, layer)
	}

	// Step 3: Rebuild the Image
	emptyImage := empty.Image
	image, err := mutate.ConfigFile(emptyImage, configFile)
	if err != nil {
		return "", fmt.Errorf("error setting config file: %v", err)
	}
	image, err = mutate.AppendLayers(image, layers...)
	if err != nil {
		return "", fmt.Errorf("error appending layers: %v", err)
	}

	// Ensure the RootFS.DiffIDs match the layers
	var diffIDs []v1.Hash
	for _, layer := range layers {
		diffID, err := layer.DiffID()
		if err != nil {
			return "", fmt.Errorf("error getting layer DiffID: %v", err)
		}
		diffIDs = append(diffIDs, diffID)
	}

	configFile.RootFS = v1.RootFS{
		Type:    "layers",
		DiffIDs: diffIDs,
	}

	image, err = mutate.ConfigFile(image, configFile)
	if err != nil {
		return "", fmt.Errorf("error updating config file with RootFS: %v", err)
	}

	// Push each layer individually to ensure they are available in the registry
	for _, layer := range layers {
		layerDigest, err := layer.Digest()
		if err != nil {
			return "", fmt.Errorf("error getting layer digest: %v", err)
		}
		layerPath := filepath.Join(c.OutputDir, "layers", strings.TrimPrefix(layerDigest.String(), "sha256:")+".tar.gz")
		layerFile, err := os.Open(layerPath)
		if err != nil {
			return "", fmt.Errorf("error opening layer file %s: %v", layerPath, err)
		}
		defer layerFile.Close()

		layer, err := tarball.LayerFromReader(layerFile)
		if err != nil {
			return "", fmt.Errorf("error creating layer from file %s: %v", layerPath, err)
		}

		repo, err := name.NewRepository(iUri.Base())
		if err != nil {
			return "", fmt.Errorf("error creating repository from URI %s: %v", iUri.String(), err)
		}
		err = remote.WriteLayer(repo, layer, remote.WithTransport(transport), remote.WithAuth(auth))
		if err != nil {
			return "", fmt.Errorf("error pushing layer %s: %v", layerDigest.String(), err)
		}
	}

	// Push the final image manifest
	err = crane.Push(image, iUri.String(), crane.WithTransport(transport), crane.WithAuth(auth))
	if err != nil {
		return "", fmt.Errorf("error pushing image: %v", err)
	} else {
		return iUri.String(), nil
	}
}

func loadConfigFile(configPath string) (*v1.ConfigFile, error) {
	logger.Debug("Loading config file %s", configPath)

	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("error opening config file: %v", err)
	}
	defer file.Close()

	var configFile v1.ConfigFile
	err = json.NewDecoder(file).Decode(&configFile)
	if err != nil {
		return nil, fmt.Errorf("error decoding config file: %v", err)
	}

	return &configFile, nil
}
