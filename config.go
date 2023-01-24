package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	accesslogv3 "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	streamv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/stream/v3"
	corsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/cors/v3"
	grpcwebv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/grpc_web/v3"
	health_checkv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/health_check/v3"
	routerv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	http_connection_managerv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	httpv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/upstreams/http/v3"
	matcherv3 "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	typev3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	sdapi "github.com/upper-institute/hike/proto/api/service-discovery"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func makeHealthCheckFilter() (*http_connection_managerv3.HttpFilter, error) {

	healthCheckPath := "/healthz"

	healthyPercentages := make(map[string]*typev3.Percent)

	healthyPercentages["test"] = &typev3.Percent{
		Value: 50,
	}

	healthCheck := &health_checkv3.HealthCheck{
		ClusterMinHealthyPercentages: healthyPercentages,
		PassThroughMode: &wrapperspb.BoolValue{
			Value: false,
		},
		Headers: []*routev3.HeaderMatcher{
			{
				Name: ":path",
				HeaderMatchSpecifier: &routev3.HeaderMatcher_ExactMatch{
					ExactMatch: healthCheckPath,
				},
			},
		},
	}

	healthCheckAny, err := anypb.New(healthCheck)
	if err != nil {
		return nil, err
	}

	return &http_connection_managerv3.HttpFilter{
		Name: wellknown.HealthCheck,
		ConfigType: &http_connection_managerv3.HttpFilter_TypedConfig{
			TypedConfig: healthCheckAny,
		},
	}, nil

}

func makeCorsFilter() (*http_connection_managerv3.HttpFilter, error) {

	cors := &corsv3.Cors{}

	corsAny, err := anypb.New(cors)
	if err != nil {
		return nil, err
	}

	return &http_connection_managerv3.HttpFilter{
		Name: wellknown.CORS,
		ConfigType: &http_connection_managerv3.HttpFilter_TypedConfig{
			TypedConfig: corsAny,
		},
	}, nil

}

func makeGrpcWebFilter() (*http_connection_managerv3.HttpFilter, error) {

	grpcWebFilter := &grpcwebv3.GrpcWeb{}

	grpcWebFilterAny, err := anypb.New(grpcWebFilter)
	if err != nil {
		return nil, err
	}

	return &http_connection_managerv3.HttpFilter{
		Name: wellknown.GRPCWeb,
		ConfigType: &http_connection_managerv3.HttpFilter_TypedConfig{
			TypedConfig: grpcWebFilterAny,
		},
	}, nil

}

func makeHttpFilters() ([]*http_connection_managerv3.HttpFilter, error) {

	httpFilters := make([]*http_connection_managerv3.HttpFilter, 0)

	healthCheckFilter, err := makeHealthCheckFilter()
	if err != nil {
		return nil, err
	}

	httpFilters = append(httpFilters, healthCheckFilter)

	corsFilter, err := makeCorsFilter()
	if err != nil {
		return nil, err
	}

	httpFilters = append(httpFilters, corsFilter)

	grpcWebFilter, err := makeGrpcWebFilter()
	if err != nil {
		return nil, err
	}

	httpFilters = append(httpFilters, grpcWebFilter)

	routerFilter, err := makeHttpRouterFilter()
	if err != nil {
		return nil, err
	}

	httpFilters = append(httpFilters, routerFilter)

	return httpFilters, nil

}

func makeHttpRouterFilter() (*http_connection_managerv3.HttpFilter, error) {

	router := &routerv3.Router{}

	routerAny, err := anypb.New(router)
	if err != nil {
		return nil, err
	}

	return &http_connection_managerv3.HttpFilter{
		Name: wellknown.Router,
		ConfigType: &http_connection_managerv3.HttpFilter_TypedConfig{
			TypedConfig: routerAny,
		},
	}, nil

}

func printFrontProxyCfg() {

	accessLogStdout := &streamv3.StdoutAccessLog{}

	accessLogStdoutAny, _ := anypb.New(accessLogStdout)

	httpFilters, _ := makeHttpFilters()

	frontProxyCfg := &sdapi.Service{
		XdsClusterName: "xds-cluster",
		ListenPort:     8081,
		DnsRecords: []*sdapi.DnsRecord{
			{
				Zone:       "dev.pomwm.com",
				RecordName: "us-east-1",
				CnameValue: "asdasd",
				Ttl:        durationpb.New(60 * time.Second),
			},
		},
		EnvoyHttpConnectionManager: &http_connection_managerv3.HttpConnectionManager{
			CodecType:  http_connection_managerv3.HttpConnectionManager_AUTO,
			StatPrefix: "frontproxy",
			AccessLog: []*accesslogv3.AccessLog{
				{
					Name: "stdout",
					ConfigType: &accesslogv3.AccessLog_TypedConfig{
						TypedConfig: accessLogStdoutAny,
					},
				},
			},
			HttpFilters: httpFilters,
		},
	}

	val, _ := protojson.Marshal(frontProxyCfg)
	println(string(val))
}

