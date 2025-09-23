package mangadexapi

import (
	"encoding/json"
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

type AuthForm struct {
	Username     string
	Password     string
	ClientID     string
	ClientSecret string
}

// Token represents an authentication token.
type Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	Expiry       time.Time `json:"-"` // don't unmarshal from JSON
}

// Client is the MangaDex API client.
type Client struct {
	httpClient  *http.Client
	baseURL     string
	userAgent   string
	rateLimiter *rate.Limiter

	token *Token
}

// Generic envelope (works for object or collection responses).
type Envelope struct {
	Result   string          `json:"result"`             // "ok" or "error"
	Response string          `json:"response,omitempty"` // "entity", "collection", etc.
	Data     json.RawMessage `json:"data,omitempty"`
	Errors   []APIError      `json:"errors,omitempty"`
	Limit    *int            `json:"limit,omitempty"`
	Offset   *int            `json:"offset,omitempty"`
	Total    *int            `json:"total,omitempty"`
	// Some endpoints include these for pagination; keep pointers so absence != 0
}

// APIError represents an error object in the API response.
type APIError struct {
	ID      string `json:"id"`
	Status  int    `json:"status"`
	Title   string `json:"title"`
	Detail  string `json:"detail"`
	Context string `json:"context,omitempty"`
}

// Advanced query params for manga search
type QueryParams struct {
	Limit int `url:"limit,omitempty"`
	// Offset                      int                       `url:"offset,omitempty"`
	ID             string   `url:"id,omitempty"`
	Title          string   `url:"title,omitempty"`
	AuthorOrArtist string   `url:"authorOrArtist,omitempty"`
	Authors        []string `url:"authors[],omitempty"`
	Artists        []string `url:"artists[],omitempty"`
	// Year                        string                    `url:"year,omitempty"`
	IncludedTags     []string `url:"includedTags[],omitempty"`
	IncludedTagsMode TagsMode `url:"includedTagsMode,omitempty"`
	ExcludedTags     []string `url:"excludedTags[],omitempty"`
	ExcludedTagsMode TagsMode `url:"excludedTagsMode,omitempty"`
	// Status           []Status `url:"status[],omitempty"`
	OriginalLanguage []string `url:"originalLanguage[],omitempty"`
	// ExcludedOriginalLanguage    []string                  `url:"excludedOriginalLanguage[],omitempty"`
	// AvailableTranslatedLanguage []string                  `url:"availableTranslatedLanguage[],omitempty"`
	// PublicationDemographic      []PublicationDemographic  `url:"publicationDemographic[],omitempty"`
	IDs           []string        `url:"ids[],omitempty"`
	ContentRating []ContentRating `url:"contentRating[],omitempty"`
	// CreatedAtSince              string                    `url:"createdAtSince,omitempty"`
	// UpdatedAtSince              string                    `url:"updatedAtSince,omitempty"`
	Includes []ReferenceExpansionManga `url:"includes[],omitempty"`
	// HasAvailableChapters        HasAvailableChapters      `url:"hasAvailableChapters,omitempty"`
	// HasUnavailableChapters      HasUnavailableChapters    `url:"hasUnavailableChapters,omitempty"`
	// Group                       string                    `url:"group,omitempty"`
	Order OrderParams `url:"order,omitempty"`
}

// OrderParams represents ordering options for manga queries.
type OrderParams map[string]string // e.g. {"title": "asc", "latestUploadedChapter": "desc"}

// Manga represents a manga object from the MangaDex API.
type Manga struct {
	ID string `json:"id"`
	// Type       string          `json:"type"`
	Attributes MangaAttributes `json:"attributes"`
	// Relationships []Relationship  `json:"relationships"`
}

