package grpc

import (
	"context"
	"log"
	"sort"
	"strings"

	searchpb "github.com/Acad600-TPA/WEB-MJ-242/backend/search-service/genproto/proto"
	"github.com/Acad600-TPA/WEB-MJ-242/backend/search-service/repository"

	"github.com/Acad600-TPA/WEB-MJ-242/backend/search-service/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type SearchHandler struct {
	searchpb.UnimplementedSearchServiceServer
	repo *repository.SearchRepository
}

const fuzzySearchSimilarityThreshold = 0.4 // 0.0 to 1.0
const initialDbFetchLimitMultiplier = 5

func NewSearchHandler(repo *repository.SearchRepository) *SearchHandler {
	return &SearchHandler{repo: repo}
}

func (h *SearchHandler) HealthCheck(ctx context.Context, in *emptypb.Empty) (*searchpb.HealthResponse, error) {
    log.Println("Search Service Health Check received")
    if err := h.repo.CheckHealth(ctx); err != nil {
        log.Printf("Search Service health check failed: %v", err)
        return &searchpb.HealthResponse{Status: "Search Service DEGRADED"}, nil
    }
	return &searchpb.HealthResponse{Status: "Search Service OK"}, nil
}

func (h *SearchHandler) SearchUsers(ctx context.Context, req *searchpb.SearchRequest) (*searchpb.SearchUserIDsResponse, error) {
	log.Printf("SearchUsers request: Query='%s', Page=%d, Limit=%d", req.Query, req.Page, req.Limit)
	if req.Query == "" {
		return nil, status.Errorf(codes.InvalidArgument, "Search query cannot be empty")
	}
	
	// Desired final limit and page
	finalLimit := int(req.Limit)
	if finalLimit <= 0 || finalLimit > 50 { finalLimit = 10 }
	
	// the 'page' and 'limit' in the request are for the final, fuzzy-matched result set
    initialDbFetchLimit := finalLimit * initialDbFetchLimitMultiplier 
    // For simplicity, apply fuzzy match to a larger candidate pool, then paginate that

	dbUsers, err := h.repo.SearchUsers(ctx, req.Query, initialDbFetchLimit, 0)
	if err != nil {
		log.Printf("Error searching users in repo: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to search users (db)")
	}

	if len(dbUsers) == 0 {
		return &searchpb.SearchUserIDsResponse{UserResults: []*searchpb.UserIDResult{}, HasMore: false}, nil
	}

	type scoredUserID struct {
		id       uint32
		similarity float64
	}
	var candidates []scoredUserID

	normalizedQuery := strings.ToLower(req.Query)

	for _, u := range dbUsers {
		// Calculate similarity against multiple fields
		simName := utils.CalculateSimilarityNormalized(u.Name, normalizedQuery)
		simUsername := utils.CalculateSimilarityNormalized(u.Username, normalizedQuery)
		simBio := 0.0
		if u.Bio != "" {
			simBio = utils.CalculateSimilarityNormalized(u.Bio, normalizedQuery)
		}


		// Take the highest similarity score among the fields
		maxSimilarity := simName
		if simUsername > maxSimilarity { maxSimilarity = simUsername }
		if simBio > maxSimilarity { maxSimilarity = simBio }
		

		if maxSimilarity >= fuzzySearchSimilarityThreshold {
			candidates = append(candidates, scoredUserID{
				id: uint32(u.ID),
				similarity: maxSimilarity,
			})
		}
	}

	// Sort candidates by similarity (descending)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].similarity > candidates[j].similarity
	})

    // Apply pagination to the fuzzy-matched and sorted results
    page := int(req.Page)
    if page <= 0 { page = 1 }
    offset := (page - 1) * finalLimit

    results := []*searchpb.UserIDResult{}
    hasMore := false
    if offset < len(candidates) {
        end := offset + finalLimit
        if end > len(candidates) {
            end = len(candidates)
        }
        for _, sc := range candidates[offset:end] {
            results = append(results, &searchpb.UserIDResult{Id: sc.id})
        }
        if end < len(candidates) {
            hasMore = true
        }
    }


	log.Printf("Fuzzy SearchUsers: Query='%s', Found %d candidates, Returning %d results, HasMore: %v", req.Query, len(candidates), len(results), hasMore)
	return &searchpb.SearchUserIDsResponse{UserResults: results, HasMore: hasMore}, nil
}

