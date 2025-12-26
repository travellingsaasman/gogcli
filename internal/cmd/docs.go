package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
	"google.golang.org/api/drive/v3"
)

func newDocsCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docs",
		Short: "Google Docs (export via Drive)",
	}
	cmd.AddCommand(newDocsExportCmd(flags))
	cmd.AddCommand(newDocsInfoCmd(flags))
	cmd.AddCommand(newDocsCreateCmd(flags))
	cmd.AddCommand(newDocsCopyCmd(flags))
	cmd.AddCommand(newDocsCatCmd(flags))
	return cmd
}

func newDocsExportCmd(flags *rootFlags) *cobra.Command {
	return newExportViaDriveCmd(flags, exportViaDriveOptions{
		Use:           "export <docId>",
		Short:         "Export a Google Doc (pdf|docx|txt)",
		ArgName:       "docId",
		ExpectedMime:  "application/vnd.google-apps.document",
		KindLabel:     "Google Doc",
		DefaultFormat: "pdf",
		FormatHelp:    "Export format: pdf|docx|txt",
	})
}

func newDocsInfoCmd(flags *rootFlags) *cobra.Command {
	return newInfoViaDriveCmd(flags, infoViaDriveOptions{
		Use:          "info <docId>",
		Short:        "Get Google Doc metadata",
		ArgName:      "docId",
		ExpectedMime: "application/vnd.google-apps.document",
		KindLabel:    "Google Doc",
	})
}

func newDocsCreateCmd(flags *rootFlags) *cobra.Command {
	var parent string

	cmd := &cobra.Command{
		Use:   "create <title>",
		Short: "Create a Google Doc",
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
				MimeType: "application/vnd.google-apps.document",
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

func newDocsCopyCmd(flags *rootFlags) *cobra.Command {
	return newCopyViaDriveCmd(flags, copyViaDriveOptions{
		Use:          "copy <docId> <title>",
		Short:        "Copy a Google Doc",
		ArgName:      "docId",
		ExpectedMime: "application/vnd.google-apps.document",
		KindLabel:    "Google Doc",
	})
}

func newDocsCatCmd(flags *rootFlags) *cobra.Command {
	var maxBytes int64

	cmd := &cobra.Command{
		Use:   "cat <docId>",
		Short: "Print a Google Doc as plain text",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			account, err := requireAccount(flags)
			if err != nil {
				return err
			}

			id := strings.TrimSpace(args[0])
			if id == "" {
				return usage("empty docId")
			}

			svc, err := newDriveService(cmd.Context(), account)
			if err != nil {
				return err
			}

			meta, err := svc.Files.Get(id).
				SupportsAllDrives(true).
				Fields("id, mimeType").
				Context(cmd.Context()).
				Do()
			if err != nil {
				return err
			}
			if meta == nil {
				return errors.New("file not found")
			}
			if meta.MimeType != "application/vnd.google-apps.document" {
				return fmt.Errorf("file is not a Google Doc (mimeType=%q)", meta.MimeType)
			}

			resp, err := driveExportDownload(cmd.Context(), svc, id, "text/plain")
			if err != nil {
				return err
			}
			if resp == nil || resp.Body == nil {
				return errors.New("empty response")
			}
			defer resp.Body.Close()

			var r io.Reader = resp.Body
			if maxBytes > 0 {
				r = io.LimitReader(resp.Body, maxBytes)
			}
			b, err := io.ReadAll(r)
			if err != nil {
				return err
			}

			if outfmt.IsJSON(cmd.Context()) {
				return outfmt.WriteJSON(os.Stdout, map[string]any{"text": string(b)})
			}
			_, err = os.Stdout.Write(b)
			return err
		},
	}

	cmd.Flags().Int64Var(&maxBytes, "max-bytes", 2_000_000, "Max bytes to read (0 = unlimited)")
	return cmd
}
