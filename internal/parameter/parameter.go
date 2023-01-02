package parameter

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"text/template"

	"github.com/upper-institute/ops-control/gen/api/parameter"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const ParameterValueSeparator = "|"

type File struct {
	Type        parameter.ParameterType
	Source      string
	Destination string
}

type ParameterSet struct {
	downloader ParameterFileDownloader
	logger     *zap.SugaredLogger

	Envs  map[string]string
	Files map[string]*File
}

func NewParameterSet(downloader ParameterFileDownloader, logger *zap.SugaredLogger) *ParameterSet {
	return &ParameterSet{
		downloader: downloader,
		logger:     logger,

		Envs:  map[string]string{},
		Files: map[string]*File{},
	}
}

func LoadParametersFromProcessEnv(paramSet *ParameterSet) error {

	envs := os.Environ()

	for _, env := range envs {

		sep := strings.IndexRune(env, '=')

		key := env[:sep]
		value := env[sep+1:]

		paramSet.logger.Debugw(
			"Load parameter from process env",
			"key", key,
		)

		paramSet.Envs[key] = value

	}

	return nil

}

func (ps *ParameterSet) Add(key string, value string) error {

	sep := strings.Index(value, ParameterValueSeparator)

	if sep == -1 {
		ps.logger.Warnw(SeparatorNotFoundErr.Error(), "key", key, "separator", ParameterValueSeparator)
		return SeparatorNotFoundErr
	}

	paramTypeStr := value[:sep]
	paramTypeVal, ok := parameter.ParameterType_value[paramTypeStr]
	if !ok {
		ps.logger.Warnw(InvalidParameterTypeErr.Error(), "key", key, "type_string", paramTypeStr)
		return InvalidParameterTypeErr
	}

	ps.logger.Debugw(
		"Add parameter",
		"key", key,
		"type_string", paramTypeStr,
	)

	switch paramType := parameter.ParameterType(paramTypeVal); paramType {

	case parameter.ParameterType_PARAMETER_TYPE_TEMPLATE_FILE, parameter.ParameterType_PARAMETER_TYPE_FILE:

		source := value[sep+1:]

		sepSource := strings.Index(source, ParameterValueSeparator)

		destination := ""

		if sepSource > -1 {
			destination = source[sepSource+1:]
		}

		source = source[:sepSource]

		ps.Files[key] = &File{
			Type:        paramType,
			Source:      source,
			Destination: destination,
		}

	case parameter.ParameterType_PARAMETER_TYPE_ENV_VAR:

		ps.Envs[key] = value[sep+1:]

	}

	return nil

}

func (ps *ParameterSet) HasEnv(key string) bool {

	_, ok := ps.Envs[key]
	return ok

}

func (ps *ParameterSet) GetEnv(key string) string {

	if value, ok := ps.Envs[key]; ok {
		return value
	}
	return ""

}

func (ps *ParameterSet) GetAllEnvs() map[string]string {

	envMap := make(map[string]string)

	for key, value := range ps.Envs {
		envMap[string(key)] = value
	}

	return envMap

}

func (ps *ParameterSet) HasFile(key string) bool {

	_, ok := ps.Files[key]
	return ok

}

func (ps *ParameterSet) GetFile(ctx context.Context, key string, buf io.Writer) error {

	paramFile, ok := ps.Files[key]

	if !ok {
		ps.logger.Warnw(FileNotFoundErr.Error(), "key", key)
		return FileNotFoundErr
	}

	switch paramFile.Type {

	case parameter.ParameterType_PARAMETER_TYPE_TEMPLATE_FILE:

		ps.logger.Debugw("Get parameter template file", "key", key)

		b := bytes.NewBuffer(nil)

		err := ps.downloader.Download(ctx, paramFile.Source, b)
		if err != nil {
			ps.logger.Warnw(err.Error(), "key", key)
			return err
		}

		ps.logger.Debugw("Downloaded parameter template file", "key", key, "buffer_size", b.Len())

		t, err := template.New(string(key)).Parse(b.String())
		if err != nil {
			ps.logger.Warnw(err.Error(), "key", key)
			return err
		}

		err = t.Execute(buf, ps)
		if err != nil {
			ps.logger.Warnw(err.Error(), "key", key)
			return err
		}

	case parameter.ParameterType_PARAMETER_TYPE_FILE:

		ps.logger.Debugw("Get parameter file", "key", key)

		err := ps.downloader.Download(ctx, paramFile.Source, buf)
		if err != nil {
			ps.logger.Warnw(err.Error(), "key", key)
			return err
		}

		ps.logger.Debugw("Downloaded parameter file", "key", key)

	}

	return nil

}

func (ps *ParameterSet) ParseProtoJson(ctx context.Context, key string, m protoreflect.ProtoMessage) error {

	buf := bytes.NewBuffer(nil)

	ps.logger.Debugw(
		"Getting proto JSON from parameter file",
		"key", key,
	)

	err := ps.GetFile(ctx, key, buf)
	if err != nil {
		ps.logger.Warnw(err.Error(), "key", key)
		return err
	}

	ps.logger.Debugw(
		"Proto JSON downloaded from parameter file",
		"key", key,
		"buffer_size", buf.Len(),
	)

	return protojson.Unmarshal(buf.Bytes(), m)

}

func (ps *ParameterSet) SaveFile(ctx context.Context, key string) error {

	paramFile, ok := ps.Files[key]

	if !ok {
		ps.logger.Warnw(FileNotFoundErr.Error(), "key", key)
		return FileNotFoundErr
	}

	f, err := os.Create(paramFile.Destination)
	if err != nil {
		ps.logger.Warnw(err.Error(), "key", key)
		return err
	}
	defer f.Close()

	return ps.GetFile(ctx, key, f)

}

func (ps *ParameterSet) SaveAllFiles(ctx context.Context) error {

	for key := range ps.Files {

		if err := ps.SaveFile(ctx, key); err != nil {
			return err
		}

	}

	return nil

}
