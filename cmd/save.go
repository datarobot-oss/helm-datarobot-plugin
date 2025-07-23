package cmd

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zstd"

	"github.com/datarobot-oss/helm-datarobot-plugin/pkg/chartutil"
	"github.com/datarobot-oss/helm-datarobot-plugin/pkg/image_uri"
	"github.com/google/go-containerregistry/pkg/crane"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/spf13/cobra"
)

type ImageManifest struct {
	ImageName     string   `json:"image_name"`
	Layers        []string `json:"layers"`
	ConfigFile    string   `json:"config_file"`
	OriginalImage string   `json:"original_image"`
}

var saveCmd = &cobra.Command{
	Use:          "save",
	Short:        "save images in single tgz file",
	SilenceUsage: true,
	Long: strings.Replace(`

This command is designed to save all images as part of the release manifest in single tgz file

Example:
'''sh
$ helm datarobot save tests/charts/test-chart1/
Pulling image: docker.io/datarobot/test-image1:1.0.0
....
Pulling image: docker.io/datarobot/test-image2:2.0.0
....
Tarball created successfully: images.tar.zst
$ du -h images.tar.zst
14M    images.tar.zst

'''`, "'", "`", -1),
	Args: cobra.MinimumNArgs(1), // Requires at least one argument (file path)
	RunE: func(cmd *cobra.Command, args []string) error {

		levelMap := map[string]zstd.EncoderLevel{
			"fastest": zstd.SpeedFastest,
			"default": zstd.SpeedDefault,
			"better":  zstd.SpeedBetterCompression,
			"best":    zstd.SpeedBestCompression,
		}

		level, ok := levelMap[saveCfg.CompressionLevel]
		if !ok {
			return fmt.Errorf("Invalid compression level. Available options: fastest, default, better, best")
		}

		images, err := chartutil.ExtractImagesFromCharts(args, annotation)
		if err != nil {
			return fmt.Errorf("Error ExtractImagesFromCharts: %v", err)
		}

		manifestFile := filepath.Join(saveCfg.OutputDir, "manifest.json")

		// Step 1: Export Layers and Save Configurations
		_, manifests := exportLayersAndConfigs(images, saveCfg, cmd)

		// Step 2: Save Manifest
		err = saveManifest(manifestFile, manifests)
		if err != nil {
			return fmt.Errorf("Error saving manifest: %v\n", err)
		}
		// Step 3: Create a Tarball
		err = createTarball(saveCfg.Output, saveCfg.OutputDir, level)
		if err != nil {
			return fmt.Errorf("Error creating tarball: %v\n", err)
		}

		err = os.RemoveAll(saveCfg.OutputDir)
		if err != nil {
			return fmt.Errorf("Error Tmp Folder: %v\n", err)
		}

		if saveCfg.DryRun {
			cmd.Printf("[Dry-Run] Tarball created successfully: %s\n", saveCfg.Output)
		} else {
			cmd.Printf("Tarball created successfully: %s\n", saveCfg.Output)
		}

		return nil
	},
}

type saveConfig struct {
	Output           string   `env:"OUTPUT"`
	OutputDir        string   `env:"OUTPUT_DIR"`
	CompressionLevel string   `env:"LEVEL"`
	ImageSkipGroup   []string `env:"IMAGE_SKIP_GROUP"`
	DryRun           bool     `env:"DRY_RUN"`
}

var saveCfg saveConfig

func init() {
	rootCmd.AddCommand(saveCmd)
	saveCmd.Flags().StringVarP(&annotation, "annotation", "a", "datarobot.com/images", "annotation to lookup")
	saveCmd.Flags().StringVarP(&saveCfg.Output, "output", "o", "images.tar.zst", "file to save")
	saveCmd.Flags().StringVarP(&saveCfg.OutputDir, "output-dir", "", "export", "file to save")
	saveCmd.Flags().StringVarP(&saveCfg.CompressionLevel, "level", "l", "best", "zstd compression level (Available options: fastest, default, better, best)")
	saveCmd.Flags().StringArrayVarP(&saveCfg.ImageSkipGroup, "skip-group", "", []string{}, "Specify which image group should be skipped (can be used multiple times)")
	saveCmd.Flags().BoolVarP(&saveCfg.DryRun, "dry-run", "", false, "Perform a dry run without making changes")
}

