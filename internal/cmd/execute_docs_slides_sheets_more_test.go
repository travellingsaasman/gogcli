package cmd

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

func TestExecute_DocsSlidesSheets_CopyCreateInfoCat_JSON(t *testing.T) {
	origNew := newDriveService
	origExport := driveExportDownload
	t.Cleanup(func() {
		newDriveService = origNew
		driveExportDownload = origExport
	})

	var createCalls int32
	var copyCalls int32
	var exportCalls int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case r.Method == http.MethodGet && strings.Contains(path, "/files/d1") && !strings.HasSuffix(path, "/copy"):
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":       "d1",
				"name":     "Doc 1",
				"mimeType": "application/vnd.google-apps.document",
			})
			return
		case r.Method == http.MethodGet && strings.Contains(path, "/files/p1") && !strings.HasSuffix(path, "/copy"):
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":       "p1",
				"name":     "Slides 1",
				"mimeType": "application/vnd.google-apps.presentation",
			})
			return
		case r.Method == http.MethodGet && strings.Contains(path, "/files/s1") && !strings.HasSuffix(path, "/copy"):
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":       "s1",
				"name":     "Sheet 1",
				"mimeType": "application/vnd.google-apps.spreadsheet",
			})
			return
		case r.Method == http.MethodPost && strings.HasSuffix(path, "/copy"):
			atomic.AddInt32(&copyCalls, 1)
			w.Header().Set("Content-Type", "application/json")
			id := "copy-unknown"
			switch {
			case strings.Contains(path, "/files/d1/copy"):
				id = "d1-copy"
			case strings.Contains(path, "/files/p1/copy"):
				id = "p1-copy"
			case strings.Contains(path, "/files/s1/copy"):
				id = "s1-copy"
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":           id,
				"name":         "Copy",
				"mimeType":     "application/octet-stream",
				"webViewLink":  "https://example.com/" + id,
				"modifiedTime": "2025-12-26T00:00:00Z",
			})
			return
		case r.Method == http.MethodPost && (strings.HasSuffix(path, "/files") || strings.HasSuffix(path, "/drive/v3/files")):
			atomic.AddInt32(&createCalls, 1)
			var req map[string]any
			_ = json.NewDecoder(r.Body).Decode(&req)
			mt, _ := req["mimeType"].(string)
			name, _ := req["name"].(string)
			id := "created"
			switch mt {
			case "application/vnd.google-apps.document":
				id = "doc-created"
			case "application/vnd.google-apps.presentation":
				id = "slides-created"
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          id,
				"name":        name,
				"mimeType":    mt,
				"webViewLink": "https://example.com/" + id,
			})
			return
		default:
			http.NotFound(w, r)
			return
		}
	}))
	defer srv.Close()

	svc, err := drive.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	newDriveService = func(context.Context, string) (*drive.Service, error) { return svc, nil }

	driveExportDownload = func(context.Context, *drive.Service, string, string) (*http.Response, error) {
		atomic.AddInt32(&exportCalls, 1)
		return &http.Response{
			Status:     "200 OK",
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("hello")),
		}, nil
	}

	run := func(args ...string) map[string]any {
		out := captureStdout(t, func() {
			_ = captureStderr(t, func() {
				if execErr := Execute(append([]string{"--json", "--account", "a@b.com"}, args...)); execErr != nil {
					t.Fatalf("Execute(%v): %v", args, execErr)
				}
			})
		})
		var parsed map[string]any
		if unmarshalErr := json.Unmarshal([]byte(out), &parsed); unmarshalErr != nil {
			t.Fatalf("json parse: %v\nout=%q", unmarshalErr, out)
		}
		return parsed
	}

	_ = run("docs", "create", "T")
	_ = run("docs", "info", "d1")
	_ = run("docs", "copy", "d1", "T2")
	gotCat := run("docs", "cat", "d1")
	if gotCat["text"] != "hello" {
		t.Fatalf("unexpected docs cat text=%v", gotCat["text"])
	}

	_ = run("slides", "create", "T")
	_ = run("slides", "info", "p1")
	_ = run("slides", "copy", "p1", "T2")

	_ = run("sheets", "copy", "s1", "T2")
	_ = run("drive", "copy", "d1", "T2")

	if atomic.LoadInt32(&createCalls) != 2 {
		t.Fatalf("createCalls=%d", createCalls)
	}
	if atomic.LoadInt32(&copyCalls) < 4 {
		t.Fatalf("copyCalls=%d", copyCalls)
	}
	if atomic.LoadInt32(&exportCalls) != 1 {
		t.Fatalf("exportCalls=%d", exportCalls)
	}
}

func TestExecute_DocsCat_WrongMime(t *testing.T) {
	origNew := newDriveService
	t.Cleanup(func() { newDriveService = origNew })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":       "x1",
			"mimeType": "application/pdf",
		})
	}))
	defer srv.Close()

	svc, err := drive.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	newDriveService = func(context.Context, string) (*drive.Service, error) { return svc, nil }

	err = Execute([]string{"--account", "a@b.com", "docs", "cat", "x1"})
	if err == nil || !strings.Contains(err.Error(), "not a Google Doc") {
		t.Fatalf("expected not a Google Doc error, got: %v", err)
	}
}
