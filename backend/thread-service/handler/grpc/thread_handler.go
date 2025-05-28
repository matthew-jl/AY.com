package grpc

import (
	"context"
	"errors"
	"log"

	threadpb "github.com/Acad600-TPA/WEB-MJ-242/backend/thread-service/genproto/proto"
	"github.com/Acad600-TPA/WEB-MJ-242/backend/thread-service/repository/postgres"
	"github.com/lib/pq"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ThreadHandler struct {
	threadpb.UnimplementedThreadServiceServer
	repo *postgres.ThreadRepository
}

func NewThreadHandler(repo *postgres.ThreadRepository) *ThreadHandler {
	return &ThreadHandler{repo: repo}
}

func (h *ThreadHandler) HealthCheck(ctx context.Context, in *emptypb.Empty) (*threadpb.HealthResponse, error) {
    log.Printf("Received Thread HealthCheck request")
     if err := h.repo.CheckHealth(ctx); err != nil {
         log.Printf("Thread Health check failed: %v", err)
         return &threadpb.HealthResponse{Status: "Thread Service is DEGRADED (DB Error)"}, nil
     }
     return &threadpb.HealthResponse{Status: "Thread Service is OK"}, nil
}

func (h *ThreadHandler) CreateThread(ctx context.Context, req *threadpb.CreateThreadRequest) (*threadpb.Thread, error) {
	log.Printf("Received CreateThread request for user %d", req.UserId)
	if req.UserId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "User ID is required")
	}
	if req.Content == "" && len(req.MediaIds) == 0 {
         return nil, status.Errorf(codes.InvalidArgument, "Thread must have content or media")
    }
    // Add more validation: content length, media ID existence (by calling media service?), etc.

	thread := &postgres.Thread{
		UserID:           uint(req.UserId),
		Content:          req.Content,
		ReplyRestriction: req.ReplyRestriction.String(),
        MediaIDs:         uint32SliceToInt64Array(req.MediaIds),
	}
    if req.GetParentThreadId() != 0 {
         parentID := uint(req.GetParentThreadId())
         thread.ParentThreadID = &parentID
    }
     if req.GetCommunityId() != 0 {
         communityID := uint(req.GetCommunityId())
         thread.CommunityID = &communityID
    }
    if req.ScheduledAt != nil && req.ScheduledAt.IsValid() {
         scheduledTime := req.ScheduledAt.AsTime()
         thread.ScheduledAt = &scheduledTime
    }
     // IsAdvertisement would be set based on user role check, likely done in gateway or here if user info passed

	err := h.repo.CreateThread(ctx, thread)
	if err != nil {
		log.Printf("Failed to create thread for user %d: %v", req.UserId, err)
		return nil, status.Errorf(codes.Internal, "Could not create thread")
	}

    log.Printf("Thread created successfully. ID: %d", thread.ID)

	// Map DB model back to proto response
	return mapThreadToProto(thread), nil
}

func (h *ThreadHandler) hydrateThreadInteractions(ctx context.Context, tProto *threadpb.Thread, currentUserID uint32) {
	if tProto == nil {
		return
	}

	likeCount, bookmarkCount, err := h.repo.GetInteractionCountsForThread(ctx, uint(tProto.Id))
	if err != nil {
		log.Printf("Error getting interaction counts for thread %d: %v", tProto.Id, err)
	} else {
		tProto.LikeCount = int32(likeCount)
		tProto.BookmarkCount = int32(bookmarkCount)
		// TODO: Populate reply_count, repost_count when implemented
	}

	if currentUserID != 0 {
		isLiked, err := h.repo.CheckUserInteraction(ctx, uint(currentUserID), uint(tProto.Id), "like")
		if err != nil {
			log.Printf("Error checking like status for user %d, thread %d: %v", currentUserID, tProto.Id, err)
		} else {
			tProto.IsLikedByCurrentUser = isLiked
		}

		isBookmarked, err := h.repo.CheckUserInteraction(ctx, uint(currentUserID), uint(tProto.Id), "bookmark")
		if err != nil {
			log.Printf("Error checking bookmark status for user %d, thread %d: %v", currentUserID, tProto.Id, err)
		} else {
			tProto.IsBookmarkedByCurrentUser = isBookmarked
		}
	}
}


func (h *ThreadHandler) GetThread(ctx context.Context, req *threadpb.GetThreadRequest) (*threadpb.Thread, error) {
     log.Printf("Received GetThread request for ID: %d", req.ThreadId)
     if req.ThreadId == 0 { return nil, status.Errorf(codes.InvalidArgument, "Thread ID is required") }

     thread, err := h.repo.GetThreadByID(ctx, uint(req.ThreadId))
     if err != nil {
          log.Printf("GetThread failed for ID %d: %v", req.ThreadId, err)
         if err.Error() == "thread not found" { return nil, status.Errorf(codes.NotFound, "Thread not found") }
         return nil, status.Errorf(codes.Internal, "Failed to retrieve thread")
     }
     
     tProto := mapThreadToProto(thread)
     h.hydrateThreadInteractions(ctx, tProto, *req.CurrentUserId)

     return tProto, nil
}

