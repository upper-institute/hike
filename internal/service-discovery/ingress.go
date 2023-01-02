package servicediscovery

import (
	"strings"

	xdscorev3 "github.com/cncf/xds/go/xds/core/v3"
	v3 "github.com/cncf/xds/go/xds/type/matcher/v3"
	accesslogv3 "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	streamv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/stream/v3"
	matchingv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/common/matching/v3"
	corsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/cors/v3"
	grpcwebv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/grpc_web/v3"
	health_checkv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/health_check/v3"
	routerv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	http_connection_managerv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	matcherv3 "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	typev3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	service_discovery "github.com/upper-institute/ops-control/gen/api/service-discovery"
	"google.golang.org/protobuf/types/known/anypb"
)

func makeGrpcWebMatcher(extensionConfig *corev3.TypedExtensionConfig) (*http_connection_managerv3.HttpFilter, error) {

	requestHeaderInput := &matcherv3.HttpRequestHeaderMatchInput{
		HeaderName: "content-type",
	}

	requestHeaderInputAny, err := anypb.New(requestHeaderInput)
	if err != nil {
		return nil, err
	}

	extensionWithMatcher := &matchingv3.ExtensionWithMatcher{
		ExtensionConfig: extensionConfig,
		XdsMatcher: &v3.Matcher{
			MatcherType: &v3.Matcher_MatcherList_{
				MatcherList: &v3.Matcher_MatcherList{
					Matchers: []*v3.Matcher_MatcherList_FieldMatcher{
						{
							Predicate: &v3.Matcher_MatcherList_Predicate{
								MatchType: &v3.Matcher_MatcherList_Predicate_NotMatcher{
									NotMatcher: &v3.Matcher_MatcherList_Predicate{
										MatchType: &v3.Matcher_MatcherList_Predicate_SinglePredicate_{
											SinglePredicate: &v3.Matcher_MatcherList_Predicate_SinglePredicate{
												Input: &xdscorev3.TypedExtensionConfig{
													Name:        "content-type",
													TypedConfig: requestHeaderInputAny,
												},
												Matcher: &v3.Matcher_MatcherList_Predicate_SinglePredicate_ValueMatch{
													ValueMatch: &v3.StringMatcher{
														MatchPattern: &v3.StringMatcher_Prefix{
															Prefix: "application/grpc-web",
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	extensionWithMatcherAny, err := anypb.New(extensionWithMatcher)
	if err != nil {
		return nil, err
	}

	return &http_connection_managerv3.HttpFilter{
		Name: wellknown.GRPCWeb,
		ConfigType: &http_connection_managerv3.HttpFilter_TypedConfig{
			TypedConfig: extensionWithMatcherAny,
		},
	}, nil

}

func makeGrpcWebFilter() (*http_connection_managerv3.HttpFilter, error) {

	grpcWebFilter := &grpcwebv3.GrpcWeb{}

	grpcWebFilterAny, err := anypb.New(grpcWebFilter)
	if err != nil {
		return nil, err
	}

	return makeGrpcWebMatcher(&corev3.TypedExtensionConfig{
		Name:        wellknown.GRPCWeb,
		TypedConfig: grpcWebFilterAny,
	})

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

func makeHealthCheckFilter(input *service_discovery.Ingress) (*http_connection_managerv3.HttpFilter, error) {

	healthCheckPath := "/healthz"

	if len(input.HealthCheck.Path) > 0 {
		healthCheckPath = input.HealthCheck.Path
	}

	healthyPercentages := make(map[string]*typev3.Percent)

	for clusterName, percentage := range input.HealthCheck.ClusterMinHealthyPercentages {
		healthyPercentages[clusterName] = &typev3.Percent{
			Value: float64(percentage),
		}
	}

	healthCheck := &health_checkv3.HealthCheck{
		ClusterMinHealthyPercentages: healthyPercentages,
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

func makeHttpFilters(input *service_discovery.Ingress) ([]*http_connection_managerv3.HttpFilter, error) {

	httpFilters := make([]*http_connection_managerv3.HttpFilter, 0)

	if input.HealthCheck != nil {

		healthCheckFilter, err := makeHealthCheckFilter(input)
		if err != nil {
			return nil, err
		}

		httpFilters = append(httpFilters, healthCheckFilter)

	}

	if input.EnableCors {

		corsFilter, err := makeCorsFilter()
		if err != nil {
			return nil, err
		}

		httpFilters = append(httpFilters, corsFilter)

	}

	if input.GrpcWeb != nil && input.GrpcWeb.Enabled {

		if input.GrpcWeb.JwtAuthentication != nil {

			jwtAuthnAny, err := anypb.New(input.GrpcWeb.JwtAuthentication)
			if err != nil {
				return nil, err
			}

			jwtAuthFilter, err := makeGrpcWebMatcher(&corev3.TypedExtensionConfig{
				Name:        "envoy.filters.http.jwt",
				TypedConfig: jwtAuthnAny,
			})
			if err != nil {
				return nil, err
			}

			httpFilters = append(httpFilters, jwtAuthFilter)

		}

		grpcWebFilter, err := makeGrpcWebFilter()
		if err != nil {
			return nil, err
		}

		httpFilters = append(httpFilters, grpcWebFilter)

	}

	routerFilter, err := makeHttpRouterFilter()
	if err != nil {
		return nil, err
	}

	httpFilters = append(httpFilters, routerFilter)

	return httpFilters, nil

}

func makeHttpConnectionManagerFilter(input *service_discovery.Ingress) (*listenerv3.Filter, error) {

	httpFilters, err := makeHttpFilters(input)
	if err != nil {
		return nil, err
	}

	accessLogStdout := &streamv3.StdoutAccessLog{}

	accessLogStdoutAny, err := anypb.New(accessLogStdout)
	if err != nil {
		return nil, err
	}

	httpConnManager := &http_connection_managerv3.HttpConnectionManager{
		CodecType:  http_connection_managerv3.HttpConnectionManager_AUTO,
		StatPrefix: strings.ToLower(input.IngressType.String()),
		AccessLog: []*accesslogv3.AccessLog{
			{
				Name: "stdout",
				ConfigType: &accesslogv3.AccessLog_TypedConfig{
					TypedConfig: accessLogStdoutAny,
				},
			},
		},
		RouteSpecifier: &http_connection_managerv3.HttpConnectionManager_Rds{
			Rds: &http_connection_managerv3.Rds{
				ConfigSource: &corev3.ConfigSource{
					ResourceApiVersion: resource.DefaultAPIVersion,
					ConfigSourceSpecifier: &corev3.ConfigSource_ApiConfigSource{
						ApiConfigSource: &corev3.ApiConfigSource{
							ApiType:                   corev3.ApiConfigSource_AGGREGATED_GRPC,
							TransportApiVersion:       resource.DefaultAPIVersion,
							SetNodeOnFirstMessageOnly: true,
							GrpcServices: []*corev3.GrpcService{{
								TargetSpecifier: &corev3.GrpcService_EnvoyGrpc_{
									EnvoyGrpc: &corev3.GrpcService_EnvoyGrpc{
										ClusterName: input.XdsClusterName,
									},
								},
							}},
						},
					},
				},
				RouteConfigName: input.IngressType.String(),
			},
		},
		HttpFilters: httpFilters,
	}

	httpConnManagerAny, err := anypb.New(httpConnManager)
	if err != nil {
		return nil, err
	}

	return &listenerv3.Filter{
		Name: wellknown.HTTPConnectionManager,
		ConfigType: &listenerv3.Filter_TypedConfig{
			TypedConfig: httpConnManagerAny,
		},
	}, nil
}
