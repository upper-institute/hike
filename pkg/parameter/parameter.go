package parameter

import (
	"bytes"
	"context"
	"net/url"

	paramapi "github.com/upper-institute/hike/proto/api/parameter"
	"go.uber.org/zap"
)

const (
	VarScheme  = "var"
	FileScheme = "file"
)

type ParameterOptions struct {
	Downloader Downloader
	Uploader   Uploader
	Writer     Writer
	Logger     *zap.SugaredLogger
}

func (options *ParameterOptions) NewFromURLString(key string, urlStr string) (*Parameter, error) {

	uri, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	return options.NewFromURI(key, uri)

}

func (options *ParameterOptions) NewFromURI(key string, uri *url.URL) (*Parameter, error) {

	value := bytes.NewBuffer(nil)

	metadata := make(url.Values)

	return &Parameter{options, key, uri, value, metadata}, nil

}

type Parameter struct {
	options  *ParameterOptions
	key      string
	uri      *url.URL
	file     *bytes.Buffer
	Metadata url.Values
}

func (p *Parameter) GetKey() string {
	return p.key
}

func (p *Parameter) GetURLString() string {
	return p.uri.String()
}

func (p *Parameter) GetFile() *bytes.Buffer {
	return p.file
}

func (p *Parameter) GetFragment() string {
	return p.uri.Fragment
}

func (p *Parameter) GetHost() string {
	return p.uri.Host
}

func (p *Parameter) GetPath() string {
	return p.uri.Path
}

func (p *Parameter) GetQuery() url.Values {
	return p.uri.Query()
}

func (p *Parameter) SetQuery(values url.Values) {
	p.uri.RawQuery = values.Encode()
}

func (p *Parameter) GetType() paramapi.ParameterType {

	switch p.uri.Scheme {

	case VarScheme:
		return paramapi.ParameterType_PT_VAR

	case FileScheme:
		return paramapi.ParameterType_PT_FILE

	}

	return paramapi.ParameterType_PT_UNKNOWN

}

func (p *Parameter) Load(ctx context.Context) error {

	if p.GetType() != paramapi.ParameterType_PT_FILE {
		return LoadOnlyFileTypeErr
	}

	p.file.Reset()

	err := p.options.Downloader.Download(ctx, p)

	return err

}

func (p *Parameter) Push(ctx context.Context) error {

	if p.GetType() == paramapi.ParameterType_PT_UNKNOWN {
		return UnknownSchemeErr
	}

	err := p.options.Writer.Put(ctx, p)
	if err != nil {
		return err
	}

	if p.GetType() == paramapi.ParameterType_PT_FILE {
		err = p.options.Uploader.Upload(ctx, p)
	}

	return err

}
