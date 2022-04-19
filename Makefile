help:
	@grep -E '^[a-z][^:]+:.*?## .*$$' $(MAKEFILE_LIST) | sed "s/Makefile://g" | sed "s/:.*## /::/g" | \
    		awk 'BEGIN {FS = "::"}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' | \
    		awk 'BEGIN {FS = ":"}; {printf "%s \033[36m%s\033[0m\n", $$1, $$2}'

api_gen: ## Generates Go Code from openapi.yaml
	docker run --rm -v "${PWD}:/local" openapitools/openapi-generator-cli generate \
		-i local/api/openapi.yaml -g go-server -o local/api/generated --minimal-update

test: ## Run tests
	go test ./... -cover -race