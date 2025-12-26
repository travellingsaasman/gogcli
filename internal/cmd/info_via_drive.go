package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
)

type infoViaDriveOptions struct {
	Use          string
	Short        string
	ArgName      string
	ExpectedMime string
	KindLabel    string
}

func newInfoViaDriveCmd(flags *rootFlags, opts infoViaDriveOptions) *cobra.Command {
	argName := strings.TrimSpace(opts.ArgName)
	if argName == "" {
		argName = "id"
	}

	cmd := &cobra.Command{
		Use:   opts.Use,
		Short: opts.Short,
		Args:  cobra.ExactArgs(1),
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

			svc, err := newDriveService(cmd.Context(), account)
			if err != nil {
				return err
			}

			f, err := svc.Files.Get(id).
				SupportsAllDrives(true).
				Fields("id, name, mimeType, size, createdTime, modifiedTime, webViewLink, parents").
				Context(cmd.Context()).
				Do()
			if err != nil {
				return err
			}
			if f == nil {
				return errors.New("file not found")
			}
			if opts.ExpectedMime != "" && f.MimeType != opts.ExpectedMime {
				label := strings.TrimSpace(opts.KindLabel)
				if label == "" {
					label = "expected type"
				}
				return fmt.Errorf("file is not a %s (mimeType=%q)", label, f.MimeType)
			}

			if outfmt.IsJSON(cmd.Context()) {
				return outfmt.WriteJSON(os.Stdout, map[string]any{"file": f})
			}

			u.Out().Printf("id\t%s", f.Id)
			u.Out().Printf("name\t%s", f.Name)
			u.Out().Printf("mime\t%s", f.MimeType)
			if f.WebViewLink != "" {
				u.Out().Printf("link\t%s", f.WebViewLink)
			}
			if f.CreatedTime != "" {
				u.Out().Printf("created\t%s", f.CreatedTime)
			}
			if f.ModifiedTime != "" {
				u.Out().Printf("modified\t%s", f.ModifiedTime)
			}
			if len(f.Parents) > 0 {
				u.Out().Printf("parents\t%s", strings.Join(f.Parents, ","))
			}
			return nil
		},
	}

	return cmd
}