func (h *ThreadHandler) DeleteThread(ctx context.Context, req *threadpb.DeleteThreadRequest) (*emptypb.Empty, error) {
    log.Printf("Received DeleteThread request for ID: %d by User: %d", req.ThreadId, req.UserId)
    if req.ThreadId == 0 || req.UserId == 0 {
        return nil, status.Errorf(codes.InvalidArgument, "Thread ID and User ID are required")
    }

    // 1. Get thread to verify ownership
    thread, err := h.repo.GetThreadByID(ctx, uint(req.ThreadId))
    if err != nil {
         if err.Error() == "thread not found" { return nil, status.Errorf(codes.NotFound, "Thread not found") }
         return nil, status.Errorf(codes.Internal, "Failed to retrieve thread for deletion check")
    }

    // 2. Check ownership
    if thread.UserID != uint(req.UserId) {
         log.Printf("User %d attempted to delete thread %d owned by user %d", req.UserId, req.ThreadId, thread.UserID)
         return nil, status.Errorf(codes.PermissionDenied, "You do not have permission to delete this thread")
    }

     // 3. Perform soft delete (GORM handles this if DeletedAt field exists)
     err = h.repo.PerformSoftDelete(ctx, uint(req.ThreadId))
     if err != nil {
		log.Printf("Failed to delete thread %d: %v", req.ThreadId, err)
        if errors.Is(err, errors.New("thread not found")) { // Check error from repo delete
            return nil, status.Errorf(codes.NotFound, "Thread not found for deletion")
        }
		return nil, status.Errorf(codes.Internal, "Failed to delete thread")
	}

     log.Printf("Thread %d soft deleted successfully by user %d", req.ThreadId, req.UserId)
     return &emptypb.Empty{}, nil
}


// --- Interaction Handlers ---

func (h *ThreadHandler) LikeThread(ctx context.Context, req *threadpb.InteractThreadRequest) (*emptypb.Empty, error) {
    log.Printf("Received LikeThread request for Thread %d by User %d", req.ThreadId, req.UserId)
    if req.ThreadId == 0 || req.UserId == 0 { return nil, status.Errorf(codes.InvalidArgument, "Thread ID and User ID required") }

    err := h.repo.AddInteraction(ctx, uint(req.UserId), uint(req.ThreadId), "like")
     if err != nil {
        if err.Error() == "interaction already exists" { return &emptypb.Empty{}, nil } // Idempotent like
        if err.Error() == "user or thread not found for interaction" { return nil, status.Errorf(codes.NotFound, "Cannot like thread: user or thread not found") }
         log.Printf("Failed to add like interaction: %v", err)
         return nil, status.Errorf(codes.Internal, "Failed to process like")
     }
     return &emptypb.Empty{}, nil
}

