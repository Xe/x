.PHONY: build dep install test

test_setup:
	cd _tests && vagrant up && cd -

test:
	go test ./... -cover

testverbose:
	go test ./... -v -cover
