package monkapi

import "encoding/json"

// APIWrapper wraps all API responses with success and data fields
type APIWrapper struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
}

// ListOptions represents options for the File API list operation
type ListOptions struct {
	ShowHidden          bool   `json:"show_hidden,omitempty"`
	LongFormat          bool   `json:"long_format,omitempty"`
	Recursive           bool   `json:"recursive,omitempty"`
	MaxDepth            int    `json:"max_depth,omitempty"`
	SortBy              string `json:"sort_by,omitempty"`
	SortOrder           string `json:"sort_order,omitempty"`
	PatternOptimization bool   `json:"pattern_optimization,omitempty"`
}

// ListResponse represents the File API list response
type ListResponse struct {
	Success      bool              `json:"success"`
	Entries      []FileEntry       `json:"entries"`
	Total        int               `json:"total"`
	HasMore      bool              `json:"has_more"`
	FileMetadata FileMetadata      `json:"file_metadata"`
}

// FileEntry represents a single file/directory entry
type FileEntry struct {
	Name            string                 `json:"name"`
	FileType        string                 `json:"file_type"`
	FileSize        int64                  `json:"file_size"`
	FilePermissions string                 `json:"file_permissions"`
	FileModified    string                 `json:"file_modified"`
	Path            string                 `json:"path"`
	APIContext      map[string]interface{} `json:"api_context"`
}

// FileMetadata represents file metadata
type FileMetadata struct {
	Size         int64  `json:"size"`
	ModifiedTime string `json:"modified_time"` // Format: ISO 8601 (RFC3339)
	CreatedTime  string `json:"created_time"`  // Format: ISO 8601 (RFC3339)
	AccessTime   string `json:"access_time"`   // Format: ISO 8601 (RFC3339)
	Type         string `json:"type"`
	Permissions  string `json:"permissions"`
}

// StatResponse represents the File API stat response
type StatResponse struct {
	Success      bool         `json:"success"`
	FileMetadata FileMetadata `json:"file_metadata"`
	Type         string       `json:"type"`
}

// RetrieveOptions represents options for the File API retrieve operation
type RetrieveOptions struct {
	StartOffset int `json:"start_offset,omitempty"`
	MaxBytes    int `json:"max_bytes,omitempty"`
}

// RetrieveResponse represents the File API retrieve response
type RetrieveResponse struct {
	Success bool        `json:"success"`
	Content interface{} `json:"content"`
}

// StoreOptions represents options for the File API store operation
type StoreOptions struct {
	CreateMissing bool `json:"create_missing,omitempty"`
}

// StoreResponse represents the File API store response
type StoreResponse struct {
	Success      bool         `json:"success"`
	FileMetadata FileMetadata `json:"file_metadata"`
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Success   bool   `json:"success"`
	Error     string `json:"error"`
	ErrorCode string `json:"error_code"`
}
