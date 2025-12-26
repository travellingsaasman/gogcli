package cmd

import (
	"errors"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
	"google.golang.org/api/drive/v3"
)

func newSlidesCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "slides",
		Short: "Google Slides (export via Drive)",
	}
	cmd.AddCommand(newSlidesExportCmd(flags))
	cmd.AddCommand(newSlidesInfoCmd(flags))
	cmd.AddCommand(newSlidesCreateCmd(flags))
	cmd.AddCommand(newSlidesCopyCmd(flags))
	return cmd
}

func newSlidesExportCmd(flags *rootFlags) *cobra.Command {
	return newExportViaDriveCmd(flags, exportViaDriveOptions{
		Use:           "export <presentationId>",
		Short:         "Export a Google Slides deck (pdf|pptx)",
		ArgName:       "presentationId",
		ExpectedMime:  "application/vnd.google-apps.presentation",
		KindLabel:     "Google Slides presentation",
		DefaultFormat: "pptx",
		FormatHelp:    "Export format: pdf|pptx",
	})
}

func newSlidesInfoCmd(flags *rootFlags) *cobra.Command {
	return newInfoViaDriveCmd(flags, infoViaDriveOptions{
		Use:          "info <presentationId>",
		Short:        "Get Google Slides presentation metadata",
		ArgName:      "presentationId",
		ExpectedMime: "application/vnd.google-apps.presentation",
		KindLabel:    "Google Slides presentation",
	})
}

func newSlidesCreateCmd(flags *rootFlags) *cobra.Command {
	var parent string

	cmd := &cobra.Command{
		Use:   "create <title>",
		Short: "Create a Google Slides presentation",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			u := ui.FromContext(cmd.Context())
			account, err := requireAccount(flags)
			if err != nil {
				return err
			}

			title := strings.TrimSpace(args[0])
			if title == "" {
				return usage("empty title")
			}

			svc, err := newDriveService(cmd.Context(), account)
			if err != nil {
				return err
			}

			f := &drive.File{
				Name:     title,
				MimeType: "application/vnd.google-apps.presentation",
			}
			parent = strings.TrimSpace(parent)
			if parent != "" {
				f.Parents = []string{parent}
			}

			created, err := svc.Files.Create(f).
				SupportsAllDrives(true).
				Fields("id, name, mimeType, webViewLink").
				Context(cmd.Context()).
				Do()
			if err != nil {
				return err
			}
			if created == nil {
				return errors.New("create failed")
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

func newSlidesCopyCmd(flags *rootFlags) *cobra.Command {
	return newCopyViaDriveCmd(flags, copyViaDriveOptions{
		Use:          "copy <presentationId> <title>",
		Short:        "Copy a Google Slides presentation",
		ArgName:      "presentationId",
		ExpectedMime: "application/vnd.google-apps.presentation",
		KindLabel:    "Google Slides presentation",
	})
}
