NAME = magalix-policy-agent

VERSION = $(shell printf "%s.%s" \
	$$(git rev-list --count HEAD) \
	$$(git rev-parse --short HEAD) \
)

BRANCH = $(shell git rev-parse --abbrev-ref HEAD)

version:
	@echo $(VERSION)

build: build@go

test: @TODO add correct paths
	go test -v ./...

checkout:
	@go list -f '{{.Dir}}' k8s.io/klog \
		| xargs -n1 -I{} bash -c 'git -C {} checkout -q v0.4.0 || true'

build@go:
	@echo :: building go binary $(VERSION)
	@go get -v -d
	@make checkout
	@rm -rf build/agent
	CGO_ENABLED=0 GOOS=linux go build -o build/agent \
		-ldflags "-X main.version=$(VERSION)" \
		-gcflags "-trimpath $(GOPATH)/src"

image: strip
	@echo :: building image $(NAME):$(VERSION)
	@docker build -t $(NAME):$(VERSION) -f Dockerfile .

anchore_scan:
	@echo :: scanning image $(NAME):$(VERSION)
	@curl -s https://ci-tools.anchore.io/inline_scan-latest | bash -s -- -f -r "$(NAME):$(VERSION)"

push@%:
	$(eval VERSION ?= latest)
	$(eval TAG ?= $*/$(NAME):$(VERSION))
	@echo :: pushing image $(NAME):$(VERSION)
	@docker tag $(NAME):$(VERSION) $(TAG)
	@docker push $(TAG)

	@if [[ "$(tag-file)" ]]; then echo "$(TAG)" > "$(tag-file)"; fi
	@if [[ "$(version-file)" ]]; then echo "$(VERSION)" > "$(version-file)"; fi

mock:
	mockgen -package mock -destination sink/mock/mock.go github.com/MagalixCorp/magalix-policy-agent/pkg/domain ValidationResultSink
	mockgen -package mock -destination policies/mock/mock.go github.com/MagalixCorp/magalix-policy-agent/pkg/domain PoliciesSource
	mockgen -package mock -destination pkg/validation/mock/mock.go github.com/MagalixCorp/magalix-policy-agent/pkg/validation Validator
