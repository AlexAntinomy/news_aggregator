package models

// Параметры пагинации
type PaginationParams struct {
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	Search   string `json:"search"`
}

// Ответ пагинации
type PaginationResponse struct {
	TotalItems   int `json:"total_items"`
	TotalPages   int `json:"total_pages"`
	CurrentPage  int `json:"current_page"`
	ItemsPerPage int `json:"items_per_page"`
}

// Ответ для списка новостей с пагинацией
type NewsResponse struct {
	Items      []News             `json:"items"`
	Pagination PaginationResponse `json:"pagination"`
}
