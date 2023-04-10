all: test

test:
	go test ./... -race -v

coverage:
	go test ./... -coverprofile c.out
	@sed -i "s%github.com/AlexNabokikh/%%" c.out

.PHONY: all test coverage
