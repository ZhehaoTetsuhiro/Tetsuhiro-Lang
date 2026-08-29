# 注曰「此乃哲浩之谕之敕令簿（Makefile），总辖编译器 zhc 之铸、装、涤、验诸务」

# 注曰「诸名聚于此，改一处而全局从之」
GO      ?= go
SRC_DIR := zhc
SOURCES := $(wildcard $(SRC_DIR)/*.go) $(SRC_DIR)/go.mod
BIN     := zhc/zhc
ZHC_DIR := $(HOME)/.zhc
ZHC_EXE := $(ZHC_DIR)/zhc
BASHRC  := $(HOME)/.bashrc

# 注曰「凡虚拟之务，皆列于此，免与凡文件同名相扰」
.PHONY: all build install uninstall check clean help

# 注曰「不唤何名，则默认行 all，all 唯铸器而已」
.DEFAULT_GOAL := all

all: build

# 注曰「铸器：聚 zhc 下诸 Go 卷，炼为二进制 zhc/zhc」
build: $(BIN)

$(BIN): $(SOURCES)
	@echo 注曰「炼石为器：go build → $(BIN)」
	cd '$(SRC_DIR)' && $(GO) build -o ../$@ .

# 注曰「安装有三事：筑居于 ~/.zhc，纳器入居，录寻径于 ~/.bashrc」
install: build
	@if [ -e '$(ZHC_DIR)' ] && [ ! -d '$(ZHC_DIR)' ]; then \
		echo 注曰「~/.zhc 今为一凡文件，不可为器之居所，请先移去之」 >&2; exit 1; fi
	@mkdir -p '$(ZHC_DIR)'
	@install -m 0755 '$(BIN)' '$(ZHC_EXE)'
	@echo 注曰「器已安于 ~/.zhc/zhc，其名曰 zhc」
	@if grep -qsF 'HOME/.zhc' '$(BASHRC)'; then \
		echo 注曰「.bashrc 已录此寻径，不复重书」; \
	else \
		printf '\n%s\n' 'export PATH="$$HOME/.zhc:$$PATH" # 注曰「哲浩之谕（zhc）居于~/.zhc，纳为寻径」' >> '$(BASHRC)'; \
		echo 注曰「已录寻径于 ~/.bashrc，新开一 shell 或 source 之乃效」; \
	fi

# 注曰「卸载：毁器于 ~/.zhc，涤寻径之录于 ~/.bashrc」
uninstall:
	@rm -f '$(ZHC_EXE)'
	@rmdir '$(ZHC_DIR)' 2>/dev/null || true
	@if [ -f '$(BASHRC)' ]; then sed -i '\|HOME/\.zhc|d' '$(BASHRC)'; fi
	@echo 注曰「器与寻径之录皆已涤除」

# 注曰「验器：取 例/首例.浩 一界，铸之且 --run 即行，以观编译器之效」
check: build
	@mkdir -p _build
	@cd _build && ../$(BIN) --run ../例/首例.浩

# 注曰「涤除：仅去根下所铸之 zhc，_build 诸物不惊」
clean:
	@rm -f '$(BIN)'
	@echo 注曰「根下之 zhc 已去」

# 注曰「垂示诸务之名，以助来者」
help:
	@echo 注曰「哲浩之谕（zhc）之敕令，可用者如下：」
	@echo '  make build     铸编译器 zhc 于 zhc/zhc'
	@echo '  make install   安器于 ~/.zhc/zhc，且录 ~/.zhc 于 ~/.bashrc 之 PATH'
	@echo '  make uninstall 毁器于 ~/.zhc，并涤 ~/.bashrc 中寻径之录'
	@echo '  make check     取 例/首例.浩 试铸且运行，以验其器'
	@echo '  make clean     去 zhc/zhc'
	@echo '  make help      垂示此篇'
