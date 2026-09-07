package media

import (
	"context"
	"io"
	"time"
)

type Object struct {
	Key, ContentType string
	Size             int64
	Body             io.ReadCloser
}
type Storage interface {
	Put(context.Context, string, io.Reader, string) error
	Get(context.Context, string) (Object, error)
	Delete(context.Context, string) error
	Exists(context.Context, string) (bool, error)
	PresignGet(context.Context, string, time.Duration) (string, error)
}
