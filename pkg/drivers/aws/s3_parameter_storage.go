package awsdriver

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go/aws/awserr"
	parameter "github.com/upper-institute/ops-control/pkg/parameter"
	"go.uber.org/zap"
)

type s3ParameterFile struct {
	s3Client     *s3.Client
	s3Downloader *manager.Downloader
	s3Uploader   *manager.Uploader

	logger *zap.SugaredLogger
}

func NewS3ParameterStorage(
	s3Client *s3.Client,
	logger *zap.SugaredLogger,
) parameter.Storage {
	return &s3ParameterFile{
		s3Client:     s3Client,
		s3Downloader: manager.NewDownloader(s3Client),
		s3Uploader:   manager.NewUploader(s3Client),
		logger:       logger,
	}
}

func (s *s3ParameterFile) Download(ctx context.Context, param *parameter.Parameter) error {

	log := s.logger.With(
		"parameter_key", param.GetKey(),
		"bucket", param.GetHost(),
		"object_key", param.GetPath(),
	)

	log.Infow("Download parameter file from S3")

	headObjectInput := &s3.HeadObjectInput{
		Bucket: aws.String(param.GetHost()),
		Key:    aws.String(param.GetPath()),
	}

	headObjectOutput, err := s.s3Client.HeadObject(ctx, headObjectInput)
	if err != nil {

		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "NotFound":
				return parameter.FileNotFoundErr
			}

		}

		return err
	}

	buf := make([]byte, int(headObjectOutput.ContentLength))

	w := manager.NewWriteAtBuffer(buf)

	input := &s3.GetObjectInput{
		Bucket: headObjectInput.Bucket,
		Key:    headObjectInput.Key,
	}

	log.Debugw("Starting download of file", "file_size", headObjectOutput.ContentLength)

	_, err = s.s3Downloader.Download(ctx, w, input)
	if err != nil {
		return err
	}

	writtenBytes, err := param.GetFile().Write(buf)

	log.Debugw("Downloaded file", "downloaded_size", writtenBytes)

	return err
}

func (s *s3ParameterFile) Upload(ctx context.Context, param *parameter.Parameter) error {

	log := s.logger.With(
		"parameter_key", param.GetKey(),
		"bucket", param.GetHost(),
		"object_key", param.GetPath(),
	)

	log.Infow("Upload parameter file to S3")

	input := &s3.PutObjectInput{
		Bucket: aws.String(param.GetHost()),
		Key:    aws.String(param.GetPath()),
		Body:   param.GetFile(),
	}

	log.Debugw("Starting upload of file")

	_, err := s.s3Uploader.Upload(ctx, input)
	if err != nil {
		return err
	}

	log.Debugw("Uploaded file")

	return err
}
