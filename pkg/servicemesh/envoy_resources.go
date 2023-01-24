package servicemesh

import (
	"crypto/sha256"
	"strconv"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	service_discovery "github.com/upper-institute/ops-control/gen/api/service-discovery"
	"google.golang.org/protobuf/encoding/protojson"
)

type Resources struct {
	resourceMap map[string][]types.Resource
}

func NewResources() *Resources {
	return &Resources{
		resourceMap: map[string][]types.Resource{
			resource.EndpointType: {},
			resource.ClusterType:  {},
			resource.SecretType:   {},
			resource.RouteType:    {},
			resource.ListenerType: {},
			resource.RuntimeType:  {},
		},
	}
}

func (r *Resources) ApplyService(svc *service_discovery.Service) {

}

func (r *Resources) Hash() []byte {

	hash := sha256.New()

	for _, resources := range r.resourceMap {

		for _, resource := range resources {

			msg, err := protojson.Marshal(resource)
			if err != nil {
				panic(err)
			}

			hash.Write(msg)

		}

	}

	return hash.Sum(nil)

}

func (r *Resources) DoSnapshot(version int64) (*cache.Snapshot, error) {

	snapshot, err := cache.NewSnapshot(strconv.FormatInt(version, 10), r.resourceMap)
	if err != nil {
		return nil, err
	}

	if err := snapshot.Consistent(); err != nil {
		return nil, err
	}

	return snapshot, nil

}
