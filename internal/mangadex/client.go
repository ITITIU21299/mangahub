package mangadex

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"mangahub/pkg/models"
)

const MANGADEX_BASE = "https://api.mangadex.org"

// Client handles MangaDex API requests
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new MangaDex client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: MANGADEX_BASE,
	}
}

// MangaDexManga represents a manga from MangaDex API
type MangaDexManga struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		Title       map[string]string `json:"title"`
		Description map[string]string `json:"description"`
		Status      string            `json:"status"`
		LastChapter string            `json:"lastChapter"`
		LastVolume  string            `json:"lastVolume"`
		Tags        []struct {
			Attributes struct {
				Name map[string]string `json:"name"`
			} `json:"attributes"`
		} `json:"tags"`
	} `json:"attributes"`
	Relationships []struct {
		Type       string `json:"type"`
		Attributes struct {
			FileName string `json:"fileName"`
			Name     string `json:"name"`
		} `json:"attributes,omitempty"`
	} `json:"relationships"`
}

// MangaDexResponse represents the MangaDex API response
type MangaDexResponse struct {
	Result   string          `json:"result"`
	Response string          `json:"response"`
	Data     []MangaDexManga `json:"data"`
	Total    int             `json:"total"`
	Limit    int             `json:"limit"`
	Offset   int             `json:"offset"`
}

// SearchManga searches MangaDex for manga
func (c *Client) SearchManga(query, genre, status string, limit, offset int) (*MangaDexResponse, error) {
	params := url.Values{}
	params.Add("limit", fmt.Sprintf("%d", limit))
	params.Add("offset", fmt.Sprintf("%d", offset))
	params.Add("includes[]", "cover_art")
	params.Add("includes[]", "author")
	params.Add("includes[]", "artist")
	params.Add("contentRating[]", "safe")
	params.Add("contentRating[]", "suggestive")
	params.Add("contentRating[]", "erotica")

	if query != "" {
		params.Add("title", query)
	}

	if status != "" {
		mappedStatus := strings.ToLower(status)
		if mappedStatus == "ongoing" || mappedStatus == "completed" || mappedStatus == "hiatus" {
			params.Add("status[]", mappedStatus)
		}
	}

	reqURL := fmt.Sprintf("%s/manga?%s", c.baseURL, params.Encode())

	log.Printf("[MangaDex] Fetching: %s", reqURL)

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", "MangaHub/1.0 (Net Centric Project)")
	req.Header.Set("Accept", "application/json")

	// Rate limiting: wait 200ms between requests
	time.Sleep(200 * time.Millisecond)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("MangaDex API error: %d - %s", resp.StatusCode, string(body))
	}

	var result MangaDexResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	log.Printf("[MangaDex] Response: %d results, total: %d", len(result.Data), result.Total)
	return &result, nil
}

