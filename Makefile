all: linux-binaries windows-binaries

clean:
	rm -rf bin

ALL_GO_FILES := $(shell find ./ -name '*.go')

export CGO_ENABLED=0

.PHONY: linux-binaries
linux-binaries: bin/ncc-neighborhood

bin/ncc-neighborhood: $(ALL_GO_FILES)
	@mkdir -p bin
	go build -o $@ ./cmd/ncc-neighborhood/...

.PHONY: windows-binaries
windows-binaries: bin/ncc-neighborhood.exe

bin/ncc-neighborhood.exe: $(ALL_GO_FILES)
	@mkdir -p bin
	GOOS=windows go build -o $@ ./cmd/ncc-neighborhood/...
