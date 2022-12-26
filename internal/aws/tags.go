package aws

import (
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
)

type ApplicationTagType string

type PlatformTagType string

type ConsumersTagType string

const (
	ApplicationTag                                = "application"
	ApplicationTag_GrpcService ApplicationTagType = "grpc-service"
	ApplicationTag_Http1Server ApplicationTagType = "http1-server"

	PlatformTag                      = "platform"
	PlataformTag_Ecs PlatformTagType = "ecs"

	ConsumersTag                          = "consumers"
	ConsumersTag_Http    ConsumersTagType = "http"
	ConsumersTag_GrpcWeb ConsumersTagType = "grpc-web"

	ConfigurationPathTag = "configuration-path"
)

type TagArrayValue map[string]bool

func (t TagArrayValue) HasValue(value string) bool {

	_, ok := t[value]

	return ok

}

type WellKnownTags struct {
	Application       TagArrayValue
	Platform          TagArrayValue
	Consumers         TagArrayValue
	ConfigurationPath string
}

func splitTagArrayValue(valueStr string) TagArrayValue {

	values := strings.Split(valueStr, ",")
	arrayValue := make(TagArrayValue)

	for _, value := range values {
		arrayValue[value] = true
	}

	return arrayValue

}

func NewTagsFromTagList(tags []types.Tag) *WellKnownTags {

	w := &WellKnownTags{
		ConfigurationPath: "",
	}

	for _, tag := range tags {

		switch aws.ToString(tag.Key) {

		case ApplicationTag:
			w.Application = splitTagArrayValue(aws.ToString(tag.Value))

		case PlatformTag:
			w.Platform = splitTagArrayValue(aws.ToString(tag.Value))

		case ConsumersTag:
			w.Consumers = splitTagArrayValue(aws.ToString(tag.Value))

		case ConfigurationPathTag:
			w.ConfigurationPath = aws.ToString(tag.Value)

		}
	}

	return w

}

func (w *WellKnownTags) IsApplication(value ApplicationTagType) bool {
	return w.Application.HasValue(string(value))
}

func (w *WellKnownTags) IsPlatform(value PlatformTagType) bool {
	return w.Platform.HasValue(string(value))
}

func (w *WellKnownTags) HasConsumer(value ConsumersTagType) bool {
	return w.Consumers.HasValue(string(value))
}