// GetMangaByID fetches a single manga from MangaDex by ID
func (c *Client) GetMangaByID(mangaID string) (*MangaDexManga, error) {
	params := url.Values{}
	params.Add("includes[]", "cover_art")
	params.Add("includes[]", "author")
	params.Add("includes[]", "artist")

	reqURL := fmt.Sprintf("%s/manga/%s?%s", c.baseURL, mangaID, params.Encode())

	log.Printf("[MangaDex] Fetching manga by ID: %s", reqURL)

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", "MangaHub/1.0 (Net Centric Project)")
	req.Header.Set("Accept", "application/json")

	// Rate limiting: wait 200ms between requests
	time.Sleep(200 * time.Millisecond)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("manga not found")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("MangaDex API error: %d - %s", resp.StatusCode, string(body))
	}

	var result struct {
		Result string        `json:"result"`
		Data   MangaDexManga `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if result.Result != "ok" {
		return nil, fmt.Errorf("MangaDex API error: result is not ok")
	}

	log.Printf("[MangaDex] Successfully fetched manga: %s", result.Data.ID)
	return &result.Data, nil
}

// AggregateResponse represents the MangaDex aggregate endpoint response
type AggregateResponse struct {
	Result  string `json:"result"`
	Volumes map[string]struct {
		Volume   string `json:"volume"`
		Count    int    `json:"count"`
		Chapters map[string]struct {
			Chapter string `json:"chapter"`
			ID      string `json:"id"`
			Count   int    `json:"count"`
		} `json:"chapters"`
	} `json:"volumes"`
}

// GetChapterCount fetches the highest chapter number from the aggregate endpoint
// This avoids language filter issues and gives accurate chapter counts
func (c *Client) GetChapterCount(mangaID string) (int, error) {
	reqURL := fmt.Sprintf("%s/manga/%s/aggregate", c.baseURL, mangaID)

	log.Printf("[MangaDex] Fetching chapter count from aggregate: %s", reqURL)

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return 0, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", "MangaHub/1.0 (Net Centric Project)")
	req.Header.Set("Accept", "application/json")

	// Rate limiting: wait 200ms between requests
	time.Sleep(200 * time.Millisecond)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("fetch failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return 0, fmt.Errorf("manga not found")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("MangaDex API error: %d - %s", resp.StatusCode, string(body))
	}

	var result AggregateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("decode response: %w", err)
	}

	if result.Result != "ok" {
		return 0, fmt.Errorf("MangaDex API error: result is not ok")
	}

	// Find the highest chapter number across all volumes
	maxChapter := 0.0
	for _, volume := range result.Volumes {
		for chapterKey := range volume.Chapters {
			var ch float64
			if _, err := fmt.Sscanf(chapterKey, "%f", &ch); err == nil {
				if ch > maxChapter {
					maxChapter = ch
				}
			}
		}
	}

	if maxChapter > 0 {
		log.Printf("[MangaDex] Found highest chapter from aggregate: %.0f", maxChapter)
		return int(maxChapter), nil
	}

	log.Printf("[MangaDex] No chapters found in aggregate response")
	return 0, nil
}

func TransformMangaDexToManga(md *MangaDexManga, client *Client, useAggregate bool) *models.Manga {
	if md == nil || md.Attributes.Title == nil {
		return nil
	}

	// Get title (prefer English, fallback to first available)
	title := ""
	if en, ok := md.Attributes.Title["en"]; ok && en != "" {
		title = en
	} else {
		for _, t := range md.Attributes.Title {
			if t != "" {
				title = t
				break
			}
		}
	}
	if title == "" {
		return nil
	}

	// Get description
	description := ""
	if desc, ok := md.Attributes.Description["en"]; ok {
		description = desc
	} else {
		for _, d := range md.Attributes.Description {
			if d != "" {
				description = d
				break
			}
		}
	}

	// Extract genres from tags
	genres := []string{}
	for _, tag := range md.Attributes.Tags {
		if tag.Attributes.Name != nil {
			if name, ok := tag.Attributes.Name["en"]; ok && name != "" {
				genres = append(genres, name)
			}
		}
	}

	// Extract author and cover URL from relationships
	author := "Unknown"
	coverURL := ""
	for _, rel := range md.Relationships {
		if rel.Type == "author" || rel.Type == "artist" {
			if rel.Attributes.Name != "" {
				if author == "Unknown" {
					author = rel.Attributes.Name
				} else {
					author += ", " + rel.Attributes.Name
				}
			}
		}
		if rel.Type == "cover_art" && rel.Attributes.FileName != "" {
			coverURL = fmt.Sprintf("https://uploads.mangadex.org/covers/%s/%s", md.ID, rel.Attributes.FileName)
		}
	}

	// Get total chapters count
	totalChapters := 0

	if useAggregate && client != nil {
		if count, err := client.GetChapterCount(md.ID); err == nil && count > 0 {
			totalChapters = count
			log.Printf("[MangaDex] Using chapter count from aggregate: %d", totalChapters)
		} else {
			log.Printf("[MangaDex] Failed to get chapter count from aggregate, falling back to lastChapter: %v", err)
		}
	}

	// Use lastChapter if aggregate wasn't used or didn't work
	if totalChapters == 0 && md.Attributes.LastChapter != "" {
		var ch float64
		if _, err := fmt.Sscanf(md.Attributes.LastChapter, "%f", &ch); err == nil {
			totalChapters = int(ch)
		}
	}

	// Map status
	status := md.Attributes.Status
	if status == "" {
		status = "Unknown"
	}

	return &models.Manga{
		ID:            "mangadex-" + md.ID, // Prefix to distinguish from local DB manga
		Title:         title,
		Author:        author,
		Genres:        genres,
		Status:        status,
		TotalChapters: totalChapters,
		Description:   description,
		CoverURL:      coverURL,
	}
}
