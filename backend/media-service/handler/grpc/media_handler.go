package grpc

import (
	"context"
	"log"
	"os"

	mediapb "github.com/Acad600-TPA/WEB-MJ-242/backend/media-service/genproto/proto"
	"github.com/Acad600-TPA/WEB-MJ-242/backend/media-service/repository/postgres"
	"github.com/Acad600-TPA/WEB-MJ-242/backend/media-service/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type MediaHandler struct {
	mediapb.UnimplementedMediaServiceServer
	repo *postgres.MediaRepository
}

func NewMediaHandler(repo *postgres.MediaRepository) *MediaHandler {
	return &MediaHandler{repo: repo}
}

 func (h *MediaHandler) HealthCheck(ctx context.Context, in *emptypb.Empty) (*mediapb.HealthResponse, error) {
     log.Printf("Received Media HealthCheck request")
     if err := h.repo.CheckHealth(ctx); err != nil {
         log.Printf("Media Health check failed: %v", err)
         return &mediapb.HealthResponse{Status: "Media Service is DEGRADED (DB Error)"}, nil
     }
     return &mediapb.HealthResponse{Status: "Media Service is OK"}, nil
 }

func (h *MediaHandler) UploadMedia(ctx context.Context, req *mediapb.UploadMediaRequest) (*mediapb.UploadMediaResponse, error) {
	log.Printf("Received UploadMedia request for user %d, filename %s", req.UploaderUserId, req.FileName)

	if len(req.FileContent) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "File content cannot be empty")
	}
    var uploaderIDForPath uint = 0
    if req.UploaderUserId != 0 {
        uploaderIDForPath = uint(req.UploaderUserId)
    } else {
        log.Println("UploadMedia: UploaderUserId is 0, treating as pre-auth upload.")
    }

	supabasePath, fileSize, err := utils.UploadFileToSupabase(ctx, uploaderIDForPath, req.FileName, req.MimeType, req.FileContent)
	if err != nil {
		log.Printf("Upload to Supabase failed for user %d: %v", req.UploaderUserId, err)
		return nil, status.Errorf(codes.Internal, "Failed to store media file")
	}

	media := &postgres.Media{
		UploaderUserID: uint(req.UploaderUserId),
		SupabasePath:   supabasePath,
        BucketName:     os.Getenv("SUPABASE_BUCKET_NAME"),
		MimeType:       req.MimeType,
        FileSize:       fileSize,
	}
	err = h.repo.CreateMediaMetadata(ctx, media)
	if err != nil {
		log.Printf("Failed to save media metadata for user %d, path %s: %v", req.UploaderUserId, supabasePath, err)
        // TODO: Consider deleting the file from Supabase if DB insert fails (compensation logic)
		return nil, status.Errorf(codes.Internal, "Failed to save media information")
	}

    publicURL := utils.GetPublicURL(media.SupabasePath)

	log.Printf("Media metadata saved successfully. ID: %d, Path: %s", media.ID, media.SupabasePath)

	return &mediapb.UploadMediaResponse{
		Media: &mediapb.Media{
			Id:             uint32(media.ID),
			UploaderUserId: uint32(media.UploaderUserID),
			SupabasePath:   media.SupabasePath,
            BucketName:     media.BucketName,
			MimeType:       media.MimeType,
            FileSize:       media.FileSize,
            PublicUrl:      publicURL,
            CreatedAt:      timestamppb.New(media.CreatedAt),
		},
	}, nil
}

func (h *MediaHandler) GetMediaMetadata(ctx context.Context, req *mediapb.GetMediaMetadataRequest) (*mediapb.Media, error) {
    log.Printf("Received GetMediaMetadata request for ID: %d", req.MediaId)
    if req.MediaId == 0 {
        return nil, status.Errorf(codes.InvalidArgument, "Media ID is required")
    }

    media, err := h.repo.GetMediaMetadataByID(ctx, uint(req.MediaId))
     if err != nil {
         log.Printf("GetMediaMetadata failed for ID %d: %v", req.MediaId, err)
         if err.Error() == "media not found" {
             return nil, status.Errorf(codes.NotFound, "Media metadata not found")
         }
         return nil, status.Errorf(codes.Internal, "Failed to retrieve media metadata")
     }

     publicURL := utils.GetPublicURL(media.SupabasePath)

     return &mediapb.Media{
         Id:             uint32(media.ID),
         UploaderUserId: uint32(media.UploaderUserID),
         SupabasePath:   media.SupabasePath,
         BucketName:     media.BucketName,
         MimeType:       media.MimeType,
         FileSize:       media.FileSize,
         PublicUrl:      publicURL,
         CreatedAt:      timestamppb.New(media.CreatedAt),
     }, nil
}

func (h *MediaHandler) GetMultipleMediaMetadata(ctx context.Context, req *mediapb.GetMultipleMediaMetadataRequest) (*mediapb.GetMultipleMediaMetadataResponse, error) {
    log.Printf("Received GetMultipleMediaMetadata request for %d IDs", len(req.MediaIds))
    if len(req.MediaIds) == 0 {
        return &mediapb.GetMultipleMediaMetadataResponse{MediaItems: make(map[uint32]*mediapb.Media)}, nil
    }

    uintMediaIDs := make([]uint, len(req.MediaIds))
    for i, id := range req.MediaIds {
        uintMediaIDs[i] = uint(id)
    }

    dbMediaItems, err := h.repo.GetMediaMetadataByIDs(ctx, uintMediaIDs)
    if err != nil {
        log.Printf("Error getting media by IDs: %v", err)
        return nil, status.Errorf(codes.Internal, "Failed to retrieve media metadata")
    }

    mediaMap := make(map[uint32]*mediapb.Media)
    for _, dbMedia := range dbMediaItems {
        publicURL := utils.GetPublicURL(dbMedia.SupabasePath)
        mediaMap[uint32(dbMedia.ID)] = &mediapb.Media{
            Id:             uint32(dbMedia.ID),
            UploaderUserId: uint32(dbMedia.UploaderUserID),
            SupabasePath:   dbMedia.SupabasePath,
            BucketName:     dbMedia.BucketName,
            MimeType:       dbMedia.MimeType,
            FileSize:       dbMedia.FileSize,
            PublicUrl:      publicURL,
            CreatedAt:      timestamppb.New(dbMedia.CreatedAt),
        }
    }
    return &mediapb.GetMultipleMediaMetadataResponse{MediaItems: mediaMap}, nil
}