M := $(or $(MAKES_REPO_DIR),.cache/makes)
$(shell [ -d $M ] || git clone -q https://github.com/makeplus/makes $M)
include $M/init.mk
include $M/clean.mk
MAKES-NO-RULES := true
include $M/go.mk
include $M/ys.mk
include $M/shell.mk

PROGRAM := $(or $(prog),hello)
PROGRAM := $(PROGRAM:%.ys=%)
PROGRAM-YS := $(PROGRAM).ys
PROGRAM-CLJ := $(PROGRAM).clj
PROGRAM-GO := $(PROGRAM).go

MAKES-CLEAN := go.mod *.clj *.go hello

AOT := ../internal/aot/clj-to-go.go


run: $(PROGRAM)
	$< $(args)

$(PROGRAM): $(PROGRAM-GO) $(GO)
	$(RM) go.mod
	go mod init $@
	go build $<

$(PROGRAM-GO): $(PROGRAM-CLJ) $(GO)
	go run $(AOT) $< > $@

$(PROGRAM-CLJ): $(PROGRAM-YS) $(YS)
	ys -c $< > $@
