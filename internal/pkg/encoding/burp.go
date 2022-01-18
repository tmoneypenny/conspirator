package encoding

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	httpEncoding "github.com/tmoneypenny/conspirator/internal/pkg/encoding/http"
	"github.com/tmoneypenny/conspirator/internal/pkg/polling"
)

/*
This package is responsible for modeling the polling interaction to
create interoperability with Burp Collab
*/

// BurpResultsV4 matches the Burp X-Collaborator-Version: 4
// expected response
type BurpResultsV4 struct {
	Responses
}

// Responses returns all event Interactions in BurpResults format
type Responses struct {
	Interactions []Response `json:"responses"`
}

// Reponse.Data is of protocolType, e.g. HTTPResultDATA
type Response struct {
	Protocol      string      `json:"protocol"`
	OpCode        string      `json:"opCode"`
	InteractionID string      `json:"interactionString"`
	ClientPart    string      `json:"clientPart"`
	Time          string      `json:"time"`
	Data          interface{} `json:"data"`
	ClientIP      string      `json:"client"`
}

// HTTPResultData will contain b64 encoded strings
type HTTPResultData struct {
	Response string `json:"response"`
	Request  string `json:"request"`
}

// DNSResultData will contain b64 encoded strings
type DNSResultData struct {
	Subdomain  string `json:"subDomain"`
	Type       uint16 `json:"type"`
	RawRequest string `json:"rawRequest"`
}

// RawResultData will contain b64 encoded strings
type RawResultData struct {
	Response string `json:"response"`
	Request  string `json:"request"`
}

// BurpMarshaller implements the Marshaller interface
type BurpMarshaller struct {
	Ndots int
}

// extractInteraction returns the interactionID of an event
func (m *BurpMarshaller) extractInteraction(event string) string {
	subdomains := strings.Split(event, ".")
	if len(subdomains) <= m.Ndots+1 {
		return subdomains[0]
	}
	return subdomains[len(subdomains)-(m.Ndots+2)]
}

// EmptyResponse returns an empty responses struct
func (m *BurpMarshaller) EmptyResponse() []byte {
	var burpResults BurpResultsV4
	emptyResponse, _ := json.Marshal(burpResults)
	return emptyResponse
}

// EventToBlob converts the polling event into a JSON blob for use with
// JSONBlob response type
func (m *BurpMarshaller) EventToBlob(data []*polling.Event) ([]byte, error) {
	var burpResults BurpResultsV4
	var burpResponses []Response
	for _, e := range data {
		var results Response
		switch v := e.Data.(type) {
		case []byte:
			json.Unmarshal(v, &results)
			burpResponses = append(burpResponses, results)
		default:
			return nil, fmt.Errorf("invalid type for event blob")
		}
	}
	burpResults.Responses.Interactions = burpResponses
	return json.Marshal(burpResults)
}

// BurpMarshaller.MarshalToJSON
func (m *BurpMarshaller) MarshalToJSON(data interface{}) ([]byte, error) {
	switch d := data.(type) {
	case *HTTPInput:
		log.Debug().Msg("Got HTTP Event")
		jsonData, error := json.Marshal(&Response{
			Protocol:      "http",
			OpCode:        "1", // only seen opCode = 1
			InteractionID: m.extractInteraction(d.Ctx.Request().Host),
			ClientPart:    "0y", // not sure how this is used
			Time:          fmt.Sprint(time.Now().UnixNano() / int64(time.Millisecond)),
			Data: HTTPResultData{
				// Response requires the proto from the request.
				Response: base64.StdEncoding.EncodeToString(httpEncoding.WriteResponse(
					d.Ctx.Response(),
					httpEncoding.Protocol{
						ProtoMajor: d.Ctx.Request().ProtoMajor,
						ProtoMinor: d.Ctx.Request().ProtoMinor,
					},
					d.Response),
				),
				Request: base64.StdEncoding.EncodeToString(httpEncoding.WriteRequest(d.Ctx.Request())),
			},
			ClientIP: RemovePortFromClientIP(d.Ctx.Request().RemoteAddr),
		})

		return jsonData, error
	case *DNSInput:
		log.Debug().Msg("Got DNS Event")
		jsonData, error := json.Marshal(&Response{
			Protocol:      "dns",
			OpCode:        strconv.Itoa(d.OpCode + 1), // Map to Burp OpCodes?
			InteractionID: m.extractInteraction(strings.TrimSuffix(d.SubdomainQuestion, ".")),
			ClientPart:    "0y",
			Time:          fmt.Sprint(time.Now().UnixNano() / int64(time.Millisecond)), // Convert to unix
			Data: DNSResultData{
				Subdomain:  d.SubdomainQuestion,
				Type:       d.RequestType,
				RawRequest: base64.StdEncoding.EncodeToString([]byte(d.RawRequest)),
			},
			ClientIP: RemovePortFromClientIP(d.ClientIP),
		})
		return jsonData, error
	case *RawInput:
		log.Debug().Msg("Got Raw Event")
		jsonData, error := json.Marshal(&Response{
			Protocol:      d.Protocol,
			OpCode:        "1", // only seen opCode = 1
			InteractionID: d.InteractionURI,
			ClientPart:    "0y", // not sure how this is used
			Time:          fmt.Sprint(time.Now().UnixNano() / int64(time.Millisecond)),
			Data: RawResultData{
				Response: base64.StdEncoding.EncodeToString(d.Response),
				Request:  base64.StdEncoding.EncodeToString(d.Request),
			},
			ClientIP: RemovePortFromClientIP(d.ClientIP),
		})
		return jsonData, error
	default:
		log.Debug().Msg("Default event")
		return nil, nil
	}
}

// NewBurpMarshaller returns BurpMarshaller that allows
// data to be marshalled according to Burp JSON format
// by calling MarshalToJSON()
func NewBurpMarshaller() *BurpMarshaller {
	nSubdomains := strings.Split(viper.GetString("domain"), ".")
	return &BurpMarshaller{Ndots: len(nSubdomains) - 1}
}
