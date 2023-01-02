package aws

import (
	"context"
	"io"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	parameter "github.com/upper-institute/ops-control/internal/parameter"
	"go.uber.org/zap"
)

type s3ParameterFileDownloader struct {
	s3Downloader *manager.Downloader

	logger *zap.SugaredLogger
}

func NewS3ParameterFileDownloader(
	s3Client *s3.Client,
	logger *zap.SugaredLogger,
) parameter.ParameterFileDownloader {
	return &s3ParameterFileDownloader{
		s3Downloader: manager.NewDownloader(s3Client),
		logger:       logger,
	}
}

func (s *s3ParameterFileDownloader) Download(ctx context.Context, source string, writer io.Writer) error {

	s.logger.Infow("Download parameter file from S3", "source", source)

	u, err := url.Parse(source)
	if err != nil {
		return err
	}

	key := u.Path[1:]

	s.logger.Debugw("S3 GetObjectInput", "source", source, "host", u.Host, "key", key)

	input := &s3.GetObjectInput{
		Bucket: aws.String(u.Host),
		Key:    aws.String(key),
	}

	buf := []byte{}

	w := manager.NewWriteAtBuffer(buf)

	_, err = s.s3Downloader.Download(ctx, w, input)
	if err != nil {
		return err
	}

	_, err = writer.Write(buf)

	return err
}
