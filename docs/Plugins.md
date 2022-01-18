## Plugins

Conspirator plugins are shared libaries loaded at runtime that extend the capabilities of the server. 

### Writing a plugin

Plugins for conspirator must import `pkg/wrapper` to provide compatibility. In addition, the plugin must implement the following exported methods:
- `NewServer(wrapper.Config) wrapper.Module`
- `Start()`
- `Stop()`

The method receiver should contain a reference to `PollingManager *polling.PollingServer`, which allows the plugin to write events to the polling server.

```
## plugin/example.go

import (
    "github.com/tmoneypenny/conspirator/pkg/wrapper"
)

type Server struct {}

func NewServer(cfg wrapper.Config) wrapper.Module {
    // cfg.PollingManager 
    return &Server{}
}

func (s *Server) Start() {}

func (s *Server) Stop() {}


## running plugin
server, _ := plugin.Lookup("NewServer")
server.(func(cfg wrapper.Config) wrapper.Module)(wrapper.Config{
    PollingManager: pollingServer
}).Start()
```

### Example

An example plugin can be found in `plugins/ldap` that implements an LDAP(s) server supporting Search lookups and anonymous binds.

### Building the LDAP plugin

Go plugins are tightly coupled to the binaries wherre they are dynamically loaded. This means that every time the plugin, and/or the binary, are update both need to be recompiled. While this makes shared libraries less portable than their counterparts, it still provides dynamic loading of plugins based on configuration at runtime. 

`go build -buildmode=plugin -o plugins/ldap.so plugins/ldap/*.go`

The package and library should be compiled on the system where you intended to run the server, otherwise you may experience inconsistent results and runtime errors.