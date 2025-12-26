package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
	"google.golang.org/api/drive/v3"
)

type copyViaDriveOptions struct {
	Use          string
	Short        string
	ArgName      string
	ExpectedMime string
	KindLabel    string
}

func newDriveCopyCmd(flags *rootFlags) *cobra.Command {
	return newCopyViaDriveCmd(flags, copyViaDriveOptions{
		Use:   "copy <fileId> <name>",
		Short: "Copy a file",
	})
}

func newCopyViaDriveCmd(flags *rootFlags, opts copyViaDriveOptions) *cobra.Command {
	var parent string

	argName := strings.TrimSpace(opts.ArgName)
	if argName == "" {
		argName = "id"
	}

	cmd := &cobra.Command{
		Use:   opts.Use,
		Short: opts.Short,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			u := ui.FromContext(cmd.Context())
			account, err := requireAccount(flags)
			if err != nil {
				return err
			}

			id := strings.TrimSpace(args[0])
			if id == "" {
				return usage(fmt.Sprintf("empty %s", argName))
			}
			name := strings.TrimSpace(args[1])
			if name == "" {
				return usage("empty name")
			}

			svc, err := newDriveService(cmd.Context(), account)
			if err != nil {
				return err
			}

			meta, err := svc.Files.Get(id).
				SupportsAllDrives(true).
				Fields("id, name, mimeType").
				Context(cmd.Context()).
				Do()
			if err != nil {
				return err
			}
			if meta == nil {
				return errors.New("file not found")
			}
			if opts.ExpectedMime != "" && meta.MimeType != opts.ExpectedMime {
				label := strings.TrimSpace(opts.KindLabel)
				if label == "" {
					label = "expected type"
				}
				return fmt.Errorf("file is not a %s (mimeType=%q)", label, meta.MimeType)
			}

			req := &drive.File{Name: name}
			parent = strings.TrimSpace(parent)
			if parent != "" {
				req.Parents = []string{parent}
			}

			created, err := svc.Files.Copy(id, req).
				SupportsAllDrives(true).
				Fields("id, name, mimeType, webViewLink").
				Context(cmd.Context()).
				Do()
			if err != nil {
				return err
			}
			if created == nil {
				return errors.New("copy failed")
			}

			if outfmt.IsJSON(cmd.Context()) {
				return outfmt.WriteJSON(os.Stdout, map[string]any{"file": created})
			}
			u.Out().Printf("id\t%s", created.Id)
			u.Out().Printf("name\t%s", created.Name)
			u.Out().Printf("mime\t%s", created.MimeType)
			if created.WebViewLink != "" {
				u.Out().Printf("link\t%s", created.WebViewLink)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&parent, "parent", "", "Destination folder ID")
	return cmd
}