func exportLayersAndConfigs(images []chartutil.DatarobotImageDeclaration, c saveConfig, cmd *cobra.Command) (map[string]string, []ImageManifest) {
	layerDir := filepath.Join(c.OutputDir, "layers")
	configDir := filepath.Join(c.OutputDir, "configs")

	// Create directories
	os.MkdirAll(layerDir, 0755)
	os.MkdirAll(configDir, 0755)

	layerFiles := make(map[string]string)
	var manifests []ImageManifest

	for _, i := range images {
		iUri, err := image_uri.NewDockerUri(i.Image)
		if err != nil {
			return nil, nil
		}

		if len(c.ImageSkipGroup) > 0 {
			_skipImage := false
			for _, group := range saveCfg.ImageSkipGroup {
				if i.Group == group {
					cmd.Printf("Skipping image: %s\n\n", iUri.String())
					_skipImage = true
					continue
				}
			}
			if _skipImage {
				continue
			}
		}

		if c.DryRun {
			cmd.Printf("[Dry-Run] Pulling image: %s\n", iUri.String())
			if i.Tag != "" {
				oldName := iUri.String()
				iUri.Tag = i.Tag
				cmd.Printf("[Dry-Run] ReTagging image: %s > %s\n", oldName, iUri.String())
			}
			continue
		}

		cmd.Printf("Pulling image: %s\n", iUri.String())

		// Pull the image
		image, err := crane.Pull(iUri.String())
		if err != nil {
			fmt.Printf("Error pulling image %s: %v\n", iUri.String(), err)
			continue
		}

		// Retrieve the configuration
		configFile, err := image.ConfigFile()
		if err != nil {
			fmt.Printf("Error retrieving config for image %s: %v\n", iUri.String(), err)
			continue
		}

		if i.Tag != "" {
			oldName := iUri.String()
			iUri.Tag = i.Tag
			cmd.Printf("ReTagging image: %s > %s\n", oldName, iUri.String())
		}

		// Save the ConfigFile
		configPath := filepath.Join(configDir, sanitizeFilename(iUri.String())+".config.json")
		saveConfigFile(configPath, configFile)

		// Get the layers
		layers, err := image.Layers()
		if err != nil {
			fmt.Printf("Error retrieving layers for image %s: %v\n", iUri.String(), err)
			continue
		}

		var layerDigests []string
		for idx, layer := range layers {
			digest, err := layer.Digest()
			if err != nil {
				fmt.Printf("Error getting digest for layer %d of image %s: %v\n", idx+1, iUri.String(), err)
				continue
			}

			layerFile := filepath.Join(layerDir, digest.Hex+".tar.gz")
			layerDigests = append(layerDigests, digest.Hex)

			// If layer is already saved, skip
			if _, exists := layerFiles[digest.Hex]; exists {
				continue
			}

			// Save the layer content
			layerReader, err := layer.Compressed()
			if err != nil {
				fmt.Printf("Error reading layer %d for image %s: %v\n", idx+1, iUri.String(), err)
				continue
			}
			defer layerReader.Close()

			saveLayerToFile(layerReader, layerFile)
			layerFiles[digest.Hex] = layerFile
		}

		// Add metadata to the manifest
		manifests = append(manifests, ImageManifest{
			ImageName:     iUri.RefName(),
			Layers:        layerDigests,
			ConfigFile:    filepath.Join("configs", sanitizeFilename(iUri.String())+".config.json"),
			OriginalImage: iUri.String(),
		})
	}

	return layerFiles, manifests
}

func saveConfigFile(filePath string, configFile *v1.ConfigFile) {
	file, err := os.Create(filePath)
	if err != nil {
		fmt.Printf("Error creating config file %s: %v\n", filePath, err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(configFile)
	if err != nil {
		fmt.Printf("Error writing config file %s: %v\n", filePath, err)
	}
}

func saveLayerToFile(reader io.Reader, filePath string) {
	file, err := os.Create(filePath)
	if err != nil {
		fmt.Printf("Error creating file %s: %v\n", filePath, err)
		return
	}
	defer file.Close()

	io.Copy(file, reader)
}

func saveManifest(filePath string, manifests []ImageManifest) error {
	// fmt.Printf("Saving manifest to %s...\n", filePath)

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("error creating manifest file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(manifests)
	if err != nil {
		return fmt.Errorf("error writing manifest: %v", err)
	}

	// fmt.Println("Manifest successfully saved.")
	return nil
}

func createTarball(outputTarball string, inputDir string, level zstd.EncoderLevel) error {
	// fmt.Printf("Creating tarball %s...\n", outputTarball)

	// Create the tar.gz file
	tarFile, err := os.Create(outputTarball)
	if err != nil {
		return fmt.Errorf("error creating tarball: %v", err)
	}
	defer tarFile.Close()

	encoder, err := zstd.NewWriter(tarFile, zstd.WithEncoderLevel(level))
	if err != nil {
		return err
	}
	defer encoder.Close()

	tarWriter := tar.NewWriter(encoder)
	defer tarWriter.Close()

	// Walk through the input directory and add files to the tar archive
	err = filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Open the file
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		// Create a tar header
		header, err := tar.FileInfoHeader(info, path)
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(inputDir, path)
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(relPath) // Use relative path inside tar

		// Write the header and file content to the tar archive
		err = tarWriter.WriteHeader(header)
		if err != nil {
			return err
		}
		_, err = io.Copy(tarWriter, file)
		return err
	})

	if err != nil {
		return fmt.Errorf("error creating tarball: %v", err)
	}

	// fmt.Println("Tarball successfully created.")
	return nil
}

func sanitizeFilename(name string) string {
	return filepath.Base(name)
}
