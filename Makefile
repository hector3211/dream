# Makefile to run Go program
#
# Variables
MAIN=cmd/main.go

# Targets
.PHONY: run

# Run the application
run:
	go run $(MAIN)

test: 
	go test cmd/main_test.go