func (h *ThreadHandler) UnlikeThread(ctx context.Context, req *threadpb.InteractThreadRequest) (*emptypb.Empty, error) {
    log.Printf("Received UnlikeThread request for Thread %d by User %d", req.ThreadId, req.UserId)
    if req.ThreadId == 0 || req.UserId == 0 { return nil, status.Errorf(codes.InvalidArgument, "Thread ID and User ID required") }

     err := h.repo.RemoveInteraction(ctx, uint(req.UserId), uint(req.ThreadId), "like")
     if err != nil {
          if err.Error() == "interaction not found" { return &emptypb.Empty{}, nil } // Idempotent unlike
          log.Printf("Failed to remove like interaction: %v", err)
          return nil, status.Errorf(codes.Internal, "Failed to process unlike")
     }
     return &emptypb.Empty{}, nil
}

 func (h *ThreadHandler) BookmarkThread(ctx context.Context, req *threadpb.InteractThreadRequest) (*emptypb.Empty, error) {
     log.Printf("Received BookmarkThread request for Thread %d by User %d", req.ThreadId, req.UserId)
     if req.ThreadId == 0 || req.UserId == 0 { return nil, status.Errorf(codes.InvalidArgument, "Thread ID and User ID required") }
     err := h.repo.AddInteraction(ctx, uint(req.UserId), uint(req.ThreadId), "bookmark")
      if err != nil {
        if err.Error() == "interaction already exists" { return &emptypb.Empty{}, nil }
        if err.Error() == "user or thread not found for interaction" { return nil, status.Errorf(codes.NotFound, "Cannot bookmark thread: user or thread not found") }
          log.Printf("Failed to add bookmark interaction: %v", err)
          return nil, status.Errorf(codes.Internal, "Failed to process bookmark")
      }
      return &emptypb.Empty{}, nil
 }

 func (h *ThreadHandler) UnbookmarkThread(ctx context.Context, req *threadpb.InteractThreadRequest) (*emptypb.Empty, error) {
     log.Printf("Received UnbookmarkThread request for Thread %d by User %d", req.ThreadId, req.UserId)
     if req.ThreadId == 0 || req.UserId == 0 { return nil, status.Errorf(codes.InvalidArgument, "Thread ID and User ID required") }
      err := h.repo.RemoveInteraction(ctx, uint(req.UserId), uint(req.ThreadId), "bookmark")
      if err != nil {
          if err.Error() == "interaction not found" { return &emptypb.Empty{}, nil }
          log.Printf("Failed to remove bookmark interaction: %v", err)
          return nil, status.Errorf(codes.Internal, "Failed to process unbookmark")
      }
      return &emptypb.Empty{}, nil
 }

 func (h *ThreadHandler) GetFeedThreads(ctx context.Context, req *threadpb.GetFeedThreadsRequest) (*threadpb.GetFeedThreadsResponse, error) {
	log.Printf("ThreadSvc: GetFeedThreads. Requester: %d, Type: %s, Exclude: %v, Include: %v",
		req.GetCurrentUserId(), req.GetFeedType(), req.GetExcludeUserIds(), req.GetIncludeOnlyUserIds())


	limit, offset := getLimitOffset(req.Page, req.Limit)

	params := postgres.GetThreadsParams{
		Limit:              limit,
		Offset:             offset,
		ExcludeUserIDs:     uint32SliceToUint(req.GetExcludeUserIds()),
		IncludeOnlyUserIDs: uint32SliceToUint(req.GetIncludeOnlyUserIds()),
	}

	dbThreads, err := h.repo.GetThreads(ctx, params)
	if err != nil {
		log.Printf("Failed to get feed threads from repo: %v", err)
		return nil, status.Errorf(codes.Internal, "Could not retrieve feed")
	}

	protoThreads := make([]*threadpb.Thread, 0, len(dbThreads))
	if len(dbThreads) > 0 {
		threadIDs := make([]uint, len(dbThreads))
		for i, t := range dbThreads {
			threadIDs[i] = t.ID
		}

		countsMap, errCounts := h.repo.GetInteractionCountsForMultipleThreads(ctx, threadIDs)
		if errCounts != nil {
			log.Printf("Error fetching batch interaction counts: %v", errCounts)
		}

		userInteractionsMap := make(map[uint]map[string]bool)
		if req.GetCurrentUserId() != 0 {
			userInteractionsMap, err = h.repo.CheckUserInteractionsForMultipleThreads(ctx, uint(req.GetCurrentUserId()), threadIDs)
			if err != nil {
				log.Printf("Error fetching batch user interactions: %v", err)
			}
		}

		for i := range dbThreads {
			tProto := mapThreadToProto(&dbThreads[i])

			if threadCounts, ok := countsMap[dbThreads[i].ID]; ok {
				tProto.LikeCount = int32(threadCounts["like"])
				tProto.BookmarkCount = int32(threadCounts["bookmark"])
				// Populate other counts (reply, repost) when available
			}

			if userThreadInteractions, ok := userInteractionsMap[dbThreads[i].ID]; ok {
				tProto.IsLikedByCurrentUser = userThreadInteractions["like"]
				tProto.IsBookmarkedByCurrentUser = userThreadInteractions["bookmark"]
			}
			protoThreads = append(protoThreads, tProto)
		}
	}

	hasMore := len(dbThreads) == limit
	log.Printf("Returning %d hydrated threads for feed request.", len(protoThreads))

	return &threadpb.GetFeedThreadsResponse{
		Threads: protoThreads,
		HasMore: hasMore,
	}, nil
}

