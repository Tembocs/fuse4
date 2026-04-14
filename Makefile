.PHONY: all stage1 runtime test clean fmt docs repro

all: stage1

stage1:
	go build -o fuse ./cmd/fuse

runtime:
	@echo "runtime: not yet implemented (Wave 08)"

test:
	go test ./...

clean:
	go clean ./...
	rm -f fuse fuse.exe

fmt:
	gofmt -s -w cmd/ compiler/

docs:
	@echo "docs: validating foundational documents exist"
	@test -f docs/language-guide.md     || (echo "MISSING: docs/language-guide.md"     && exit 1)
	@test -f docs/implementation-plan.md || (echo "MISSING: docs/implementation-plan.md" && exit 1)
	@test -f docs/repository-layout.md  || (echo "MISSING: docs/repository-layout.md"  && exit 1)
	@test -f docs/rules.md              || (echo "MISSING: docs/rules.md"              && exit 1)
	@test -f docs/learning-log.md       || (echo "MISSING: docs/learning-log.md"       && exit 1)
	@echo "docs: all foundational documents present"

repro:
	@echo "repro: not yet implemented (Wave 15)"
