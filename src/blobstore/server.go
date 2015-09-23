package main

import (
	"blob"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"uuid"
)

const ApplicationOctetStream = "application/octet-stream"

// URL for blobs
var blobPathMatcher = regexp.MustCompile(`^/([a-zA-Z0-9\-]+)$`)

// Fetch, decode and verify authorization header.
func authenticate(r *http.Request) (map[string]interface{}, error) {
	if *secretKey == "" {
		return map[string]interface{}{}, nil // DISABLE AUTH
	}
	token := r.Header.Get("Authorization")
	return jwtDecode(*secretKey, token)
}

// UploadHandler accepts multipart form uploads of a blob and stores it on S3
func BlobHandler(w http.ResponseWriter, r *http.Request) {
	if err := blobHandler(w, r); err != nil {
		fmt.Fprintln(os.Stderr, err)
		http.Error(w, err.Error(), 400)
	}
}

func blobHandler(w http.ResponseWriter, r *http.Request) error {
	// Enable CORS
	h := w.Header()
	if origin := h.Get("Origin"); origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
	}
	h.Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	h.Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	// Router
	switch r.Method {
	case "POST":
		return uploadHandler(w, r)
	case "OPTIONS":
		return nil
	default:
		return downloadHandler(w, r)
	}
}

func downloadHandler(w http.ResponseWriter, r *http.Request) error {
	match := blobPathMatcher.FindStringSubmatch(r.URL.Path)
	if len(match) != 2 {
		return errors.New("bad request: " + r.URL.String())
	}
	id, err := uuid.ParseUUID(match[1])
	if err != nil {
		return err
	}
	blob, err := blob.Get(id)
	if err != nil {
		return err
	}
	f, err := blob.File()
	if err != nil {
		return err
	}
	defer f.Close()
	if blob.ContentType != "" {
		w.Header().Set("Content-Type", blob.ContentType)
	}
	http.ServeContent(w, r, blob.Name, blob.Time(), f)
	return nil
}

// Copy form file to Blob
func upload(f *multipart.FileHeader) (b *blob.Blob, err error) {
	// Open
	upload, err := f.Open()
	if err != nil {
		return
	}
	defer upload.Close()
	// Create blob
	b = blob.New()
	// Set filename from request
	b.Name = f.Filename
	// Set content-type from request
	if ct := f.Header.Get("Content-Type"); ct != "" {
		b.ContentType = ct
	}
	// Guess content-type from extension if missing
	if b.ContentType == "" || b.ContentType == ApplicationOctetStream {
		if ext := filepath.Ext(b.Name); ext != "" {
			b.ContentType = mime.TypeByExtension(ext)
		}
	}
	if b.ContentType == "" {
		b.ContentType = ApplicationOctetStream
	}
	// Write
	err = b.WriteFrom(upload)
	if err != nil {
		return
	}
	return
}

func uploadHandler(w http.ResponseWriter, r *http.Request) error {
	// Auth
	if _, err := authenticate(r); err != nil {
		return err
	}
	// Parse
	if err := r.ParseMultipartForm(*serverMaxUploadMem * MB); err != nil {
		return err
	}
	form := r.MultipartForm
	defer form.RemoveAll()
	// For each file
	uploads := form.File["file"]
	blobs := []*blob.Blob{}
	for _, f := range uploads {
		b, err := upload(f)
		if err != nil {
			return err
		}
		blobs = append(blobs, b)
		fmt.Println("created blob", b.ID, "for", b.Name, b.ContentType)
	}
	// Check we actually uploaded something
	if len(blobs) == 0 {
		return errors.New("no blobs stored")
	}
	// Write JSON response
	enc := json.NewEncoder(w)
	if err := enc.Encode(blobs); err != nil {
		return err
	}
	return nil
}

// ListenAndServe starts the HTTP server listening on port 8080
func ListenAndServe(addr string) error {
	fmt.Println("starting blobstore service at", addr)
	mux := http.NewServeMux()
	mux.Handle("/test/", http.StripPrefix("/test/", http.FileServer(http.Dir("public"))))
	mux.Handle("/favicon.ico", http.FileServer(http.Dir("public")))
	mux.HandleFunc("/", BlobHandler)
	return http.ListenAndServe(addr, mux)
}
