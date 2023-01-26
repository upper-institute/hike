package servicemesh

import (
	"crypto/sha256"
	"strconv"
	"time"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	sdapi "github.com/upper-institute/hike/proto/api/service-discovery"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"

	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/stream/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/cors/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/grpc_web/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/health_check/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/jwt_authn/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/upstreams/http/v3"

	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	http_connection_managerv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
)

type Resources struct {
	virtualHosts VirtualHostMap
	resourceMap  map[string][]types.Resource
	logger       *zap.SugaredLogger
}

func NewResources(logger *zap.SugaredLogger) *Resources {
	return &Resources{
		virtualHosts: make(VirtualHostMap),
		resourceMap: map[string][]types.Resource{
			resource.EndpointType:    {},
			resource.ClusterType:     {},
			resource.SecretType:      {},
			resource.RouteType:       {},
			resource.ListenerType:    {},
			resource.RuntimeType:     {},
			resource.VirtualHostType: {},
		},
		logger: logger,
	}
}

func (r *Resources) ApplyService(svc *sdapi.Service) {

	r.logger.Infow("Apply service resources", "service_name", svc.ServiceName)

	cluster := svc.EnvoyCluster

	if cluster != nil {

		cluster = &clusterv3.Cluster{
			Name:                 svc.ServiceName,
			ConnectTimeout:       durationpb.New(15 * time.Second),
			ClusterDiscoveryType: &clusterv3.Cluster_Type{Type: clusterv3.Cluster_EDS},
			LbPolicy:             clusterv3.Cluster_ROUND_ROBIN,
			DnsLookupFamily:      clusterv3.Cluster_V4_ONLY,
			EdsClusterConfig: &clusterv3.Cluster_EdsClusterConfig{
				ServiceName: svc.ServiceName,
				EdsConfig: &corev3.ConfigSource{
					ResourceApiVersion: resource.DefaultAPIVersion,
					ConfigSourceSpecifier: &corev3.ConfigSource_ApiConfigSource{
						ApiConfigSource: &corev3.ApiConfigSource{
							TransportApiVersion:       resource.DefaultAPIVersion,
							ApiType:                   corev3.ApiConfigSource_GRPC,
							SetNodeOnFirstMessageOnly: true,
							GrpcServices: []*corev3.GrpcService{{
								TargetSpecifier: &corev3.GrpcService_EnvoyGrpc_{
									EnvoyGrpc: &corev3.GrpcService_EnvoyGrpc{
										ClusterName: svc.XdsClusterName,
									},
								},
							}},
						},
					},
				},
			},
		}

		r.resourceMap[resource.ClusterType] = append(r.resourceMap[resource.ClusterType], cluster)

	}

	httpConn := svc.EnvoyHttpConnectionManager

	if httpConn != nil {

		if httpConn.RouteSpecifier == nil {

			httpConn.RouteSpecifier = &http_connection_managerv3.HttpConnectionManager_Rds{
				Rds: &http_connection_managerv3.Rds{
					ConfigSource: &corev3.ConfigSource{
						ResourceApiVersion: resource.DefaultAPIVersion,
						ConfigSourceSpecifier: &corev3.ConfigSource_ApiConfigSource{
							ApiConfigSource: &corev3.ApiConfigSource{
								ApiType:                   corev3.ApiConfigSource_GRPC,
								TransportApiVersion:       resource.DefaultAPIVersion,
								SetNodeOnFirstMessageOnly: false,
								GrpcServices: []*corev3.GrpcService{{
									TargetSpecifier: &corev3.GrpcService_EnvoyGrpc_{
										EnvoyGrpc: &corev3.GrpcService_EnvoyGrpc{
											ClusterName: svc.XdsClusterName,
										},
									},
								}},
							},
						},
					},
					RouteConfigName: svc.ServiceName,
				},
			}

		}

		httpConnManagerAny, err := anypb.New(httpConn)
		if err != nil {
			panic(err)
		}

		r.resourceMap[resource.ListenerType] = append(
			r.resourceMap[resource.ListenerType],
			&listenerv3.Listener{
				Name: svc.ServiceName,
				Address: &corev3.Address{
					Address: &corev3.Address_SocketAddress{
						SocketAddress: &corev3.SocketAddress{
							Protocol: corev3.SocketAddress_TCP,
							Address:  "0.0.0.0",
							PortSpecifier: &corev3.SocketAddress_PortValue{
								PortValue: svc.ListenPort,
							},
						},
					},
				},
				FilterChains: []*listenerv3.FilterChain{
					{
						Filters: []*listenerv3.Filter{
							{
								Name: wellknown.HTTPConnectionManager,
								ConfigType: &listenerv3.Filter_TypedConfig{
									TypedConfig: httpConnManagerAny,
								},
							},
						},
					},
				},
			},
		)

		r.resourceMap[resource.RouteType] = append(r.resourceMap[resource.RouteType], &routev3.RouteConfiguration{
			Name:                     svc.ServiceName,
			IgnorePortInHostMatching: true,
			Vhds: &routev3.Vhds{
				ConfigSource: &corev3.ConfigSource{
					ResourceApiVersion: resource.DefaultAPIVersion,
					ConfigSourceSpecifier: &corev3.ConfigSource_ApiConfigSource{
						ApiConfigSource: &corev3.ApiConfigSource{
							ApiType:                   corev3.ApiConfigSource_DELTA_GRPC,
							TransportApiVersion:       resource.DefaultAPIVersion,
							SetNodeOnFirstMessageOnly: true,
							GrpcServices: []*corev3.GrpcService{{
								TargetSpecifier: &corev3.GrpcService_EnvoyGrpc_{
									EnvoyGrpc: &corev3.GrpcService_EnvoyGrpc{
										ClusterName: svc.XdsClusterName,
									},
								},
							}},
						},
					},
				},
			},
		})

	}

	for _, endpoint := range svc.EnvoyEndpoints {
		endpoint.ClusterName = svc.ServiceName
		r.resourceMap[resource.EndpointType] = append(r.resourceMap[resource.EndpointType], endpoint)
	}

	for _, route := range svc.EnvoyRoutes {
		r.virtualHosts.MergeRoute(route)
	}

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

	r.resourceMap[resource.VirtualHostType] = r.virtualHosts.ToResourceSlice()

	snapshot, err := cache.NewSnapshot(strconv.FormatInt(version, 10), r.resourceMap)
	if err != nil {
		return nil, err
	}

	if err := snapshot.Consistent(); err != nil {
		return nil, err
	}

	return snapshot, nil

}
