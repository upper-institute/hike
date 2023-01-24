package parameter

import (
	"context"
	"net/url"
	"os"
	"strings"

	"github.com/upper-institute/ops-control/gen/api/parameter"
)

type CacheOptions struct {
	*ParameterOptions
	Store Store
}

func (options *CacheOptions) NewFromURLString(urlStr string) (*Cache, error) {

	uri, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	kv := make(map[string]*Parameter)

	return &Cache{uri, options, kv}, nil

}

type Cache struct {
	uri     *url.URL
	options *CacheOptions
	kv      map[string]*Parameter
}

func (c *Cache) Restore(ctx context.Context) error {

	pullReq := &PullRequest{
		ParameterOptions: c.options.ParameterOptions,
		Url:              c.uri,
		Result:           make(chan *Parameter),
	}

	endCh := make(chan error)

	go func() {
		endCh <- c.options.Store.Pull(ctx, pullReq)
		close(endCh)
	}()

	for {

		select {

		case err, ok := <-endCh:
			if err != nil {
				return err
			}
			if !ok {
				endCh = nil
			}

		case parameter, ok := <-pullReq.Result:
			if !ok {
				return nil
			}
			c.kv[parameter.key] = parameter

		}

	}

}

func (c *Cache) Has(key string) bool {
	_, ok := c.kv[key]
	return ok
}

func (c *Cache) HasWellKnown(wellKnown parameter.WellKnown) bool {
	return c.Has(wellKnown.String())
}

func (c *Cache) Get(key string) *Parameter {
	return c.kv[key]
}

func (c *Cache) GetWellKnown(key parameter.WellKnown) *Parameter {
	return c.kv[key.String()]
}

func (c *Cache) List() []*Parameter {

	list := make([]*Parameter, 0)

	for _, param := range c.kv {
		list = append(list, param)
	}

	return list

}

func (c *Cache) RestoreFromProcessEnvs() error {

	envs := os.Environ()

	for _, env := range envs {

		sep := strings.IndexRune(env, '=')

		key := env[:sep]
		value := env[sep+1:]

		parameter, err := c.options.ParameterOptions.NewFromURI(key, &url.URL{
			Fragment: value,
			Scheme:   VarScheme,
		})
		if err != nil {
			return err
		}

		c.kv[key] = parameter

	}

	return nil
}
