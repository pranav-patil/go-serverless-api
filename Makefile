.PHONY: build clean
.PHONY: lint
.PHONY: mocks

# Get all function binaries for this code base
# Find all directories with .go files
GOFILES := $(shell find . -name "*.go")

TARGETS=$(sort $(dir $(wildcard func/*/*.go)))
FUNCTIONS=$(addsuffix bootstrap,$(TARGETS))
PACKAGES=$(FUNCTIONS:/bootstrap=.zip)
ARTIFACT=bin/

LOCALSTACK_DIR=integration_test/localstack

build-ci: test $(ARTIFACT) $(FUNCTIONS) $(PACKAGES)
build: lint build-ci

tidy: | node_modules/go.mod
	go mod tidy

.PHONY: update-dependencies
update-dependencies:
	@echo "Updating Go dependencies"
	@cat go.mod | grep -E "^\t" | grep -v "// indirect" | cut -f 2 | cut -d ' ' -f 1 | xargs -n 1 -t go get -d -u
	@go mod vendor
	@go mod tidy

lint: mocks
	@if [ -z "$$(command -v golangci-lint)" ]; then \
    	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
    fi

	@golangci-lint run ./...

test: mocks
	go test -v -failfast -cover $$(go list ./... | grep -v node_modules | grep -v integration_test) -coverprofile=coverage.out

mocks:
	@if [ -z "$$(command -v mockgen)" ]; then \
    	go install github.com/golang/mock/mockgen@v1.6.0; \
    fi
	
	go generate ./...

%/bootstrap: %/*.go | node_modules/go.mod
	go test -v ./$*
	env GOARCH=amd64 GOOS=linux go build -tags lambda.norpc -o $@ ./$*
	zip -FS -j $*.zip $@
	cp $*.zip $(ARTIFACT)


coverage:
	go tool cover -html=coverage.out

# node_modules/go.mod used to ignore possible go modules in node_modules.
node_modules/go.mod:
	-@touch $@

$(ARTIFACT):
	@mkdir -p $(dir $(ARTIFACT))
	@echo
	@echo ====== Build Successful ========
	@echo

clean:
	rm -vf coverage.*
	rm -vf *.pid
	find . -name mocks -type d -exec rm -rf {} +
	find . -name gomock_reflect_* -type d -exec rm -rf {} +
	$(RM) $(FUNCTIONS) $(PACKAGES)
	$(RM) -r $(ARTIFACT)

export STAGE := testing
export LOG_LEVEL := debug
export BOOKMARK_URLS_BUCKET := integration-test-bookmark-urls
export BOOKMARK_WEBPAGES_BUCKET := integration-test-bookmark-webpages

.PHONY: localstack-start
localstack-start:
	export LOCALSTACK_VOLUME_DIR=$(pwd)/.filesystem/var/lib/localstack
	docker-compose -f $(LOCALSTACK_DIR)/docker-compose.localstack.yml up -d
	go run integration_test/localstack/localstack_infra.go setup

.PHONY: localstack-stop
localstack-stop:
	docker-compose -f $(LOCALSTACK_DIR)/docker-compose.localstack.yml stop

.PHONY: localstack-destroy
localstack-destroy:
	docker-compose -f $(LOCALSTACK_DIR)/docker-compose.localstack.yml down --rmi all -v --remove-orphans
	$(RM) -r $(LOCALSTACK_DIR)/volume

.PHONY: runapi
runapi:
	go run ./func/api

INTTEST_PID_FILE := int_test.pid

runapi-bkgrd: $(GITALY_PID_FILE)
	go run ./func/api & echo "$$!" > $(INTTEST_PID_FILE);\
	sleep 10

.PHONY: int-test
int-test:
	go test -timeout 300s -v -failfast -cover $$(go list ./integration_test/integration/...) -coverprofile=coverage.out

runapi-kill:
	if [ -f "$(INTTEST_PID_FILE)" ] ; then \
		echo "Clean up integration test" ;\
		kill -9 $$(cat $(INTTEST_PID_FILE)) ;\
		rm $(INTTEST_PID_FILE) ;\
	else \
		echo "Integration test not running" ;\
	fi ;\

.PHONY: run-int-test
run-int-test: runapi-kill localstack-start runapi-bkgrd int-test runapi-kill localstack-stop
