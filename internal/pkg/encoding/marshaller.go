package encoding

import (
	"github.com/labstack/echo/v4"
	"github.com/tmoneypenny/conspirator/internal/pkg/polling"
)

// Marshaller interface implements the builder pattern for
// constructing  JSON formats for desired output customization
type Marshaller interface {
	MarshalToJSON(interface{}) ([]byte, error)
	EventToBlob([]*polling.Event) ([]byte, error)
	EmptyResponse() []byte
}

// Format takes desired format for Marshaller
func Format(c string) Marshaller {
	if c == "burp" {
		return NewBurpMarshaller()
	}
	return NewBurpMarshaller()
}

// Marshal contains Marshaller method
type Marshal struct {
	Marshaller
}

// HTTPInput defines HTTPInput interaction data to marshal
type HTTPInput struct {
	Ctx      echo.Context
	Request  []byte
	Response []byte
}

// DNSInput defines DNS interaction data to marshal
type DNSInput struct {
	SubdomainQuestion string
	RawRequest        string
	RequestType       uint16
	Answer            string
	ClientIP          string
	OpCode            int
}

// RawInput is used as a generic input to the marshaller
type RawInput struct {
	InteractionURI string
	Protocol       string
	ClientIP       string
	Request        []byte
	Response       []byte
}

// NewMarshaller constructs a new JSON Marshaller
func NewMarshaller(m Marshaller) *Marshal {
	return &Marshal{m}
}
