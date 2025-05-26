package models

// PaginationParams represents the pagination parameters
type PaginationParams struct {
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	Search   string `json:"search"`
}

// PaginationResponse represents the pagination response
type PaginationResponse struct {
	TotalItems   int `json:"total_items"`
	TotalPages   int `json:"total_pages"`
	CurrentPage  int `json:"current_page"`
	ItemsPerPage int `json:"items_per_page"`
}

// NewsResponse represents the response for news list with pagination
type NewsResponse struct {
	Items      []News             `json:"items"`
	Pagination PaginationResponse `json:"pagination"`
}
