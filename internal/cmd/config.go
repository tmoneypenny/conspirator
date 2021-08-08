package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configureSuggestion = fmt.Sprintf(`
Did you try this?
	%s

Run '%s %s --help' for usage.
`, configCmd.Use, ProjectName, configCmd.Use)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "write a configuration file",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		generateConfig()
	},
}

type Configuration struct {
	Domain           string            `json:"domain"`
	PublicAddress    string            `json:"publicAddress"`
	LogLevel         string            `json:"logLevel"`
	PollingEncoding  string            `json:"pollingEncoding"`
	MaxPollingEvents int               `json:"maxPollingEvents"`
	HTTP             HTTPConfiguration `json:"http"`
	DNS              DNSConfiguration  `json:"dns"`
}

type DNSConfiguration struct {
	Zones     []string       `json:"zones"`
	Listeners []DNSListeners `json:"listeners"`
}

type DNSListeners struct {
	Address string                `json:"address"`
	Proto   string                `json:"proto"`
	Port    int                   `json:"port"`
	TLS     *DNSTransportSecurity `json:"tls,omitempty"`
}

type DNSTransportSecurity struct {
	PublicKey  string `json:"publicKey"`
	PrivateKey string `json:"privateKey"`
}

type HTTPConfiguration struct {
	EnableV2     bool            `json:"enableV2"`
	Username     string          `json:"username"`
	Password     string          `json:"password"`
	CsrfKey      string          `json:"csrfKey"`
	SigningKey   string          `json:"signingKey"`
	TemplatePath string          `json:"templatePath"`
	Listeners    []HTTPListeners `json:"listeners"`
}

type HTTPListeners struct {
	Address          string                `json:"address"`
	PollingSubdomain string                `json:"pollingSubdomain"`
	AllowList        []string              `json:"allowlist"`
	Port             int                   `json:"port"`
	TLS              HTTPTransportSecurity `json:"tls"`
}

type HTTPTransportSecurity struct {
	Port       int    `json:"port"`
	PublicKey  string `json:"publicKey"`
	PrivateKey string `json:"privateKey"`
}

func generateConfig() {
	cfg, _ := json.MarshalIndent(Configuration{
		Domain:           "example.test.domain",
		PublicAddress:    "345.345.345.345",
		PollingEncoding:  "burp",
		MaxPollingEvents: 256,
		LogLevel:         "INFO",
		HTTP: HTTPConfiguration{
			EnableV2:     true,
			Username:     generateCredentials("username"),
			Password:     generateCredentials("password"),
			CsrfKey:      generateCredentials("csrfKey"),
			SigningKey:   generateCredentials("signingKey"),
			TemplatePath: "internal/pkg/http/template/",
			Listeners: []HTTPListeners{
				{
					Address:          "",
					PollingSubdomain: "polling",
					AllowList:        []string{"127.0.0.1", "192.168.0.1"},
					Port:             80,
					TLS: HTTPTransportSecurity{
						Port:       443,
						PublicKey:  "/usr/local/share/certs/star.example.test.domain/fullchain.pem",
						PrivateKey: "/usr/local/share/certs/star.example.test.domain/privkey.pem",
					},
				},
			},
		},
		DNS: DNSConfiguration{
			Zones: []string{"example.test.domain", "example2.test.domain"},
			Listeners: []DNSListeners{
				{
					Address: "",
					Proto:   "tcp",
					Port:    53,
				},
				{
					Address: "",
					Proto:   "udp",
					Port:    53,
				},
				{
					Address: "",
					Proto:   "tcp-tls",
					Port:    53,
					TLS: &DNSTransportSecurity{
						PublicKey:  "/usr/local/share/certs/star.example.test.domain/fullchain.pem",
						PrivateKey: "/usr/local/share/certs/star.example.test.domain/privkey.pem",
					},
				},
			},
		},
	}, "", "    ")

	fmt.Println(string(cfg))
	var resp, cfgPath string
	defaultCfgPath := "configs/" + ProjectName + ".config"
	fmt.Println("Would you like to write this to a file [Y/n]?")
	fmt.Scanln(&resp)
	if resp == "Y" {
		fmt.Printf("Config file path [Default: configs/%s.config]: ", ProjectName)
		fmt.Scanln(&cfgPath)
		if cfgPath == "" {
			fmt.Println("Using default filepath", defaultCfgPath)
		} else {
			fmt.Println("Writing to filepath:", cfgPath)
			defaultCfgPath = cfgPath
		}

		err := ioutil.WriteFile(defaultCfgPath, cfg, 0644)
		if err != nil {
			log.Fatal().Msgf("failed to write config: %v", err)
		}
	}
}

func generateCredentials(pwType string) string {
	var (
		lowerCharSet     = "abcdefghijklmnopqrstuvwxyz"
		upperCharSet     = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		specialCharSet   = "~!@#$%^&*_=+[]{}"
		numberSet        = "1234567890"
		usernameSet      = lowerCharSet
		passwordSet      = lowerCharSet + upperCharSet + specialCharSet + numberSet
		csrfSet          = lowerCharSet + numberSet
		SigningSet       = lowerCharSet + upperCharSet + numberSet
		usernameLength   = 8
		passwordLength   = 16
		signingKeyLength = 32
		csrfKeyLength    = 32
	)

	generator := func(length int, charSet string) string {
		var cred strings.Builder

		// we don't care if this is cryptographically secure as this
		// is intended to be used exclusively as an example. what is
		// generated is a temporary set of credentials the user is expected
		// to change
		for i := 0; i < length; i++ {
			rChar := rand.Intn(len(charSet))
			cred.WriteString(string(charSet[rChar]))
		}

		return cred.String()
	}

	switch pwType {
	case "username":
		return generator(usernameLength, usernameSet)
	case "password":
		return generator(passwordLength, passwordSet)
	case "csrfKey":
		return generator(csrfKeyLength, csrfSet)
	case "signingKey":
		return generator(signingKeyLength, SigningSet)
	}

	return ""
}

func initConfig() {
	if viper.GetString("config") == "" {
		viper.AddConfigPath(fmt.Sprintf("$HOME/%s/configs", ProjectName))
		viper.AddConfigPath(".")
		viper.AddConfigPath("./configs")
		viper.SetConfigName(fmt.Sprintf("%s.config", ProjectName))
		viper.SetConfigType("json")
	} else {
		viper.SetConfigType("json")
		viper.SetConfigFile(viper.GetString("config"))
	}

	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Error: %v\n%s", err, configureSuggestion)
		os.Exit(1)
	}

	fmt.Println(viper.Get("domain"))

	fmt.Println(viper.ConfigFileUsed())
	// check flag, if no flag, check default, otherwise recommend config command
}
