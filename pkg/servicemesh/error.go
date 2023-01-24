package servicemesh

import "errors"

var (
	OneDiscoveryCyclePerServerErr = errors.New("Only one discovery cycle is allowed per server")
)
