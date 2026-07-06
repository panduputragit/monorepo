SERVICES := api-gateway auth-service employee-service branch-service member-service membership-service attendance-service notification-service payment-service
BUILD_TARGETS := $(addprefix build-,$(SERVICES))
VET_TARGET := $(addprefix vet-,$(SERVICES))
RUN_TARGETS := $(addprefix run-,$(SERVICES))

.PHONY: gen build-all deps help $(BUILD_TARGETS) $(RUN_TARGETS)

# Generate proto code
gen:
	@echo "Generating proto code..."
	cd protos && buf generate

build-all: $(BUILD_TARGETS)

$(BUILD_TARGETS):
	@service=$(@:build-%=%); \
	echo "Building $$service..."; \
	cd app/$$service && go build -buildvcs=false -o ../../bin/$$service ./cmd/server

$(VET_TARGET):
	@service=$(@:vet-%=%); \
	echo "ve $$service..."; \
	cd app/$$service && go vet -buildvcs=false -o ../../bin/$$service ./cmd/server

$(RUN_TARGETS):
	@service=$(@:run-%=%); \
	echo "Running $$service..."; \
	cd app/$$service && go run ./cmd/server

# Install dependencies
deps:
	@for service in packages $(SERVICES:%=app/%); do \
		echo "Tidying $$service..."; \
		(cd $$service && go mod tidy); \
	done

help:
	@echo "Available targets:"
	@echo "  gen          - Generate proto code"
	@echo "  build-all    - Build all services"
	@echo "  build-<name> - Build one service, e.g. build-auth-service"
	@echo "  run-<name>   - Run one service, e.g. run-api-gateway"
	@echo "  deps         - Install dependencies"
