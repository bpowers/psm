all:
	go test
	go install

version:
	git describe --long | sed 's/v\([0-9]*\)\.\([0-9]*\)-\([0-9]*\).*/\1.\2.\3/' >version.txt

.PHONY: all version
