package pull

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"badgermaps/events"

	"github.com/spf13/cobra"
)

// ResponseSaveOptions captures CLI preferences for saving raw pull payloads.
type ResponseSaveOptions struct {
	Save       bool
	OutputPath string
}

func (o ResponseSaveOptions) enabled() bool {
	if strings.TrimSpace(o.OutputPath) != "" {
		return true
	}
	if o.OutputPath == "-" {
		return true
	}
	return o.Save
}

func bindResponseSaveFlags(cmd *cobra.Command) *ResponseSaveOptions {
	opts := &ResponseSaveOptions{}
	cmd.Flags().BoolVarP(&opts.Save, "save-response", "s", false, "Save the fetched record as JSON under ./pull-responses.")
	cmd.Flags().StringVarP(&opts.OutputPath, "response-file", "o", "", "Write JSON response to the provided path (use '-' for stdout). Implies --save-response.")
	return opts
}

func (p *CliPresenter) saveResponse(resource, identifier string, payload interface{}, opts ResponseSaveOptions) error {
	if !opts.enabled() || payload == nil {
		return nil
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal %s payload: %w", resource, err)
	}

	outputPath, err := resolveOutputPath(resource, identifier, opts)
	if err != nil {
		return err
	}

	if outputPath == "-" {
		if _, err := fmt.Fprintln(os.Stdout, string(data)); err != nil {
			return fmt.Errorf("failed writing %s payload to stdout: %w", resource, err)
		}
		p.App.Events.Dispatch(events.Infof("pull", "Wrote %s response to stdout.", resource))
		return nil
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write %s response to %s: %w", resource, outputPath, err)
	}
	p.App.Events.Dispatch(events.Infof("pull", "Saved %s response to %s", resource, outputPath))
	return nil
}

func resolveOutputPath(resource, identifier string, opts ResponseSaveOptions) (string, error) {
	if opts.OutputPath == "-" {
		return "-", nil
	}

	path := strings.TrimSpace(opts.OutputPath)
	if path == "" {
		dir := filepath.Join(".", "pull-responses")
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("failed to create response directory: %w", err)
		}
		filename := fmt.Sprintf("pull-%s-%s-%s.json", sanitizeComponent(resource), sanitizeComponent(identifier), time.Now().Format("20060102-150405"))
		return filepath.Join(dir, filename), nil
	}

	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("failed to create response directory %s: %w", dir, err)
		}
	}
	return path, nil
}

func sanitizeComponent(value string) string {
	v := strings.TrimSpace(value)
	if v == "" {
		return "record"
	}
	v = strings.ReplaceAll(v, "..", "_")
	v = strings.ReplaceAll(v, "/", "_")
	v = strings.ReplaceAll(v, string(os.PathSeparator), "_")
	return v
}
