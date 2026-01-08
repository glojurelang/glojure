M := $(or $(MAKES_REPO_DIR),.cache/makes)
M := .cache/makes
C := 46b9706d4af95d678935dba931d8c4c84eaebbe5
$(shell [ -d $M ] || git clone -q https://github.com/makeplus/makes $M)
$(shell [ -d $M ] || ( \
  git clone -depth=1 -q https://github.com/makeplus/makes $M && \
  git -C $M reset -q --hard $C))
include $M/init.mk
MAKES-NO-RULES := true
GO-VERSION ?= 1.19.3
include $M/go.mk
include $M/clean.mk
include $M/shell.mk

MAKES-DISTCLEAN := .cache/ .clj-kondo/ .lsp/ .vscode/


all \
aot \
build \
format \
generate \
glj-bins \
glj-imports \
stdlib-targets \
test \
test-glj \
test-suite \
update-clojure-sources \
vet \
:: $(GO)
