package dto

type CategoryRequest struct {
	Name string `json:"name" binding:"required,min=2"`
}

type CategoryResponse struct {
	Id   int64  `json:"id"`
	Name string `json:"name"`
}
