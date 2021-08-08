## Templates

Go does not statically link templates into the binary. It's possible, but it 
is a bit of an anti-pattern. Therefore, copying the templates to the target server
is advisable. By default, the executable will look in the same folder structure as
defined from the root of the project `internal/pkg/http/template/`.