func printServiceCfg() {

	protocolOptions, _ := anypb.New(&httpv3.HttpProtocolOptions{
		UpstreamProtocolOptions: &httpv3.HttpProtocolOptions_ExplicitHttpConfig_{
			ExplicitHttpConfig: &httpv3.HttpProtocolOptions_ExplicitHttpConfig{
				ProtocolConfig: &httpv3.HttpProtocolOptions_ExplicitHttpConfig_HttpProtocolOptions{
					HttpProtocolOptions: &corev3.Http1ProtocolOptions{},
				},
			},
		},
	})

	routeToClusterAction := &routev3.Route_Route{
		Route: &routev3.RouteAction{
			ClusterSpecifier: &routev3.RouteAction_Cluster{
				Cluster: "service-cluster-name",
			},
			Timeout: durationpb.New(60 * time.Second),
			MaxStreamDuration: &routev3.RouteAction_MaxStreamDuration{
				GrpcTimeoutHeaderMax: durationpb.New(60 * time.Second),
			},
		},
	}

	serviceCfg := &sdapi.Service{
		XdsClusterName: "xds-cluster",
		ListenPort:     9091,
		EnvoyEndpoints: []*endpointv3.ClusterLoadAssignment{
			{},
		},
		EnvoyCluster: &clusterv3.Cluster{
			ConnectTimeout:       durationpb.New(15 * time.Second),
			ClusterDiscoveryType: &clusterv3.Cluster_Type{Type: clusterv3.Cluster_EDS},
			LbPolicy:             clusterv3.Cluster_ROUND_ROBIN,
			DnsLookupFamily:      clusterv3.Cluster_V4_ONLY,
			TypedExtensionProtocolOptions: map[string]*anypb.Any{
				"envoy.extensions.upstreams.http.v3.HttpProtocolOptions": protocolOptions,
			},
			Http2ProtocolOptions: &corev3.Http2ProtocolOptions{
				MaxConcurrentStreams: wrapperspb.UInt32(2147483647),
			},
		},

		EnvoyRoutes: []*routev3.RouteConfiguration{
			{
				Name:                     "front-proxy",
				IgnorePortInHostMatching: true,
				VirtualHosts: []*routev3.VirtualHost{
					{
						Name:    "local",
						Domains: []string{"*"},
						Routes: []*routev3.Route{
							&routev3.Route{
								Match: &routev3.RouteMatch{
									PathSpecifier: &routev3.RouteMatch_Prefix{
										Prefix: "/api/pomwm.",
									},
								},
								Action: routeToClusterAction,
							},
						},
						Cors: &routev3.CorsPolicy{
							AllowMethods: "*",
							AllowHeaders: "*",
							MaxAge:       "1728000",
							AllowOriginStringMatch: []*matcherv3.StringMatcher{{
								MatchPattern: &matcherv3.StringMatcher_Prefix{
									Prefix: "*",
								},
							}},
						},
					},
				},
			},
		},
	}

	val, _ := protojson.Marshal(serviceCfg)
	println(string(val))
}

func testUrl() {

	u := &url.URL{
		Path: "/asd/asdasd",
	}

	// q := u.Query()

	// q.Set("k", "Victor França Lopes")

	// u.RawQuery = q.Encode()

	fmt.Println(u.String())

	u2, err := url.Parse(u.String())

	if err != nil {
		fmt.Println(err)
		return
	}

	data, _ := json.Marshal(u2)
	fmt.Printf("%s\n", data)
	fmt.Println(u.Fragment, "=", u2.Fragment)
	fmt.Println(u.Fragment, "=", u2.Fragment)
}

func main() {

	// http1ServerConfig := &service_discovery.ServiceDiscovery_AddServiceClusterInput{
	// 	ServiceClusterName:   "http1-server",
	// 	XdsClusterName:       "xds-cluster",
	// 	ServiceType:          service_discovery.ServiceDiscovery_SERVICE_TYPE_HTTP1_SERVER,
	// 	MinHealthyPercentage: 70,
	// 	IngressType:          service_discovery.ServiceDiscovery_INGRESS_TYPE_WEB_SECURE_TRAFFIC,
	// 	Routing: &service_discovery.ServiceDiscovery_Routing{
	// 		MatchDomains:      []string{"backstage.dev.*"},
	// 		MatchPathPrefixes: []string{"/"},
	// 	},
	// 	UpstreamPort: 7007,
	// }

	// grcpServiceConfig := &service_discovery.ServiceDiscovery_AddServiceClusterInput{
	// 	ServiceClusterName:   "grpc-server",
	// 	XdsClusterName:       "xds-cluster",
	// 	ServiceType:          service_discovery.ServiceDiscovery_SERVICE_TYPE_GRPC_SERVICE,
	// 	MinHealthyPercentage: 70,
	// 	IngressType:          service_discovery.ServiceDiscovery_INGRESS_TYPE_WEB_SECURE_TRAFFIC,
	// 	Routing: &service_discovery.ServiceDiscovery_Routing{
	// 		MatchDomains:      []string{"*"},
	// 		MatchPathPrefixes: []string{"/grpc.reflection", "/wealth.proto"},
	// 	},
	// 	UpstreamPort: 9090,
	// }

	testUrl()
}
