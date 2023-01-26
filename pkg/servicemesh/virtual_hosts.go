package servicemesh

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"

	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	matcherv3 "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/upper-institute/hike/pkg/helpers"
	"google.golang.org/protobuf/encoding/protojson"
)

type virtualHost struct {
	*routev3.VirtualHost
	corsAllowMethods           helpers.Set
	corsAllowHeaders           helpers.Set
	corsExposeHeaders          helpers.Set
	corsAllowOriginStringMatch helpers.Set
}

func (v *virtualHost) MergeCorsPolicy(cors *routev3.CorsPolicy) {

	if cors.AllowMethods != "" {
		v.corsAllowMethods.AddFromString(cors.AllowMethods, ",", " ")
	}

	if cors.AllowHeaders != "" {
		v.corsAllowHeaders.AddFromString(cors.AllowHeaders, ",", " ")
	}

	if cors.ExposeHeaders != "" {
		v.corsExposeHeaders.AddFromString(cors.ExposeHeaders, ",", " ")
	}

	if cors.AllowOriginStringMatch != nil {

		for _, matcher := range cors.AllowOriginStringMatch {
			buf, _ := protojson.Marshal(matcher)
			v.corsAllowOriginStringMatch.Add(string(buf))
		}
	}

	v.Cors.AllowMethods = v.corsAllowMethods.ToString(",")
	v.Cors.AllowHeaders = v.corsAllowHeaders.ToString(",")
	v.Cors.ExposeHeaders = v.corsExposeHeaders.ToString(",")

	v.Cors.AllowOriginStringMatch = []*matcherv3.StringMatcher{}

	for src := range v.corsAllowOriginStringMatch {

		matcher := &matcherv3.StringMatcher{}

		protojson.Unmarshal([]byte(src), matcher)

		v.Cors.AllowOriginStringMatch = append(v.Cors.AllowOriginStringMatch, matcher)

	}

}

type VirtualHostMap map[string]*virtualHost

func (v VirtualHostMap) MergeRoute(routeCfg *routev3.RouteConfiguration) {

	for _, routeVh := range routeCfg.VirtualHosts {

		for _, domain := range routeVh.Domains {

			if _, ok := v[domain]; !ok {

				h := md5.New()

				io.WriteString(h, domain)

				v[domain] = &virtualHost{
					VirtualHost: &routev3.VirtualHost{
						Name:    fmt.Sprintf("%s/%s", routeCfg.Name, hex.EncodeToString(h.Sum(nil))),
						Domains: []string{domain},
						Routes:  make([]*routev3.Route, 0),
						Cors: &routev3.CorsPolicy{
							MaxAge: "1728000",
						},
					},
					corsAllowMethods:           make(helpers.Set),
					corsAllowHeaders:           make(helpers.Set),
					corsExposeHeaders:          make(helpers.Set),
					corsAllowOriginStringMatch: make(helpers.Set),
				}

			}

			vh := v[domain]

			vh.Routes = append(vh.Routes, routeVh.Routes...)

			vh.MergeCorsPolicy(routeVh.Cors)

		}

	}

}

func (v VirtualHostMap) ToResourceSlice() []types.Resource {

	res := []types.Resource{}

	for _, vh := range v {
		res = append(res, vh.VirtualHost)
	}

	return res

}
