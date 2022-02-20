## Templates

Go does not statically link templates into the binary. It's possible, but it 
is a bit of an anti-pattern. Therefore, copying the templates to the target server
is advisable. By default, the executable will look in the same folder structure as
defined from the root of the project `internal/pkg/http/template/`. It is possible to override this path by setting the `http.templatePath` setting in the configuration.

## Static

Static content can be hosted from the HTTP server by setting `http.static.enable: true`. Assets in `http.static.path` will be available on the server under the `http.static.prefix` path.

| Setting | Value Type | Default | Description |
| ------- | ---------- | ------- | ----------- |
| `browsing` | bool | `true` | Enable directory browsing (indexing) |
| `enable` | bool | `true` | Enable the HTTP server to host static content |
| `path` | string | `static/` | Directory on disk with static assets to serve |
| `prefix` | string | `/repository` | Path prefix to access the static content via URL |