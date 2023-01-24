package parameter

import (
	"context"
	"net/url"
)

type PullRequest struct {
	*ParameterOptions
	Url    *url.URL
	Result chan *Parameter
}

type Reader interface {
	Pull(ctx context.Context, options *PullRequest) error
}

type Writer interface {
	Put(ctx context.Context, parameter *Parameter) error
}

type Store interface {
	Reader
	Writer
}

type Downloader interface {
	Download(ctx context.Context, parameter *Parameter) error
}

type Uploader interface {
	Upload(ctx context.Context, parameter *Parameter) error
}

type Storage interface {
	Downloader
	Uploader
}
