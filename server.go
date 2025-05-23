package main

import (
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

// MimeTypeResponseWriter is a wrapper around http.ResponseWriter that allows
// for setting the Content-Type header based on the file extension.
type MimeTypeResponseWriter struct {
	http.ResponseWriter
}

// WriteHeader sets the Content-Type header before writing headers.
func (w *MimeTypeResponseWriter) WriteHeader(statusCode int) {
	// This is a bit of a simplification. A more robust solution would inspect
	// the request path from within a handler that has access to it.
	// For this specific server, we'll make a custom handler.
	w.ResponseWriter.WriteHeader(statusCode)
}

func main() {
	port := "8080"
	log.Printf("Starting server on http://localhost:%s\n", port)

	// Custom handler to set MIME types
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		filePath := r.URL.Path
		if filePath == "/" {
			filePath = "/index.html" // Serve index.html by default
		}
		// Ensure the path is clean and prevent directory traversal
		filePath = filepath.Clean(filePath)
		if strings.HasPrefix(filePath, "..") { // Basic security check
			http.NotFound(w, r)
			return
		}

		// Set Content-Type based on extension
		ext := filepath.Ext(filePath)
		mimeType := ""
		switch ext {
		case ".js":
			mimeType = "application/javascript"
		case ".wasm":
			mimeType = "application/wasm"
		case ".css":
			mimeType = "text/css"
		case ".html":
			mimeType = "text/html; charset=utf-8"
		case ".json":
			mimeType = "application/json"
		case ".png":
			mimeType = "image/png"
		case ".jpg", ".jpeg":
			mimeType = "image/jpeg"
		// Add other types as needed
		default:
			// Let http.ServeFile try to determine or use a default
			// For unknown types, some servers might default to text/plain or application/octet-stream
		}

		if mimeType != "" {
			w.Header().Set("Content-Type", mimeType)
		}

		// Serve the file from the current directory "."
		// Using http.Dir(".") specifies the root of the file server.
		// The path given to ServeFile must be relative to this root.
		// We need to strip the leading "/" from filePath if it exists.
		http.ServeFile(w, r, filepath.Join(".", strings.TrimPrefix(filePath, "/")))
	})

	// Start the server
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
