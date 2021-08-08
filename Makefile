.Default_GOAL := build

BIN_FILE=conspirator

build:
	swag init -dir internal/pkg/http/api/v1/ -g api.go --output internal/pkg/http/api/v1/docs
	go build -o "${BIN_FILE}" cmd/conspirator/main.go 

bundle:
	swag init -dir internal/pkg/http/api/v1/ -g api.go --output internal/pkg/http/api/v1/docs
	go build -o "${BIN_FILE}" cmd/conspirator/main.go 
	mkdir -p bundle/bin/ bundle/config/ bundle/templates/
	cp "${BIN_FILE}" bundle/bin/${BIN_FILE}
	cp internal/pkg/http/template/*.tmpl bundle/templates/
	cp configs/conspirator.config bundle/config/conspirator.config
	tar -czf "${BIN_FILE}.tgz" bundle/
	rm -rf bundle/ "${BIN_FILE}"

run: 
	./${BIN_FILE} start

debug:
	swag init -dir internal/pkg/http/api/v1/ -g api.go --output internal/pkg/http/api/v1/docs
	go build -o "${BIN_FILE}" cmd/conspirator/main.go 
	./${BIN_FILE} start --profile

priv:
	swag init -dir internal/pkg/http/api/v1/ -g api.go --output internal/pkg/http/api/v1/docs
	go build -o "${BIN_FILE}" cmd/conspirator/main.go 
	sudo ./${BIN_FILE} start --profile

test:
	go test -v ./internal/pkg/...

clean:
	go clean
	rm "${BIN_FILE}"
	rm templates.tgz

