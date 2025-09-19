package mangadexapi

import "github.com/google/uuid"

type AuthForm struct {
	GrantType    string
	Username     string
	Password     string
	ClientID     string
	ClientSecret string
}

// Token represents an authentication token.
type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
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

// OrderParams represents ordering options for manga queries.
type OrderParams map[string]string // e.g. {"title": "asc", "latestUploadedChapter": "desc"}

// Advanced query params for manga search
type QueryParams struct {
	Limit                       int                       `url:"limit,omitempty"`
	Offset                      int                       `url:"offset,omitempty"`
	ID                          uuid.UUID                 `url:"id,omitempty"`
	Title                       string                    `url:"title,omitempty"`
	AuthorOrArtist              uuid.UUID                 `url:"authorOrArtist,omitempty"`
	Authors                     []uuid.UUID               `url:"authors[],omitempty"`
	Artists                     []uuid.UUID               `url:"artists[],omitempty"`
	Year                        string                    `url:"year,omitempty"`
	IncludedTags                []uuid.UUID               `url:"includedTags[],omitempty"`
	IncludedTagsMode            TagsMode                  `url:"includedTagsMode,omitempty"`
	ExcludedTags                []uuid.UUID               `url:"excludedTags[],omitempty"`
	ExcludedTagsMode            TagsMode                  `url:"excludedTagsMode,omitempty"`
	Status                      []Status                  `url:"status[],omitempty"`
	OriginalLanguage            []string                  `url:"originalLanguage[],omitempty"`
	ExcludedOriginalLanguage    []string                  `url:"excludedOriginalLanguage[],omitempty"`
	AvailableTranslatedLanguage []string                  `url:"availableTranslatedLanguage[],omitempty"`
	PublicationDemographic      []PublicationDemographic  `url:"publicationDemographic[],omitempty"`
	IDs                         []uuid.UUID               `url:"ids[],omitempty"`
	ContentRating               []ContentRating           `url:"contentRating[],omitempty"`
	CreatedAtSince              string                    `url:"createdAtSince,omitempty"`
	UpdatedAtSince              string                    `url:"updatedAtSince,omitempty"`
	Includes                    []ReferenceExpansionManga `url:"includes[],omitempty"`
	HasAvailableChapters        HasAvailableChapters      `url:"hasAvailableChapters,omitempty"`
	HasUnavailableChapters      HasUnavailableChapters    `url:"hasUnavailableChapters,omitempty"`
	Group                       uuid.UUID                 `url:"group,omitempty"`
	Order                       OrderParams               `url:"order,omitempty"`
}

// Response represents a generic API response.
type EntityResponse struct {
	Result   string     `json:"result"`
	Response string     `json:"response"`
	Data     Manga      `json:"data"`
	Errors   []APIError `json:"errors,omitempty"`
}

type CollectionResponse struct {
	Result   string     `json:"result"`
	Response string     `json:"response"`
	Data     []Manga    `json:"data"`
	Limit    int        `json:"limit"`
	Offset   int        `json:"offset"`
	Total    int        `json:"total"`
	Errors   []APIError `json:"errors,omitempty"`
}

type ResultOnlyResponse struct {
	Result string `json:"result"`
	Status string `json:"status,omitempty"`
}

type ErrorResponse struct {
	Result string     `json:"result"`
	Errors []APIError `json:"errors"`
}

// APIError represents an error object in the API response.
type APIError struct {
	ID      string `json:"id"`
	Status  int    `json:"status"`
	Title   string `json:"title"`
	Detail  string `json:"detail"`
	Context string `json:"context,omitempty"`
}

// Manga represents a manga object from the MangaDex API.
type Manga struct {
	ID            string          `json:"id"`
	Type          string          `json:"type"`
	Attributes    MangaAttributes `json:"attributes"`
	Relationships []Relationship  `json:"relationships"`
}

// MangaAttributes represents the attributes of a manga.
type MangaAttributes struct {
	Title                          map[string]string      `json:"title"`
	AltTitles                      []map[string]string    `json:"altTitles"`
	Description                    map[string]string      `json:"description"`
	IsLocked                       bool                   `json:"isLocked"`
	Links                          map[string]string      `json:"links"`
	OriginalLanguage               string                 `json:"originalLanguage"`
	LastVolume                     string                 `json:"lastVolume"`
	LastChapter                    string                 `json:"lastChapter"`
	PublicationDemographic         PublicationDemographic `json:"publicationDemographic"`
	Status                         Status                 `json:"status"`
	Year                           int                    `json:"year"`
	ContentRating                  ContentRating          `json:"contentRating"`
	ChapterNumbersResetOnNewVolume bool                   `json:"chapterNumbersResetOnNewVolume"`
	AvailableTranslatedLanguages   []string               `json:"availableTranslatedLanguages"`
	LatestUploadedChapter          string                 `json:"latestUploadedChapter"`
	Tags                           []Tag                  `json:"tags"`
	State                          string                 `json:"state"`
	Version                        int                    `json:"version"`
	CreatedAt                      string                 `json:"createdAt"`
	UpdatedAt                      string                 `json:"updatedAt"`
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
