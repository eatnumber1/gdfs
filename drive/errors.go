package drive

type DriveErrorCode int

type driveError struct {
	msg string
	code DriveErrorCode
}

const (
	NOT_FOUND DriveErrorCode = iota
	UNKNOWN_MIME DriveErrorCode = iota
)

func NewDriveError(msg string, code DriveErrorCode) error {
	return &driveError{
		msg: msg,
		code: code,
	}
}

func (this *driveError) Error() string {
	return this.msg
}