// MangaAttributes represents the attributes of a manga.
type MangaAttributes struct {
	Title     map[string]string   `json:"title"`
	AltTitles []map[string]string `json:"altTitles"`
	//	Description                    map[string]string      `json:"description"`
	//	IsLocked                       bool                   `json:"isLocked"`
	Links map[string]string `json:"links"`
	//	OriginalLanguage               string                 `json:"originalLanguage"`
	//	LastVolume                     string                 `json:"lastVolume"`
	//	LastChapter                    string                 `json:"lastChapter"`
	//	PublicationDemographic         PublicationDemographic `json:"publicationDemographic"`
	//	Status                         Status                 `json:"status"`
	//	Year                           int                    `json:"year"`
	//	ContentRating                  ContentRating          `json:"contentRating"`
	//	ChapterNumbersResetOnNewVolume bool                   `json:"chapterNumbersResetOnNewVolume"`
	//	AvailableTranslatedLanguages   []string               `json:"availableTranslatedLanguages"`
	//	LatestUploadedChapter          string                 `json:"latestUploadedChapter"`
	// Tags []Tag `json:"tags"`
	// State                          string                 `json:"state"`
	// Version                        int                    `json:"version"`
	// CreatedAt                      string                 `json:"createdAt"`
	// UpdatedAt                      string                 `json:"updatedAt"`
}

// Tag represents a tag object from the MangaDex API.
type Tag struct {
	ID            string         `json:"id"`
	Type          string         `json:"type"`
	Attributes    TagAttributes  `json:"attributes"`
	Relationships []Relationship `json:"relationships"`
}

// TagAttributes represents the attributes of a tag object from the MangaDex API.
type TagAttributes struct {
	Name        map[string]string `json:"name"`
	Description map[string]string `json:"description"`
	Group       string            `json:"group"`
	Version     int               `json:"version"`
}

// Relationship represents a relationship object from the MangaDex API.
type Relationship struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Related    string                 `json:"related"`
	Attributes map[string]interface{} `json:"attributes"`
}

// Status for manga search
type Status string

const (
	StatusOngoing   Status = "ongoing"
	StatusCompleted Status = "completed"
	StatusHiatus    Status = "hiatus"
	StatusCancelled Status = "cancelled"
)

// ContentRating for manga search
type ContentRating string

const (
	ContentRatingSafe         ContentRating = "safe"
	ContentRatingSuggestive   ContentRating = "suggestive"
	ContentRatingErotica      ContentRating = "erotica"
	ContentRatingPornographic ContentRating = "pornographic"
)

// PublicationDemographic for manga search
type PublicationDemographic string

const (
	PubDemoShounen PublicationDemographic = "shounen"
	PubDemoShoujo  PublicationDemographic = "shoujo"
	PubDemoJosei   PublicationDemographic = "josei"
	PubDemoSeinen  PublicationDemographic = "seinen"
	PubDemoNone    PublicationDemographic = "none"
)

// IncludedTagsMode and ExcludedTagsMode
type TagsMode string

const (
	TagsModeAND TagsMode = "AND"
	TagsModeOR  TagsMode = "OR"
)

// HasAvailableChapters
type HasAvailableChapters string

const (
	HasAvailableChapters0     HasAvailableChapters = "0"
	HasAvailableChapters1     HasAvailableChapters = "1"
	HasAvailableChaptersTrue  HasAvailableChapters = "true"
	HasAvailableChaptersFalse HasAvailableChapters = "false"
)

// HasUnavailableChapters
type HasUnavailableChapters string

const (
	HasUnavailableChapters0 HasUnavailableChapters = "0"
	HasUnavailableChapters1 HasUnavailableChapters = "1"
)

// ReferenceExpansionManga
type ReferenceExpansionManga string

const (
	RefExpManga    ReferenceExpansionManga = "manga"
	RefExpCoverArt ReferenceExpansionManga = "cover_art"
	RefExpAuthor   ReferenceExpansionManga = "author"
	RefExpArtist   ReferenceExpansionManga = "artist"
	RefExpTag      ReferenceExpansionManga = "tag"
	RefExpCreator  ReferenceExpansionManga = "creator"
)

// ReadingStatus for user manga reading status
type ReadingStatus string

const (
	ReadingStatusReading    ReadingStatus = "reading"
	ReadingStatusOnHold     ReadingStatus = "on_hold"
	ReadingStatusPlanToRead ReadingStatus = "plan_to_read"
	ReadingStatusDropped    ReadingStatus = "dropped"
	ReadingStatusReReading  ReadingStatus = "re_reading"
	ReadingStatusCompleted  ReadingStatus = "completed"
)
