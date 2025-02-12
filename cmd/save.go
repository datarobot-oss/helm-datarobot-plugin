package cmd

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/datarobot-oss/helm-datarobot-plugin/pkg/image_uri"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/klauspost/compress/zstd"
	"github.com/sethvargo/go-envconfig"
	"github.com/spf13/cobra"
)

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

		ctx := context.Background()
		if err := envconfig.Process(ctx, &loadCfg); err != nil {
			return fmt.Errorf("%v", err)
		}

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

		images, err := ExtractImagesFromCharts(args)
		if err != nil {
			return fmt.Errorf("Error ExtractImagesFromCharts: %v", err)
		}

		tgzFiles := []string{}
		img := empty.Image

		for _, image := range images {
			iUri, err := image_uri.NewDockerUri(image.Image)
			if err != nil {
				return err
			}

			if len(saveCfg.ImageSkipGroup) > 0 {
				_skipImage := false
				for _, group := range saveCfg.ImageSkipGroup {
					if image.Group == group {
						cmd.Printf("Skipping image: %s\n\n", iUri.String())
						_skipImage = true
						continue
					}
				}
				if _skipImage {
					continue
				}
			}

			if saveCfg.DryRun {
				cmd.Printf("[Dry-Run] Pulling image: %s\n", iUri.String())
			} else {
				cmd.Printf("Pulling image: %s\n", iUri.String())
				img, err = crane.Pull(iUri.String())
				if err != nil {
					return fmt.Errorf("failed to pull image: %w", err)
				}
			}

			oldName := iUri.String()
			if image.Tag != "" {
				iUri.Tag = image.Tag
				if saveCfg.DryRun {
					cmd.Printf("[Dry-Run] ReTagging image: %s > %s\n", oldName, iUri.String())
				} else {
					cmd.Printf("ReTagging image: %s > %s\n", oldName, iUri.String())
				}

			}
			imageDir := ""
			if iUri.Organization != "" || iUri.Project != "" {
				imageDir = iUri.Join([]string{iUri.Organization, iUri.Project}, "/")
				if err := os.MkdirAll(imageDir, os.ModePerm); err != nil {
					return fmt.Errorf("creating directory %s: %v", imageDir, err)
				}
			}

			tgzFileName := iUri.Join([]string{imageDir, iUri.ImageName}, "/") + ":" + iUri.Tag + ".tgz"
			tgzFiles = append(tgzFiles, tgzFileName)
			if _, err := os.Stat(tgzFileName); err == nil {
				cmd.Printf(" archive %s already exists\n", tgzFileName)
				continue
			}

			if saveCfg.DryRun {
				cmd.Printf("[Dry-Run] adding image to tgz: %s\n", tgzFileName)
			} else {
				ref, err := name.ParseReference(iUri.String())
				if err != nil {
					return fmt.Errorf("Error parsing image reference %s: %v\n", image, err)
				}
				if err := tarball.WriteToFile(tgzFileName, ref, img); err != nil {
					return fmt.Errorf("Error writing image %s to tarball: %v\n", iUri.String(), err)
				}
			}

		}
		if !saveCfg.DryRun {
			err = createTarball(saveCfg.Output, tgzFiles, level)
			if err != nil {
				return fmt.Errorf("Error createTarball: %v\n", err)
			}
			err = deleteTmpFiles(tgzFiles)
			if err != nil {
				return fmt.Errorf("Error deleteTmpFiles: %v\n", err)
			}
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
	CompressionLevel string   `env:"LEVEL"`
	ImageSkipGroup   []string `env:"IMAGE_SKIP_GROUP"`
	DryRun           bool     `env:"DRY_RUN"`
}

var saveCfg saveConfig

func init() {
	rootCmd.AddCommand(saveCmd)
	saveCmd.Flags().StringVarP(&annotation, "annotation", "a", "datarobot.com/images", "annotation to lookup")
	saveCmd.Flags().StringVarP(&saveCfg.Output, "output", "o", "images.tar.zst", "file to save")
	saveCmd.Flags().StringVarP(&saveCfg.CompressionLevel, "level", "l", "best", "zstd compression level (Available options: fastest, default, better, best)")
	saveCmd.Flags().StringArrayVarP(&saveCfg.ImageSkipGroup, "skip-group", "", []string{}, "Specify which image group should be skipped (can be used multiple times)")
	saveCmd.Flags().BoolVarP(&saveCfg.DryRun, "dry-run", "", false, "Perform a dry run without making changes")
}

// CreateZST creates a .zst archive from the specified input TGZ files
func createTarball(outputPath string, inputTGZPaths []string, level zstd.EncoderLevel) error {
	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	encoder, err := zstd.NewWriter(outFile, zstd.WithEncoderLevel(level))
	if err != nil {
		return err
	}
	defer encoder.Close()

	// Create a new tar writer
	tarWriter := tar.NewWriter(encoder)
	defer tarWriter.Close()

	for _, tgzPath := range inputTGZPaths {
		err := addFileToTarball(tarWriter, tgzPath)
		if err != nil {
			return err
		}
	}

	return nil
}

func addFileToTarball(tarWriter *tar.Writer, filePath string) error {
	// Open the file to be added
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	// Get the file info
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file %s: %w", filePath, err)
	}
	header := &tar.Header{
		Name: filePath,
		Size: fileInfo.Size(),
		Mode: int64(fileInfo.Mode()),
	}
	// Write the header to the tar writer
	err = tarWriter.WriteHeader(header)
	if err != nil {
		return fmt.Errorf("failed to write header for file %s: %w", filePath, err)
	}

	// Copy the file data to the tar writer
	_, err = io.Copy(tarWriter, file)
	if err != nil {
		return fmt.Errorf("failed to write file %s to tar: %w", filePath, err)
	}

	return nil
}

func deleteTmpFiles(filePaths []string) error {
	// Create a map to track directories to be checked for emptiness
	directoriesToCheck := make(map[string]struct{})

	for _, filePath := range filePaths {
		// Check if the file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			continue
		}

		// Get the directory of the file
		dir := filepath.Dir(filePath)
		directoriesToCheck[dir] = struct{}{} // Mark the directory for checking later

		// Delete the file
		err := os.Remove(filePath)
		if err != nil {
			return fmt.Errorf("error deleting file %s: %v", filePath, err)
		}
	}

	// Check each directory to see if it is empty and delete it if so
	for dir := range directoriesToCheck {
		if dir == "." {
			// Skip the current directory
			continue
		}
		err := os.Remove(dir)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("error deleting directory %s: %v", dir, err)
		}
	}

	return nil
}
