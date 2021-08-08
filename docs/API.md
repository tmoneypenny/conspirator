## The API docs are provided by Swagger

Navigate to the project root:

Build Versioned:
`swag init -dir internal/pkg/http/api/v1/ -g api.go --output internal/pkg/http/api/v1/docs`

Once built, you can access the server after authenticating using:
`http[s]://localhost:<port>/admin/docs/index.html`