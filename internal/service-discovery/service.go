package servicediscovery

import (
	"time"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	httpv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/upstreams/http/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	service_discovery "github.com/upper-institute/ops-control/gen/api/service-discovery"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func makeServiceCluster(input *service_discovery.ServiceCluster) (*clusterv3.Cluster, error) {

	cluster := &clusterv3.Cluster{
		Name:                 input.ServiceClusterName,
		ConnectTimeout:       durationpb.New(15 * time.Second),
		ClusterDiscoveryType: &clusterv3.Cluster_Type{Type: clusterv3.Cluster_EDS},
		LbPolicy:             clusterv3.Cluster_ROUND_ROBIN,
		DnsLookupFamily:      clusterv3.Cluster_V4_ONLY,
		EdsClusterConfig: &clusterv3.Cluster_EdsClusterConfig{
			ServiceName: input.ServiceClusterName,
			EdsConfig: &corev3.ConfigSource{
				ResourceApiVersion: resource.DefaultAPIVersion,
				ConfigSourceSpecifier: &corev3.ConfigSource_ApiConfigSource{
					ApiConfigSource: &corev3.ApiConfigSource{
						TransportApiVersion:       resource.DefaultAPIVersion,
						ApiType:                   corev3.ApiConfigSource_GRPC,
						SetNodeOnFirstMessageOnly: true,
						GrpcServices: []*corev3.GrpcService{{
							TargetSpecifier: &corev3.GrpcService_EnvoyGrpc_{
								EnvoyGrpc: &corev3.GrpcService_EnvoyGrpc{ClusterName: input.XdsClusterName},
							},
						}},
					},
				},
			},
		},
	}

	switch input.ServiceType {

	case service_discovery.ServiceType_SERVICE_TYPE_GRPC_SERVICE:
		cluster.Http2ProtocolOptions = &corev3.Http2ProtocolOptions{
			MaxConcurrentStreams: wrapperspb.UInt32(2147483647),
		}

	case service_discovery.ServiceType_SERVICE_TYPE_HTTP1_SERVER:
		protocolOptions, err := anypb.New(&httpv3.HttpProtocolOptions{
			UpstreamProtocolOptions: &httpv3.HttpProtocolOptions_ExplicitHttpConfig_{
				ExplicitHttpConfig: &httpv3.HttpProtocolOptions_ExplicitHttpConfig{
					ProtocolConfig: &httpv3.HttpProtocolOptions_ExplicitHttpConfig_HttpProtocolOptions{
						HttpProtocolOptions: &corev3.Http1ProtocolOptions{},
					},
				},
			},
		})

		if err != nil {
			return nil, err
		}

		cluster.TypedExtensionProtocolOptions = map[string]*anypb.Any{
			"envoy.extensions.upstreams.http.v3.HttpProtocolOptions": protocolOptions,
		}

	}

	return cluster, nil
}
