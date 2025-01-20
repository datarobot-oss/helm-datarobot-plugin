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
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
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
Tarball created successfully: images.tgz
$ du -h images.tgz
14M    images.tgz

'''`, "'", "`", -1),
	Args: cobra.MinimumNArgs(1), // Requires at least one argument (file path)
	RunE: func(cmd *cobra.Command, args []string) error {
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

			if saveDryRun {
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
				if saveDryRun {
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
			if _, err := os.Stat(tgzFileName); err == nil {
				cmd.Printf(" archive %s already exists\n", tgzFileName)
				continue
			}

			if saveDryRun {
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
			tgzFiles = append(tgzFiles, tgzFileName)

		}
		if !saveDryRun {
			err = CreateTGZ(saveOutput, tgzFiles)
			if err != nil {
				return fmt.Errorf("Error: %v\n", err)
			}
		}
		if saveDryRun {
			cmd.Printf("[Dry-Run] Tarball created successfully: %s\n", saveOutput)
		} else {
			cmd.Printf("Tarball created successfully: %s\n", saveOutput)
		}
		return nil
	},
}

var saveOutput string
var saveDryRun bool

func init() {
	rootCmd.AddCommand(saveCmd)
	saveCmd.Flags().StringVarP(&annotation, "annotation", "a", "datarobot.com/images", "annotation to lookup")
	saveCmd.Flags().StringVarP(&saveOutput, "output", "o", "images.tgz", "file to save")
	saveCmd.Flags().BoolVarP(&saveDryRun, "dry-run", "", false, "Perform a dry run without making changes")
}

func CreateTGZ(outputPath string, inputTGZPaths []string) error {
	// Open the output file for writing
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer outputFile.Close()

	// Create a gzip writer
	gzipWriter := gzip.NewWriter(outputFile)
	defer gzipWriter.Close()

	// Create a tar writer
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	// Iterate over the list of input tgz paths
	for _, tgzPath := range inputTGZPaths {
		// Open the tgz file
		tgzFile, err := os.Open(tgzPath)
		if err != nil {
			return fmt.Errorf("failed to open tgz file %q: %v", tgzPath, err)
		}
		defer tgzFile.Close()

		// Extract file information (like name and size)
		fileInfo, err := tgzFile.Stat()
		if err != nil {
			return fmt.Errorf("failed to get file info for %q: %v", tgzPath, err)
		}

		// Create a tar header for the tgz file
		header := &tar.Header{
			Name: tgzPath,
			Size: fileInfo.Size(),
			Mode: int64(fileInfo.Mode()),
		}

		// Write the header for the tgz file to the tar archive
		err = tarWriter.WriteHeader(header)
		if err != nil {
			return fmt.Errorf("failed to write tar header: %v", err)
		}

		// Copy the content of the tgz file to the tar archive
		_, err = io.Copy(tarWriter, tgzFile)
		if err != nil {
			return fmt.Errorf("failed to copy tgz file content: %v", err)
		}

		// Delete the tgz file
		err = os.Remove(tgzPath)
		if err != nil {
			return fmt.Errorf("failed to delete tgz file %q: %v", tgzPath, err)
		}

		dirPath := filepath.Dir(tgzPath)
		// Read the contents of the directory
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			return fmt.Errorf("error reading directory: %s", err)
		}
		if len(entries) == 0 {
			// Remove the directory if empty
			err = os.Remove(dirPath)
			if err != nil {
				return fmt.Errorf("error deleting directory: %s", err)
			}
		}
	}

	return nil
}
