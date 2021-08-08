package bind

import (
	"testing"
	"time"

	"github.com/tmoneypenny/conspirator/internal/pkg/util"
)

// dig <domain> @address -p <port>
// kdig -d @127.0.0.1 -p 8854 +tls-ca +tls-host=default.src.properties a.default.src.properties
// use 0.0.0.0 to bind to any interface
func TestBindServer(t *testing.T) {
	// domain sets UpsertRRS test domain
	domain = "default.src.properties"

	TestUpsertRRS(t)
	BindServer(&BindConfig{
		Configs: []BindServerConfig{
			{
				Address: util.StrToPtr("127.0.0.1"),
				Port:    util.IntToPtr(8853),
				Net:     util.StrToPtr("udp"),
			},
			{
				Address: util.StrToPtr("127.0.0.1"),
				Port:    util.IntToPtr(8853),
				Net:     util.StrToPtr("tcp"),
			},
			{
				Address: util.StrToPtr("127.0.0.1"),
				Port:    util.IntToPtr(8853),
				Net:     util.StrToPtr("tcp-tls"),
				TLS: &TLSConfig{
					CertFile: util.StrToPtr("/home/tmoneypenny/Documents/conf/collab-bundle/certs/star.default.src.properties/fullchain.pem"),
					KeyFile:  util.StrToPtr("/home/tmoneypenny/Documents/conf/collab-bundle/certs/star.default.src.properties/privkey.pem"),
				},
			},
		},
		Zones: []string{domain},
	}).Start()

	/* keep alive for longer testing
	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	*/
	time.Sleep(1 * time.Second)
}