func (h *SearchHandler) SearchThreads(ctx context.Context, req *searchpb.SearchRequest) (*searchpb.SearchThreadIDsResponse, error) {
	log.Printf("SearchThreads request: Query='%s', Page=%d, Limit=%d", req.Query, req.Page, req.Limit)
    if req.Query == "" { return nil, status.Errorf(codes.InvalidArgument, "Search query cannot be empty") }

	finalLimit := int(req.Limit)
	if finalLimit <= 0 || finalLimit > 50 { finalLimit = 10 }
    initialDbFetchLimit := finalLimit * initialDbFetchLimitMultiplier

	dbThreads, err := h.repo.SearchThreads(ctx, req.Query, initialDbFetchLimit, 0)
	if err != nil {
		log.Printf("Error searching threads in repo: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to search threads")
	}

	if len(dbThreads) == 0 {
		return &searchpb.SearchThreadIDsResponse{ThreadResults: []*searchpb.ThreadIDResult{}, HasMore: false}, nil
	}
	
	type scoredThread struct {
		idResult     *searchpb.ThreadIDResult
		similarity float64
	}
	var candidates []scoredThread
	normalizedQuery := strings.ToLower(req.Query)

	for _, t := range dbThreads {
		similarity := utils.CalculateSimilarityPerWord(t.Content, normalizedQuery)
		if similarity >= fuzzySearchSimilarityThreshold {
			candidates = append(candidates, scoredThread{
				idResult: &searchpb.ThreadIDResult{
					Id:             uint32(t.ID),
					ContentSnippet: firstNChars(t.Content, 150),
					UserId:         uint32(t.UserID),
				},
				similarity: similarity,
			})
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].similarity > candidates[j].similarity
	})

	for _, c := range candidates {
		log.Printf("Candidate: ID=%d, Similarity=%.2f, Content=%s", c.idResult.Id, c.similarity, c.idResult.ContentSnippet)
	}

    // Apply pagination
    page := int(req.Page)
    if page <= 0 { page = 1 }
    offset := (page - 1) * finalLimit
    
    paginatedCandidates := []*searchpb.ThreadIDResult{}
    hasMore := false
    if offset < len(candidates) {
        end := offset + finalLimit
        if end > len(candidates) { end = len(candidates) }
        for _, sc := range candidates[offset:end] {
            paginatedCandidates = append(paginatedCandidates, sc.idResult)
        }
        if end < len(candidates) { hasMore = true }
    }
    
	log.Printf("Fuzzy SearchThreads: Query='%s', Found %d candidates, Returning %d results, HasMore: %v", req.Query, len(candidates), len(paginatedCandidates), hasMore)
	return &searchpb.SearchThreadIDsResponse{ThreadResults: paginatedCandidates, HasMore: hasMore}, nil
}

func (h *SearchHandler) GetTrendingHashtags(ctx context.Context, req *searchpb.GetTrendingHashtagsRequest) (*searchpb.GetTrendingHashtagsResponse, error) {
    log.Printf("GetTrendingHashtags request: Limit=%d", req.Limit)
    limit := req.Limit
    if limit <= 0 || limit > 20 {
        limit = 10
    }
    tagsWithScores, err := h.repo.GetTrendingHashtags(ctx, int64(limit))
    if err != nil {
        log.Printf("Error getting trending hashtags: %v", err)
        return nil, status.Errorf(codes.Internal, "Failed to retrieve trending hashtags")
    }

	protoTrendingHashtags := make([]*searchpb.TrendingHashtag, len(tagsWithScores))
    for i, tagScore := range tagsWithScores {
        protoTrendingHashtags[i] = &searchpb.TrendingHashtag{
            Tag:   tagScore.Tag,
            Count: int32(tagScore.Count),
        }
    }

    return &searchpb.GetTrendingHashtagsResponse{TrendingHashtags: protoTrendingHashtags}, nil
}

func (h *SearchHandler) IncrementHashtagCounts(ctx context.Context, req *searchpb.IncrementHashtagCountsRequest) (*searchpb.IncrementHashtagCountsResponse, error) {
	if len(req.Hashtags) == 0 {
		return &searchpb.IncrementHashtagCountsResponse{Success: false}, nil
	}
	h.repo.IncrementHashtagCounts(ctx, req.Hashtags)
	return &searchpb.IncrementHashtagCountsResponse{Success: true}, nil
}

func (h *SearchHandler) GetTopUsersToFollow(ctx context.Context, req *searchpb.GetTopUsersToFollowRequest) (*searchpb.SearchUserIDsResponse, error) {
	log.Printf("GetTopUsersToFollow request: Limit=%d, ExcludeUserID=%d", req.Limit, req.GetExcludeUserId())
	limit := int(req.Limit)
	if limit <= 0 || limit > 10 {
		limit = 3
	}

    var excludeID *uint
    if req.GetExcludeUserId() != 0 {
        uid := uint(req.GetExcludeUserId())
        excludeID = &uid
    }

	topDBUsers, err := h.repo.GetTopUsersByFollowerCount(ctx, limit, excludeID)
	if err != nil {
		log.Printf("Error getting top users from repo: %v", err)
		return nil, status.Errorf(codes.Internal, "Failed to retrieve top users to follow")
	}

	userResults := make([]*searchpb.UserIDResult, len(topDBUsers))
	for i, u := range topDBUsers {
		userResults[i] = &searchpb.UserIDResult{
			Id: uint32(u.ID),
		}
	}

	return &searchpb.SearchUserIDsResponse{UserResults: userResults, HasMore: false}, nil
}

// Helper for pagination
func getLimitOffsetSearch(page, limit int32) (int, int) {
	p := int(page); l := int(limit)
	if l <= 0 || l > 50 { l = 20 }
	if p <= 0 { p = 1 }
	return l, (p - 1) * l
}

func firstNChars(s string, n int) string {
    if len(s) <= n { return s }
    return s[:n] + "..."
}