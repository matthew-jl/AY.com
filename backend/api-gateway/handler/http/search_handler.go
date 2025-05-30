package http

import (
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/Acad600-TPA/WEB-MJ-242/backend/api-gateway/client"
	// mediapb "github.com/Acad600-TPA/WEB-MJ-242/backend/media-service/genproto/proto"
	searchpb "github.com/Acad600-TPA/WEB-MJ-242/backend/search-service/genproto/proto"
	threadpb "github.com/Acad600-TPA/WEB-MJ-242/backend/thread-service/genproto/proto"
	userpb "github.com/Acad600-TPA/WEB-MJ-242/backend/user-service/genproto/proto"
	"github.com/gin-gonic/gin"
)

type SearchHandler struct {
	searchClient *client.SearchClient
	userClient   *client.UserClient
	threadClient *client.ThreadClient
	mediaClient *client.MediaClient
}

func NewSearchHandler(sc *client.SearchClient, uc *client.UserClient, tc *client.ThreadClient, mc *client.MediaClient) *SearchHandler {
	return &SearchHandler{searchClient: sc, userClient: uc, threadClient: tc, mediaClient: mc}
}

type SearchUsersAPIResponse struct {
	Users   []FrontendUserProfile `json:"users"`
	HasMore bool                  `json:"has_more"`
}

type SearchThreadsAPIResponse struct {
	Threads []FrontendThreadData `json:"threads"`
	HasMore bool                 `json:"has_more"`
}

func (h *SearchHandler) SearchUsersHTTP(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query parameter 'q' is required"})	
		return
	}
	page, limit := parsePagination(c)

	// 1. Get User IDs from Search Service
	idResp, err := h.searchClient.SearchUsers(c.Request.Context(), &searchpb.SearchRequest{Query: query, Page: page, Limit: limit})
	if err != nil { handleGRPCError(c, "search user IDs", err); return }

	if idResp == nil || len(idResp.GetUserResults()) == 0 {
		c.JSON(http.StatusOK, SearchUsersAPIResponse{Users: []FrontendUserProfile{}, HasMore: false})
		return
	}

	// 2. Collect User IDs to fetch full profiles
	userIDsToFetch := make([]uint32, 0, len(idResp.GetUserResults()))
	for _, userResult := range idResp.GetUserResults() {
		userIDsToFetch = append(userIDsToFetch, userResult.GetId())
	}

	// 3. Fetch full user profiles from User Service
	fullProfilesMap := make(map[uint32]*userpb.User)
	// requesterID, _ := getUserIDFromContext(c)

	if len(userIDsToFetch) > 0 && h.userClient != nil {

        userObjectsResp, err := h.userClient.GetUserProfilesByIds(c.Request.Context(), &userpb.GetUserProfilesByIdsRequest{UserIds: userIDsToFetch})
        if err == nil && userObjectsResp != nil {
            fullProfilesMap = userObjectsResp.GetUsers()
        } else if err != nil {
            log.Printf("SearchUsersHTTP: Error fetching full profiles: %v", err)
        }
	}

	// 4. Map to Frontend Structure
	frontendUsers := make([]FrontendUserProfile, 0, len(idResp.GetUserResults()))
	for _, idResult := range idResp.GetUserResults() {
		if fullProfile, ok := fullProfilesMap[idResult.GetId()]; ok && fullProfile != nil {
			frontendUsers = append(frontendUsers, FrontendUserProfile{
				ID:             fullProfile.GetId(),
				Name:           fullProfile.GetName(),
				Username:       fullProfile.GetUsername(),
				Email:          fullProfile.GetEmail(), // Decide if this should be public
				ProfilePicture: fullProfile.GetProfilePicture(),
                // Bio:            fullProfile.GetBio(),
                // Add other fields from UserProfileBasic/UserCoreData
			})
		}
	}

	c.JSON(http.StatusOK, SearchUsersAPIResponse{
		Users:   frontendUsers,
		HasMore: idResp.GetHasMore(),
	})
}

