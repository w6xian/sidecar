package muxhttp

import (
	"github.com/google/uuid"
)

type IResult interface {
	ToBytes() ([]byte, error)
}

type IError interface {
	Error() string
}

type Error struct {
	Result
}

func NewErr(err error) *Error {
	er := &Error{}
	if err != nil {
		er.Text = err.Error()
	}
	er.Status = 500
	er.Track = uuid.NewString()
	return er
}

func NewError(text string) *Error {
	err := Error{}
	err.Status = 500
	err.Text = text
	err.Track = uuid.NewString()
	return &err
}

func NoAuthorization(txt string) *Error {
	err := Error{}
	err.Status = 401
	err.Text = "Invali Authorization:" + txt
	err.Track = uuid.NewString()
	return &err
}

func NewArgsError(txt string) *Error {
	err := Error{}
	err.Status = 701
	err.Text = "Invali argments:" + txt
	err.Track = uuid.NewString()
	return &err
}

func NewArgsErr(e error) *Error {
	err := Error{}
	err.Status = 701
	err.Text = "Invali argments:" + e.Error()
	err.Track = uuid.NewString()
	return &err
}
func NewArgsValidErr(e error) *Error {
	err := Error{}
	err.Status = 701
	err.Text = "argments validate error:" + e.Error()
	err.Track = uuid.NewString()
	return &err
}
