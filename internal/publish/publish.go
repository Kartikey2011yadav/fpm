package publish

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

type Publisher struct {
	Repository string
	Token      string
	Username   string
	Password   string
}

type PublishOptions struct {
	Repository string
	Token      string
	Username   string
	Password   string
}

func New(opts PublishOptions) *Publisher {
	if opts.Repository == "" {
		opts.Repository = "https://upload.pypi.org/legacy/"
	}
	if opts.Username == "" && opts.Token != "" {
		opts.Username = "__token__"
		opts.Password = opts.Token
	}
	return &Publisher{
		Repository: opts.Repository,
		Token:      opts.Token,
		Username:   opts.Username,
		Password:   opts.Password,
	}
}

func (p *Publisher) Upload(distPaths []string) error {
	for _, path := range distPaths {
		if err := p.uploadFile(path); err != nil {
			return fmt.Errorf("failed to upload %s: %w", filepath.Base(path), err)
		}
		fmt.Printf("  Uploaded %s\n", filepath.Base(path))
	}
	return nil
}

func (p *Publisher) uploadFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add the file
	part, err := writer.CreateFormFile("content", filepath.Base(filePath))
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, file); err != nil {
		return err
	}

	// Add protocol version
	writer.WriteField(":action", "file_upload")
	writer.WriteField("protocol_version", "1")

	writer.Close()

	req, err := http.NewRequest("POST", p.Repository, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	if p.Username != "" {
		req.SetBasicAuth(p.Username, p.Password)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}
