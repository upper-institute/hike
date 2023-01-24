package domainregistry

import "errors"

var (
	MissingWellKnownTlsAccountErr = errors.New("Missing well known tls account key in the parameter cache")
	UnknownPrivateKeyTypeErr      = errors.New("Unknown private key type")
)
