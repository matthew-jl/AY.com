package http

import (
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/Acad600-TPA/WEB-MJ-242/backend/api-gateway/client"
	mediapb "github.com/Acad600-TPA/WEB-MJ-242/backend/media-service/genproto/proto"
	"github.com/gin-gonic/gin"
)

type MediaHandler struct {
	mediaClient *client.MediaClient
}

func NewMediaHandler(mediaClient *client.MediaClient) *MediaHandler {
	return &MediaHandler{mediaClient: mediaClient}
}

// UploadMedia handles file uploads from multipart form.
func (h *MediaHandler) UploadMedia(c *gin.Context) {
	var uploaderID uint32 = 0 // Default to 0 for anonymous/pre-auth uploads
    userIDFromCtx, ok := c.Get("userID")
    if ok {
        if uid, typeOK := userIDFromCtx.(uint); typeOK {
            uploaderID = uint32(uid)
        }
    }
    log.Printf("UploadMedia attempt by UserID (0 if anonymous): %d", uploaderID)

	// Get the file from the form
	fileHeader, err := c.FormFile("media_file") // "media_file" is the expected form field name
	if err != nil {
		log.Printf("Error getting form file: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing or invalid 'media_file' in request: " + err.Error()})
		return
	}

	// TODO: Add file size validation BEFORE reading into memory
	// Example: Check fileHeader.Size against a limit (e.g., 10 * 1024 * 1024 for 10MB)
	maxSize := int64(25 * 1024 * 1024) // 25 MB limit example
	if fileHeader.Size > maxSize {
		 c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "File size exceeds the limit of " + strconv.FormatInt(maxSize/1024/1024, 10) + "MB"})
		 return
	}

	// Open the file
	file, err := fileHeader.Open()
	if err != nil {
		log.Printf("Error opening uploaded file: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process uploaded file"})
		return
	}
	defer file.Close()

	// Read the file content into a byte slice
	// WARNING: Reads the entire file into memory. For very large files, streaming is better.
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		log.Printf("Error reading uploaded file content: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read uploaded file"})
		return
	}

	// Get MIME type
	mimeType := fileHeader.Header.Get("Content-Type")
	// TODO: Add MIME type validation (check against allowed types)
	if mimeType == "" {
		// Try to detect, but relying on client header is better
		mimeType = http.DetectContentType(fileBytes)
		log.Printf("Detected MIME type: %s for file %s", mimeType, fileHeader.Filename)
		// Add check here if detection failed or type is not allowed
	}


	// Prepare gRPC request
	grpcReq := &mediapb.UploadMediaRequest{
		UploaderUserId: uploaderID,
		FileName:       fileHeader.Filename, // Use original filename
		MimeType:       mimeType,
		FileContent:    fileBytes,
	}

	// Call Media Service
	resp, err := h.mediaClient.UploadMedia(c.Request.Context(), grpcReq)
	if err != nil {
		handleGRPCError(c, "upload media", err) // Use shared helper
		return
	}

	// Return the response from Media Service (contains ID, URL, etc.)
	c.JSON(http.StatusCreated, resp)
}