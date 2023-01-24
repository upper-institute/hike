package parameter

import (
	"context"
	"net/url"
	"os"
	"strings"

	paramapi "github.com/upper-institute/hike/proto/api/parameter"
)

type SourceOptions struct {
	*ParameterOptions
	Store Store
}

func (options *SourceOptions) NewFromURLString(urlStr string) (*Source, error) {

	uri, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	kv := make(map[string]*Parameter)

	return &Source{uri, options, kv}, nil

}

type Source struct {
	uri     *url.URL
	options *SourceOptions
	kv      map[string]*Parameter
}

func (c *Source) Restore(ctx context.Context) error {

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

		case param, ok := <-pullReq.Result:
			if !ok {
				return nil
			}
			c.kv[param.key] = param

		}

	}

}

func (c *Source) Has(key string) bool {
	_, ok := c.kv[key]
	return ok
}

func (c *Source) HasWellKnown(wellKnown paramapi.WellKnown) bool {
	return c.Has(wellKnown.String())
}

func (c *Source) Get(key string) *Parameter {
	return c.kv[key]
}

func (c *Source) GetWellKnown(key paramapi.WellKnown) *Parameter {
	return c.kv[key.String()]
}

func (c *Source) List() []*Parameter {

	list := make([]*Parameter, 0)

	for _, param := range c.kv {
		list = append(list, param)
	}

	return list

}

func (c *Source) RestoreFromProcessEnvs() error {

	envs := os.Environ()

	for _, env := range envs {

		sep := strings.IndexRune(env, '=')

		key := env[:sep]
		value := env[sep+1:]

		param, err := c.options.ParameterOptions.NewFromURI(key, &url.URL{
			Fragment: value,
			Scheme:   VarScheme,
		})
		if err != nil {
			return err
		}

		c.kv[key] = param

	}

	return nil
}
