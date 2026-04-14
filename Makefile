.PHONY: all stage1 runtime runtime-test test clean fmt docs repro

# Use gcc by default; override with CC=clang or CC=cc if desired.
ifeq ($(origin CC),default)
CC := gcc
endif
RT_SRC = runtime/src/mem.c runtime/src/panic.c runtime/src/io.c \
         runtime/src/proc.c runtime/src/time.c runtime/src/thread.c \
         runtime/src/sync.c
RT_INC = -Iruntime/include
RT_OUT = runtime/libfuse_rt.a

all: stage1

stage1:
	go build -o fuse ./cmd/fuse

runtime: $(RT_OUT)

$(RT_OUT): $(RT_SRC) runtime/include/fuse_rt.h
	$(CC) -c $(RT_INC) -std=c11 -Wall -Wextra -pedantic -O2 \
		runtime/src/mem.c -o runtime/src/mem.o
	$(CC) -c $(RT_INC) -std=c11 -Wall -Wextra -pedantic -O2 \
		runtime/src/panic.c -o runtime/src/panic.o
	$(CC) -c $(RT_INC) -std=c11 -Wall -Wextra -pedantic -O2 \
		runtime/src/io.c -o runtime/src/io.o
	$(CC) -c $(RT_INC) -std=c11 -Wall -Wextra -pedantic -O2 \
		runtime/src/proc.c -o runtime/src/proc.o
	$(CC) -c $(RT_INC) -std=c11 -Wall -Wextra -pedantic -O2 \
		runtime/src/time.c -o runtime/src/time.o
	$(CC) -c $(RT_INC) -std=c11 -Wall -Wextra -pedantic -O2 \
		runtime/src/thread.c -o runtime/src/thread.o
	$(CC) -c $(RT_INC) -std=c11 -Wall -Wextra -pedantic -O2 \
		runtime/src/sync.c -o runtime/src/sync.o
	ar rcs $(RT_OUT) runtime/src/*.o

runtime-test: $(RT_OUT)
	$(CC) $(RT_INC) -std=c11 -Wall -Wextra -O0 -g \
		runtime/tests/test_rt.c $(RT_OUT) -o runtime/tests/test_rt
	runtime/tests/test_rt

test: runtime-test
	go test ./...

clean:
	go clean ./...
	rm -f fuse fuse.exe
	rm -f runtime/src/*.o $(RT_OUT)
	rm -f runtime/tests/test_rt runtime/tests/test_rt.exe
	rm -f _fuse_rt_test_tmp.txt

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
