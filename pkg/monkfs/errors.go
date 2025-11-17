package monkfs

import (
	"syscall"

	"github.com/ianzepp/monk-api-fuse/pkg/monkapi"
)

// HTTPErrorToErrno maps HTTP status codes and error codes to FUSE errno values
func HTTPErrorToErrno(err error) syscall.Errno {
	if err == nil {
		return 0
	}

	apiErr, ok := err.(*monkapi.APIError)
	if !ok {
		return syscall.EIO
	}

	switch apiErr.StatusCode {
	case 401: // TOKEN_INVALID
		return syscall.EACCES
	case 403: // PERMISSION_DENIED
		return syscall.EPERM
	case 404: // RECORD_NOT_FOUND, SCHEMA_NOT_FOUND, FIELD_NOT_FOUND
		return syscall.ENOENT
	case 400:
		switch apiErr.ErrorCode {
		case "INVALID_PATH":
			return syscall.EINVAL
		case "NOT_A_FILE":
			return syscall.EISDIR
		case "WILDCARDS_NOT_ALLOWED":
			return syscall.EINVAL
		default:
			return syscall.EINVAL
		}
	case 409: // RECORD_EXISTS
		return syscall.EEXIST
	default:
		return syscall.EIO
	}
}
