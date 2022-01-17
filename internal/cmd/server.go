package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"plugin"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"github.com/tmoneypenny/conspirator/internal/pkg/bind"
	"github.com/tmoneypenny/conspirator/internal/pkg/http"
	"github.com/tmoneypenny/conspirator/internal/pkg/polling"
	"github.com/tmoneypenny/conspirator/pkg/wrapper"
)

// configureBind builds a BindConfig by parsing the config file
func configureBind() *bind.BindConfig {
	bindServers := []bind.BindServerConfig{}

	for i, v := range viper.Get("dns.listeners").([]interface{}) {
		address := v.(map[string]interface{})["address"].(string)
		port := int(v.(map[string]interface{})["port"].(float64))
		proto := v.(map[string]interface{})["proto"].(string)
		tls := v.(map[string]interface{})["tls"]

		bindServers = append(bindServers, bind.BindServerConfig{
			Address: &address,
			Port:    &port,
			Net:     &proto,
		})

		if tls != nil {
			cert := tls.(map[string]interface{})["publicKey"].(string)
			pk := tls.(map[string]interface{})["privateKey"].(string)
			bindServers[i].TLS = &bind.TLSConfig{
				CertFile: &cert,
				KeyFile:  &pk,
			}
		}
	}

	return &bind.BindConfig{
		Zones:         viper.GetStringSlice("dns.zones"),
		Configs:       bindServers,
		PublicAddress: viper.GetString("publicAddress"),
	}
}

func configureHTTP() *http.HTTPServerConfig {
	v := viper.Get("http.listeners").([]interface{})
	address := v[0].(map[string]interface{})["address"].(string)
	polling := v[0].(map[string]interface{})["pollingSubdomain"].(string)
	allowList := v[0].(map[string]interface{})["allowlist"].([]interface{})
	port := int(v[0].(map[string]interface{})["port"].(float64))
	tls := v[0].(map[string]interface{})["tls"]
	v2 := viper.GetBool("http.enableV2")

	var allowedIPs []string
	if allowList != nil {
		for i := range allowList {
			allowedIPs = append(allowedIPs, allowList[i].(string))
		}
	}

	server := &http.HTTPServerConfig{
		Address:       &address,
		AllowList:     &allowedIPs,
		Port:          &port,
		PollingDomain: &polling,
		Version2:      &v2,
	}

	if tls != nil {
		cert := tls.(map[string]interface{})["publicKey"].(string)
		pk := tls.(map[string]interface{})["privateKey"].(string)
		tlsPort := int(tls.(map[string]interface{})["port"].(float64))
		server.TLS = &http.TLSConfig{
			CertFile: &cert,
			KeyFile:  &pk,
			Port:     &tlsPort,
		}
	}

	return server
}

// serverHandler is responsible for starting and stopping all extensions
func serverHandler() {
	// Start DNS
	log.Debug().Msg("Starting Server Handler")
	bindConfig := configureBind()
	httpConfig := configureHTTP()

	pm := polling.New(&polling.PollingConfig{
		MaxBufferSize: 250,
		DeleteAfter:   true,
	})

	manager := pm.Start()
	httpConfig.PollingManager = manager
	bindConfig.PollingManager = manager

	bind.BindServer(bindConfig).Start()
	http.HTTPServer(httpConfig).Start()

	extShutdown := make(chan bool)
	go extensionHandler(manager, extShutdown)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	<-c // block until signal caught

	// Stop services gracefully
	defer func() {
		log.Info().Msg("Shutting down services...")
		http.HTTPServer(httpConfig).Stop()
		bind.BindServer(bindConfig).Stop()
		extShutdown <- true // Initial plugin shutdown
		<-extShutdown       // Wait for shutdown
		manager.Stop()      // Stop polling server
		log.Info().Msg("Bye!")
	}()
}

// Modules must implement Start and Stop methods.
// Extension may interact with packages, but not vice-versa

// ExtensionPlugins
type ExtensionPlugins struct {
	PollingServer *polling.PollingServer
}

// extensionHandler is used to load modules defined in the config
func extensionHandler(pollingServer *polling.PollingServer, shutdown chan bool) {
	log.Debug().Msg("Starting extension handler...")

	plugins := viper.Get("plugins")
	log.Debug().Msg("Loading plugins...")

	extensions := ExtensionPlugins{
		PollingServer: pollingServer,
	}

	for k := range plugins.(map[string]interface{}) {
		extensions.extensionLoader(k)
	}

	log.Info().Msg("All plugins started!")
	<-shutdown

	log.Info().Msg("Unloading plugins...")
	for k := range plugins.(map[string]interface{}) {
		extensions.extensionUnloader(k)
	}

	shutdown <- true
}

func (e *ExtensionPlugins) extensionLoader(pluginName string) {
	p, err := plugin.Open(viper.GetString("pluginsDirectory") + fmt.Sprintf("%s.so", pluginName))
	if err != nil {
		log.Fatal().Msgf("Failed to open plugin: %s [%v]", pluginName, err)
	}

	server, err := p.Lookup("NewServer")
	if err != nil {
		log.Fatal().Msgf("Failed to load plugins: %v", err)
	}

	// Start Server
	server.(func(cfg wrapper.Config) wrapper.Module)(wrapper.Config{
		PollingManager: e.PollingServer,
	}).Start()

}

func (e *ExtensionPlugins) extensionUnloader(pluginName string) {
	p, err := plugin.Open(viper.GetString("pluginsDirectory") + fmt.Sprintf("%s.so", pluginName))
	if err != nil {
		log.Fatal().Msgf("Failed to open plugin symbols: %s [%v]", pluginName, err)
	}

	server, err := p.Lookup("NewServer")
	if err != nil {
		log.Fatal().Msgf("Failed to load plugins: %v", err)
	}

	// Stop Server
	server.(func(cfg wrapper.Config) wrapper.Module)(wrapper.Config{
		PollingManager: e.PollingServer,
	}).Stop()
}
