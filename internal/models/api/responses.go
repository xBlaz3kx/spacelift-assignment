package api

type ErrorResponse struct {
	InternalCode int    `json:"code"`
	Message      string `json:"message"`
}
