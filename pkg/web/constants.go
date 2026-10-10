package web

import (
	. "github.com/BeamFoundry/rays/pkg/lang"
)

// Protocol is an Enum alias for ProtocolEnum
type Protocol = Enum[ProtocolEnum]

// ProtocolEnum is an Enum struct defining web protocols
type ProtocolEnum struct {
	HTTP,
	HTTPS,
	GRPC Protocol
}

// Protocols is an initialized ProtocolEnum variable.
var Protocols = Protocol{}.New()

// HttpMethod is an Enum aloas for HttpMethodEnum
type HttpMethod = Enum[HttpMethodEnum]

// HttpMethodEnum is an Enum struct defining HTTP Methods
type HttpMethodEnum struct {
	GET,
	HEAD,
	POST,
	PUT,
	PATCH,
	DELETE,
	CONNECT,
	OPTIONS,
	TRACE,
	USE HttpMethod
}

// HttpMethods is an initialized HttpMethodEnum variable.
var HttpMethods = HttpMethod{}.New()

// RequestMapping is a marker interface used to indicate a possible mapping for a [core.Controller] or a [core.Controller] method.
//
// It should contain in a struct tag the following elements :
//
//   - Path : the requested path URI to map to
//   - Method : the [HttpMethod] to map to
//
// NOTE: Rays primary module (this project) does not provide by itself a working core.Controller implementation.
type RequestMapping interface {
	_RequestMapping()
	_Path() string
	_Method() string
}
