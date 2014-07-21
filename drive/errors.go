package drive

type DriveErrorCode int

type DriveError interface {
	error
	Code() DriveErrorCode
}

type driveError struct {
	msg string
	code DriveErrorCode
}

const (
	NOT_FOUND DriveErrorCode = iota
	BANNED_MIME DriveErrorCode = iota
)

func NewDriveError(msg string, code DriveErrorCode) DriveError {
	return &driveError{
		msg: msg,
		code: code,
	}
}

func (this *driveError) Error() string {
	return this.msg
}

func (this *driveError) Code() DriveErrorCode {
	return this.code
}
