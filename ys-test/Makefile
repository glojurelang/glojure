M := $(or $(MAKES_REPO_DIR),.cache/makes)
$(shell [ -d $M ] || git clone -q https://github.com/makeplus/makes $M)
include $M/init.mk
include $M/clean.mk
MAKES-NO-RULES := true
GO-VERSION := 1.23.0
include $M/go.mk
include $M/ys.mk
include $M/shell.mk

PROGRAM := hello
PROGRAM-YS := $(PROGRAM).ys
PROGRAM-CLJ := $(PROGRAM).clj
PROGRAM-GO := $(PROGRAM).go

AOT-DIR := ../internal/aot
AOT := $(AOT-DIR)/glj-aot

override PATH := $(AOT-DIR):$(PATH)

MAKES-DISTCLEAN := .cache/
MAKES-CLEAN := \
  go.sum \
  hello \
  hello.clj \
  hello.go* \
  out* \
  $(AOT) \


test: $(PROGRAM)
	@echo
	time ./$<
	@echo
	time ./$< Glojure

$(PROGRAM): $(PROGRAM-GO)
	go mod tidy
	go build -o $@

$(PROGRAM-GO): $(PROGRAM-CLJ) $(AOT)
	glj-aot $< > $@

$(PROGRAM-CLJ): $(PROGRAM-YS) $(YS)
	ys -c $< > $@

$(AOT): $(GO) $(AOT).go
	(cd $(AOT-DIR) && go mod tidy && go build)
	[[ -x $@ ]]
