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
	s3Client     *s3.Client
	s3Downloader *manager.Downloader

	logger *zap.SugaredLogger
}

func NewS3ParameterFileDownloader(
	s3Client *s3.Client,
	logger *zap.SugaredLogger,
) parameter.ParameterFileDownloader {
	return &s3ParameterFileDownloader{
		s3Client:     s3Client,
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

	headObjectInput := &s3.HeadObjectInput{
		Bucket: aws.String(u.Host),
		Key:    aws.String(key),
	}

	headObjectOutput, err := s.s3Client.HeadObject(ctx, headObjectInput)
	if err != nil {
		return err
	}

	buf := make([]byte, int(headObjectOutput.ContentLength))

	w := manager.NewWriteAtBuffer(buf)

	input := &s3.GetObjectInput{
		Bucket: headObjectInput.Bucket,
		Key:    headObjectInput.Key,
	}

	s.logger.Debugw("Starting download of file", "source", source, "host", u.Host, "key", key, "file_size", headObjectOutput.ContentLength)

	_, err = s.s3Downloader.Download(ctx, w, input)
	if err != nil {
		return err
	}

	writtenBytes, err := writer.Write(buf)

	s.logger.Debugw("Downloaded file", "source", source, "host", u.Host, "key", key, "downloaded_size", writtenBytes)

	return err
}
