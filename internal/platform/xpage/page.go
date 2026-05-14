package xpage

import (
	"net/http"
	"strconv"
)

const (
	DefaultPage     = 1
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// Params 表示分页请求参数。
type Params struct {
	Page     int `json:"page" form:"page"`
	PageSize int `json:"page_size" form:"page_size"`
}

// Result 表示带分页信息的结果集。
type Result[T any] struct {
	Items    []T   `json:"items"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

// Parse 用于解析分页参数。
func Parse(page string, pageSize string) Params {
	params := Params{
		Page:     parseInt(page),
		PageSize: parseInt(pageSize),
	}

	return Normalize(params)
}

// ParseRequest 用于从 HTTP 请求中解析分页参数。
func ParseRequest(r *http.Request) Params {
	if r == nil {
		return Normalize(Params{})
	}

	return Parse(r.URL.Query().Get("page"), r.URL.Query().Get("page_size"))
}

// Normalize 用于归一化分页参数。
func Normalize(params Params) Params {
	if params.Page <= 0 {
		params.Page = DefaultPage
	}

	switch {
	case params.PageSize <= 0:
		params.PageSize = DefaultPageSize
	case params.PageSize > MaxPageSize:
		params.PageSize = MaxPageSize
	}

	return params
}

// Offset 返回当前分页参数对应的偏移量。
func (p Params) Offset() int {
	normalized := Normalize(p)
	return (normalized.Page - 1) * normalized.PageSize
}

// Limit 返回当前分页参数对应的查询条数。
func (p Params) Limit() int {
	return Normalize(p).PageSize
}

// NewResult 用于构造带分页信息的结果集。
func NewResult[T any](items []T, total int64, params Params) Result[T] {
	normalized := Normalize(params)
	return Result[T]{
		Items:    items,
		Total:    total,
		Page:     normalized.Page,
		PageSize: normalized.PageSize,
	}
}

// parseInt 用于安全解析整数字符串。
func parseInt(value string) int {
	if value == "" {
		return 0
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}

	return parsed
}
