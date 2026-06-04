package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kartikeyyadav/fpm/internal/publish"
	"github.com/spf13/cobra"
)

func init() {
	publishCmd.RunE = runPublish
}

func runPublish(cmd *cobra.Command, args []string) error {
	repository, _ := cmd.Flags().GetString("repository")
	token, _ := cmd.Flags().GetString("token")

	// If no token flag, check environment
	if token == "" {
		token = os.Getenv("FPM_PUBLISH_TOKEN")
	}

	// Find dist files
	var distPaths []string
	if len(args) > 0 {
		distPaths = args
	} else {
		// Default: look in dist/
		matches, _ := filepath.Glob("dist/*")
		distPaths = matches
	}

	if len(distPaths) == 0 {
		return fmt.Errorf("no distribution files found. Run 'fpm build' first")
	}

	repoURL := "https://upload.pypi.org/legacy/"
	if repository == "testpypi" {
		repoURL = "https://test.pypi.org/legacy/"
	} else if repository != "pypi" && repository != "" {
		repoURL = repository
	}

	publisher := publish.New(publish.PublishOptions{
		Repository: repoURL,
		Token:      token,
	})

	fmt.Printf("Publishing to %s...\n", repository)
	if err := publisher.Upload(distPaths); err != nil {
		return err
	}

	fmt.Println("Published successfully.")
	return nil
}
