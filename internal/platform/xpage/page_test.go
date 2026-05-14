package xpage

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestParseAppliesDefaults 用于验证分页默认值会被正确应用。
func TestParseAppliesDefaults(t *testing.T) {
	params := Parse("", "")

	require.Equal(t, Params{
		Page:     DefaultPage,
		PageSize: DefaultPageSize,
	}, params)
}

// TestParseCapsPageSize 用于验证分页大小会被限制在最大值以内。
func TestParseCapsPageSize(t *testing.T) {
	params := Parse("2", "999")

	require.Equal(t, Params{
		Page:     2,
		PageSize: MaxPageSize,
	}, params)
}

// TestOffsetUsesNormalizedValues 用于验证偏移量会基于归一化后的参数计算。
func TestOffsetUsesNormalizedValues(t *testing.T) {
	require.Equal(t, 20, Params{Page: 2, PageSize: 20}.Offset())
	require.Equal(t, 0, Params{}.Offset())
}

// TestParseRequestReadsQuery 用于验证分页参数可以从请求查询串中读取。
func TestParseRequestReadsQuery(t *testing.T) {
	req := httptest.NewRequest("GET", "/users?page=3&page_size=5", nil)

	params := ParseRequest(req)

	require.Equal(t, Params{Page: 3, PageSize: 5}, params)
}
