INTEGRATION_TEST_PATH?=./it

test.unit:
	go test -v ./lb

test.integration:
	go test -tags='integration' $(INTEGRATION_TEST_PATH) -count=1 -timeout=50m -v -run=$(INTEGRATION_TEST_SUITE_PATH) -failfast

run.load.balancer:
	go run main.go -port=8000

run.mock.server.1:
	go run mock/server.go -port=8081

run.mock.server.2:
	go run mock/server.go -port=8082

run.mock.server.3:
	go run mock/server.go -port=8083

run.mock.server.4:
	go run mock/server.go -port=8084
