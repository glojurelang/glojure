M := $(or $(MAKES_REPO_DIR),.cache/makes)
$(shell [ -d $M ] || git clone -q https://github.com/makeplus/makes $M)
include $M/init.mk
include $M/clean.mk
MAKES-NO-RULES := true
GO-VERSION := 1.25.0
include $M/go.mk
include $M/wasmtime.mk
include $M/ys.mk
include $M/shell.mk

PROGRAM := hello
PROGRAM-YS := $(PROGRAM).ys
PROGRAM-CLJ := $(PROGRAM).clj
PROGRAM-GO := $(PROGRAM).go
PROGRAM-WASM := $(PROGRAM).wasm

AOT-DIR := ../internal/aot
AOT := $(AOT-DIR)/glj-aot

override PATH := $(AOT-DIR):$(PATH)

MAKES-DISTCLEAN := .cache/
MAKES-CLEAN := \
  go.sum \
  hello \
  hello.clj \
  hello.go \
  hello.wasm \
  out* \
  $(AOT) \
  $(AOT-DIR)/go.sum \


test: test-aot test-wasm

test-aot: $(PROGRAM)
	@echo
	time ./$<
	@echo
	time ./$< Glojure
	@echo

test-wasm: $(PROGRAM-WASM) $(WASMTIME)
	@echo
	time wasmtime $<
	@echo
	time wasmtime $< Glojure
	@echo

$(PROGRAM): $(PROGRAM-GO)
	@echo
	$(RM) $@
	go mod tidy
	go build -o $@ $<
	[[ -f $@ ]]
	@echo

$(PROGRAM-WASM): $(PROGRAM-GO)
	@echo
	$(RM) $@
	go mod tidy
	GOOS=wasip1 GOARCH=wasm go build -o $@ $<
	[[ -f $@ ]]
	@echo

$(PROGRAM-GO): $(PROGRAM-CLJ) $(AOT)
	glj-aot $< > $@

$(PROGRAM-CLJ): $(PROGRAM-YS) $(YS)
	ys -c $< > $@

$(AOT): $(GO) $(AOT).go
	(cd $(AOT-DIR) && go mod tidy && go build)
	[[ -x $@ ]]