func (h *SearchHandler) SearchThreadsHTTP(c *gin.Context) {
	query := c.Query("q")
	if query == "" { 
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query parameter 'q' is required"})	
		return
	}
	page, limit := parsePagination(c)
    requesterUserID, _ := getUserIDFromContext(c)

	// 1. Get Thread IDs (and snippets, author IDs) from Search Service
	idResp, err := h.searchClient.SearchThreads(c.Request.Context(), &searchpb.SearchRequest{Query: query, Page: page, Limit: limit})
	if err != nil { handleGRPCError(c, "search thread IDs", err); return }

	if idResp == nil || len(idResp.GetThreadResults()) == 0 {
		c.JSON(http.StatusOK, SearchThreadsAPIResponse{Threads: []FrontendThreadData{}, HasMore: false})
		return
	}

	// 2. Collect Thread IDs and unique Author IDs
	threadIDsToFetch := make([]uint32, 0, len(idResp.GetThreadResults()))
	authorIDsSet := make(map[uint32]bool)
	for _, threadResult := range idResp.GetThreadResults() {
		threadIDsToFetch = append(threadIDsToFetch, threadResult.GetId())
		if threadResult.GetUserId() != 0 {
			authorIDsSet[threadResult.GetUserId()] = true
		}
	}
	var authorIDsToFetchDetails []uint32
	for id := range authorIDsSet { authorIDsToFetchDetails = append(authorIDsToFetchDetails, id) }


	// 3. Fetch Full Thread Details (this will include likes, bookmarks specific to requester) and Author Details in parallel
	var wg sync.WaitGroup
	fullThreadsMap := make(map[uint32]*threadpb.Thread) // From thread.proto
	authorsMap := make(map[uint32]*userpb.User)      // From user.proto
    // mediaMap := make(map[uint32]*mediapb.Media)      // From media.proto
	// var threadsErr error 
	var authorsErr error 
	// var mediaErr error

	// Goroutine to fetch details for each thread
	threadDetailsChan := make(chan *threadpb.Thread, len(threadIDsToFetch))

	for _, threadID := range threadIDsToFetch {
		wg.Add(1)
		go func(tid uint32) {
			defer wg.Done()
			if h.threadClient == nil { log.Println("SearchThreadsHTTP: threadClient is nil"); return }
			threadDetail, err := h.threadClient.GetThread(c.Request.Context(), &threadpb.GetThreadRequest{
				ThreadId:      tid,
				CurrentUserId: &requesterUserID,
			})
			if err != nil {
				log.Printf("SearchThreadsHTTP: Error fetching detail for thread %d: %v", tid, err)
				return
			}
			threadDetailsChan <- threadDetail
		}(threadID)
	}

	// Goroutine to fetch author details
	if len(authorIDsToFetchDetails) > 0 && h.userClient != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := h.userClient.GetUserProfilesByIds(c.Request.Context(), &userpb.GetUserProfilesByIdsRequest{UserIds: authorIDsToFetchDetails})
			if err != nil { authorsErr = err; return }
			if resp != nil { authorsMap = resp.GetUsers() }
		}()
	}

    go func() {
        wg.Wait()
        close(threadDetailsChan)
    }()

    for threadDetail := range threadDetailsChan {
        if threadDetail != nil {
            fullThreadsMap[threadDetail.GetId()] = threadDetail
        }
    }


    if authorsErr != nil { log.Printf("SearchThreadsHTTP: Error fetching all author details: %v", authorsErr) }
    // TODO: Media hydration for threads from search results.

	// 4. Map to Frontend Structure
	frontendThreads := make([]FrontendThreadData, 0, len(idResp.GetThreadResults()))
	for _, searchResultThread := range idResp.GetThreadResults() {
		if fullThread, ok := fullThreadsMap[searchResultThread.GetId()]; ok && fullThread != nil {
			feThread := mapProtoThreadToFrontend(fullThread, authorsMap, nil) // Pass nil for mediaMap for now
            // Add the snippet from search result if different from full content
            if searchResultThread.GetContentSnippet() != "" && searchResultThread.GetContentSnippet() != fullThread.GetContent() {
                // feThread.Content = searchResultThread.GetContentSnippet() // Or add a new field for snippet
            }
			frontendThreads = append(frontendThreads, feThread)
		}
	}


	c.JSON(http.StatusOK, SearchThreadsAPIResponse{
		Threads: frontendThreads,
		HasMore: idResp.GetHasMore(),
	})
}

func (h *SearchHandler) GetTrendingHashtagsHTTP(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 20 {
		limit = 10
	}

	grpcReq := &searchpb.GetTrendingHashtagsRequest{Limit: int32(limit)}
	resp, err := h.searchClient.GetTrendingHashtags(c.Request.Context(), grpcReq)
	if err != nil {
		handleGRPCError(c, "get trending hashtags", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *SearchHandler) SearchServiceHealthHTTP(c *gin.Context) {
    resp, err := h.searchClient.HealthCheck(c.Request.Context())
    if err != nil {
        handleGRPCError(c, "search service health check", err)
        return
    }
    c.JSON(http.StatusOK, resp)
}