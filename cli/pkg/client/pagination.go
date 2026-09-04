package client

// PaginationParams is the request-side pagination control.
type PaginationParams struct {
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
}

// Pagination is the response-side pagination block. Traceway returns this
// on every paginated list endpoint.
type Pagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
	Total      int64 `json:"total"`
	TotalPages int64 `json:"totalPages"`
}