func (h *ThreadHandler) GetUserThreads(ctx context.Context, req *threadpb.GetUserThreadsRequest) (*threadpb.GetUserThreadsResponse, error) {
	log.Printf("ThreadSvc: GetUserThreads. Target: %d, Requester: %d, Type: %s, Exclude: %v",
		req.TargetUserId, req.GetRequesterUserId(), req.ThreadType, req.GetExcludeUserIds())
	
	if req.TargetUserId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Target User ID is required")
	}

	limit, offset := getLimitOffset(req.Page, req.Limit)
	var dbThreads []postgres.Thread
	var err error

	// Fetch threads based on type
	if req.ThreadType == "likes" {
		dbThreads, err = h.repo.GetLikedThreadsByUser(ctx, uint(req.TargetUserId), limit, offset)
	} else {
		// For posts, replies, media
		params := postgres.GetThreadsParams{
			Limit:       limit,
			Offset:      offset,
			ByUserID:    pointToUint(uint(req.TargetUserId)),
			FeedTabType: req.ThreadType,
			ExcludeUserIDs: uint32SliceToUint(req.GetExcludeUserIds()),
		}
		dbThreads, err = h.repo.GetThreads(ctx, params)
	}

	if err != nil {
		log.Printf("Failed to get user threads from repo (type: %s, user: %d): %v", req.ThreadType, req.TargetUserId, err)
		return nil, status.Errorf(codes.Internal, "Could not retrieve user threads")
	}

	protoThreads := make([]*threadpb.Thread, 0, len(dbThreads))
	if len(dbThreads) > 0 {
		threadIDs := make([]uint, len(dbThreads))
		for i, t := range dbThreads { threadIDs[i] = t.ID }

		countsMap, errCounts := h.repo.GetInteractionCountsForMultipleThreads(ctx, threadIDs)
		if errCounts != nil { log.Printf("Error fetching batch interaction counts for user threads: %v", errCounts) }

		userInteractionsMap := make(map[uint]map[string]bool)
		if req.GetRequesterUserId() != 0 {
			userInteractionsMap, err = h.repo.CheckUserInteractionsForMultipleThreads(ctx, uint(req.GetRequesterUserId()), threadIDs)
			if err != nil { log.Printf("Error fetching batch user interactions for user threads: %v", err) }
		}

		for i := range dbThreads {
			tProto := mapThreadToProto(&dbThreads[i])
			if threadCounts, ok := countsMap[dbThreads[i].ID]; ok {
				tProto.LikeCount = int32(threadCounts["like"])
				tProto.BookmarkCount = int32(threadCounts["bookmark"])
			}
			if userThreadInteractions, ok := userInteractionsMap[dbThreads[i].ID]; ok {
				tProto.IsLikedByCurrentUser = userThreadInteractions["like"]
				tProto.IsBookmarkedByCurrentUser = userThreadInteractions["bookmark"]
			}
			protoThreads = append(protoThreads, tProto)
		}
	}

	hasMore := len(dbThreads) == limit
	log.Printf("Returning %d hydrated threads for GetUserThreads request.", len(protoThreads))

	return &threadpb.GetUserThreadsResponse{
		Threads: protoThreads,
		HasMore: hasMore,
	}, nil
}


// --- Helper Functions ---

func mapThreadToProto(t *postgres.Thread) *threadpb.Thread {
    if t == nil { return nil }
    protoThread := &threadpb.Thread{
        Id:              uint32(t.ID),
        UserId:          uint32(t.UserID),
        Content:         t.Content,
        ReplyRestriction: mapStringToReplyRestriction(t.ReplyRestriction),
        PostedAt:        timestamppb.New(t.PostedAt),
        IsAdvertisement: t.IsAdvertisement,
        MediaIds:        int64ArrayToUint32Slice(t.MediaIDs),
        CreatedAt:       timestamppb.New(t.CreatedAt),
        // Interaction counts need to be populated separately
    }
    if t.ParentThreadID != nil {
        parentID := uint32(*t.ParentThreadID)
		protoThread.ParentThreadId = &parentID
    }
    if t.CommunityID != nil {
        communityID := uint32(*t.CommunityID)
        protoThread.CommunityId = &communityID
    }
     if t.ScheduledAt != nil {
         protoThread.ScheduledAt = timestamppb.New(*t.ScheduledAt)
     }
    return protoThread
}

func mapStringToReplyRestriction(s string) threadpb.ReplyRestriction {
    switch s {
    case "following": return threadpb.ReplyRestriction_FOLLOWING
    case "verified": return threadpb.ReplyRestriction_VERIFIED
    case "everyone": fallthrough
    default:         return threadpb.ReplyRestriction_EVERYONE
    }
}

func int64ArrayToUint32Slice(arr pq.Int64Array) []uint32 {
    if arr == nil { return nil }
    res := make([]uint32, len(arr))
    for i, v := range arr {
        res[i] = uint32(v)
    }
    return res
}

func uint32SliceToInt64Array(slice []uint32) pq.Int64Array {
    if slice == nil { return nil }
    res := make(pq.Int64Array, len(slice))
    for i, v := range slice {
        res[i] = int64(v)
    }
    return res
}

func pointToUint(val uint) *uint {
	return &val
}

func getLimitOffset(page, limit int32) (int, int) {
	p := int(page); l := int(limit)
	if l <= 0 || l > 50 { l = 20 }
	if p <= 0 { p = 1 }
	return l, (p - 1) * l
}

func uint32SliceToUint(slice []uint32) []uint {
	if slice == nil {
		return nil
	}
	res := make([]uint, len(slice))
	for i, v := range slice {
		res[i] = uint(v)
	}
	return res
}
