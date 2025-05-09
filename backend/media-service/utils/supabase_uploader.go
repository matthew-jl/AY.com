package utils

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	supabase "github.com/nedpals/supabase-go"
)

var (
	supabaseClient    *supabase.Client
	supabaseBucket    string
)

func init() {
	supabaseURL := os.Getenv("SUPABASE_URL")
	supabaseKey := os.Getenv("SUPABASE_SERVICE_KEY")
	supabaseBucket = os.Getenv("SUPABASE_BUCKET_NAME")

	if supabaseURL == "" || supabaseKey == "" || supabaseBucket == "" {
		log.Println("Warning: Supabase environment variables (URL, SERVICE_KEY, BUCKET_NAME) not fully configured. Uploads will fail.")
        return
	}
    log.Printf("Initializing Supabase client for URL: %s, Bucket: %s", supabaseURL, supabaseBucket)
	supabaseClient = supabase.CreateClient(supabaseURL, supabaseKey)
}

// UploadFileToSupabase uploads data and returns the storage path (not public URL).
func UploadFileToSupabase(ctx context.Context, uploaderID uint, originalFilename string, mimeType string, data []byte) (filePath string, fileSize int64, err error) {
    if supabaseClient == nil {
		 return "", 0, fmt.Errorf("supabase client not initialized due to missing configuration")
    }

	// Generate a unique filename using UUID and original extension
	ext := filepath.Ext(originalFilename)
	uniqueFilename := fmt.Sprintf("%d/%s%s", uploaderID, uuid.NewString(), ext) // Store in user-specific folder

	log.Printf("Uploading file to Supabase bucket '%s' at path '%s'", supabaseBucket, uniqueFilename)

	// Upload the file content
	fileReader := bytes.NewReader(data)

	uploadErr := supabaseClient.Storage.From(supabaseBucket).Upload(uniqueFilename, fileReader, &supabase.FileUploadOptions{ // Use correct type name
		ContentType: mimeType,
		Upsert:      false,
	})

	if uploadErr.Message != "" {
		log.Printf("ERROR: Supabase upload failed: %s", uploadErr.Message)
		return "", 0, fmt.Errorf("failed to upload file to Supabase: %s", uploadErr.Message)
	}

    fileSize = int64(len(data))
	log.Printf("File uploaded successfully to Supabase. Path: %s, Size: %d", uniqueFilename, fileSize)
	return uniqueFilename, fileSize, nil
}

// GetPublicURL generates the public URL for a given path.
func GetPublicURL(path string) string {
     if supabaseClient == nil { return "" }
    res := supabaseClient.Storage.From(supabaseBucket).GetPublicUrl(path)
    return res.SignedUrl
}