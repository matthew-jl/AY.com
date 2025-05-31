package http

import (
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/Acad600-TPA/WEB-MJ-242/backend/api-gateway/client"
	mediapb "github.com/Acad600-TPA/WEB-MJ-242/backend/media-service/genproto/proto"
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

	// 1. Get Thread ID Results from Search Service
	searchServiceResp, err := h.searchClient.SearchThreads(c.Request.Context(), &searchpb.SearchRequest{Query: query, Page: page, Limit: limit})
	if err != nil { handleGRPCError(c, "search thread IDs", err); return }

	if searchServiceResp == nil || len(searchServiceResp.GetThreadResults()) == 0 {
		c.JSON(http.StatusOK, SearchThreadsAPIResponse{Threads: []FrontendThreadData{}, HasMore: false})
		return
	}

	// 2. Collect IDs
	threadIDsToFetchFullDetails := make([]uint32, 0, len(searchServiceResp.GetThreadResults()))
	authorIDsSet := make(map[uint32]bool)      // To collect unique author IDs

	for _, searchResult := range searchServiceResp.GetThreadResults() {
		threadIDsToFetchFullDetails = append(threadIDsToFetchFullDetails, searchResult.GetId())
		if searchResult.GetUserId() != 0 {
			authorIDsSet[searchResult.GetUserId()] = true
		}
	}

	var authorIDsToFetchProfiles []uint32
	for id := range authorIDsSet { authorIDsToFetchProfiles = append(authorIDsToFetchProfiles, id) }

	// 3. Fetch Full Thread Details, Author Details, and Media Details in Parallel
	var wg sync.WaitGroup
	fullThreadsDataMap := make(map[uint32]*threadpb.Thread)
	authorsProfileMap := make(map[uint32]*userpb.User)
	mediaMetadataMap := make(map[uint32]*mediapb.Media)
	var authorsErr, mediaErr error

	fetchedThreadsChan := make(chan *threadpb.Thread, len(threadIDsToFetchFullDetails))

	// Launch goroutines for fetching individual full thread details
	for _, threadID := range threadIDsToFetchFullDetails {
		wg.Add(1)
		go func(tid uint32, reqUID uint32) {
			defer wg.Done()
			if h.threadClient == nil {
				log.Println("SearchThreadsHTTP: threadClient is nil")
				return
			}
			threadDetail, err := h.threadClient.GetThread(c.Request.Context(), &threadpb.GetThreadRequest{
				ThreadId:      tid,
				CurrentUserId: &reqUID,
			})
			if err != nil {
				log.Printf("SearchThreadsHTTP: Error fetching full detail for thread %d: %v", tid, err)
				return
			}
			fetchedThreadsChan <- threadDetail
		}(threadID, requesterUserID)
	}

	// Launch goroutine for fetching author profiles
	if len(authorIDsToFetchProfiles) > 0 && h.userClient != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := h.userClient.GetUserProfilesByIds(c.Request.Context(), &userpb.GetUserProfilesByIdsRequest{UserIds: authorIDsToFetchProfiles})
			if err != nil {
				authorsErr = err
				return
			}
			if resp != nil {
				authorsProfileMap = resp.GetUsers()
			}
		}()
	}

	allMediaIDsToFetch := make(chan []uint32, 1)

	// Goroutine to collect thread results and trigger media fetch
	go func() {
		tempAllMediaIDsSet := make(map[uint32]bool)

		for threadDetail := range fetchedThreadsChan {
			if threadDetail != nil {
				fullThreadsDataMap[threadDetail.GetId()] = threadDetail
				for _, mediaID := range threadDetail.GetMediaIds() {
					if mediaID != 0 {
						tempAllMediaIDsSet[mediaID] = true
					}
				}
			}
		}

		// After all threads are processed from the channel, send media IDs to be fetched
		var mediaIDs []uint32
		for id := range tempAllMediaIDsSet {
			mediaIDs = append(mediaIDs, id)
		}
		allMediaIDsToFetch <- mediaIDs
		close(allMediaIDsToFetch)
	}()

    go func() {
        wg.Wait()
        close(fetchedThreadsChan)
    }()


    // Now fetch media based on IDs collected from fullThreadsDataMap
    mediaIDsFromFullThreads := <-allMediaIDsToFetch

    if len(mediaIDsFromFullThreads) > 0 && h.mediaClient != nil {
        resp, err := h.mediaClient.GetMultipleMediaMetadata(c.Request.Context(), &mediapb.GetMultipleMediaMetadataRequest{MediaIds: mediaIDsFromFullThreads})
        if err != nil {
            mediaErr = err
        } else if resp != nil {
            mediaMetadataMap = resp.GetMediaItems()
        }
    }

	if authorsErr != nil { log.Printf("SearchThreadsHTTP: Error occurred during author profile fetching: %v", authorsErr) }
	if mediaErr != nil { log.Printf("SearchThreadsHTTP: Error occurred during media metadata fetching: %v", mediaErr) }

	// 4. Map to Frontend Structure, using the order from original search results
	frontendThreads := make([]FrontendThreadData, 0, len(searchServiceResp.GetThreadResults()))
	for _, searchResult := range searchServiceResp.GetThreadResults() {
		if fullThreadData, ok := fullThreadsDataMap[searchResult.GetId()]; ok && fullThreadData != nil {
			feThread := mapProtoThreadToFrontend(fullThreadData, authorsProfileMap, mediaMetadataMap)
			frontendThreads = append(frontendThreads, feThread)
		} else {
			log.Printf("SearchThreadsHTTP: Full details not found or fetch failed for searched thread ID %d, using minimal data.", searchResult.GetId())
             frontendThreads = append(frontendThreads, FrontendThreadData{
                 ID: searchResult.GetId(),
                 UserID: searchResult.GetUserId(),
                 Content: searchResult.GetContentSnippet(),
             })
		}
	}

	c.JSON(http.StatusOK, SearchThreadsAPIResponse{
		Threads: frontendThreads,
		HasMore: searchServiceResp.GetHasMore(),
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