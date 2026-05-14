package xerror

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestConflictBuildsBusinessError 用于验证冲突错误的构造结果。
func TestConflictBuildsBusinessError(t *testing.T) {
	err := Conflict(1002, "user email already exists")

	require.Equal(t, Error{
		Code:    1002,
		Message: "user email already exists",
	}, err)
}

// TestAsExtractsAppError 用于验证可以从错误接口中提取业务错误。
func TestAsExtractsAppError(t *testing.T) {
	err := error(Conflict(1004, "user not found"))

	appErr, ok := As(err)

	require.True(t, ok)
	require.Equal(t, 1004, appErr.Code)
}

// TestAsRejectsUnknownError 用于验证普通错误不会被识别为业务错误。
func TestAsRejectsUnknownError(t *testing.T) {
	appErr, ok := As(errors.New("boom"))

	require.False(t, ok)
	require.Equal(t, Error{}, appErr)
}

// TestWrapKeepsCause 用于验证包装错误时会保留底层原因。
func TestWrapKeepsCause(t *testing.T) {
	cause := errors.New("db timeout")

	err := Wrap(cause, 1005, "query failed")

	require.ErrorIs(t, err, cause)
	require.Equal(t, 1005, CodeOf(err))
	require.Equal(t, "query failed", MessageOf(err))
}
