GO ?= go
PYTHON ?= python3
SSH ?= ssh
SCP ?= scp
SSH_OPTS ?= -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o BatchMode=yes -o ConnectTimeout=5
SCP_OPTS ?= -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o BatchMode=yes -o ConnectTimeout=5
MKDIR_P ?= mkdir -p
RM := rm -rf

BIN_DIR := bin
LOG_DIR := logs
DOWNLOAD_DIR := $(LOG_DIR)/download
REMOTE_DIR ?= /osds_project2

MINER_BIN := $(BIN_DIR)/miner
CLIENT_BIN := $(BIN_DIR)/client
FAKEMINER_BIN := $(BIN_DIR)/fakeminer

COUNT ?= 5
DIFFICULTY ?= 6
MINER_PORT ?= 8001
MINER_ADDR := 0.0.0.0:$(MINER_PORT)

DEPLOY_TS := $(shell date +%Y%m%d_%H%M%S)
DEPLOY_LOG := $(LOG_DIR)/deploy_$(DEPLOY_TS).log
WALLET_DIR := $(LOG_DIR)/wallets

.PHONY: compile stop_miner deploy_miner download_log environment

compile: $(MINER_BIN) $(CLIENT_BIN) $(FAKEMINER_BIN)
	@echo "Binaries are ready in $(BIN_DIR)/"

$(MINER_BIN): $(shell find cmd/miner -name '*.go') $(shell find pkg -name '*.go')
	@$(MKDIR_P) $(BIN_DIR)
	@$(GO) build -o $@ ./cmd/miner

$(CLIENT_BIN): $(shell find cmd/client -name '*.go') $(shell find pkg -name '*.go')
	@$(MKDIR_P) $(BIN_DIR)
	@$(GO) build -o $@ ./cmd/client

$(FAKEMINER_BIN): $(shell find cmd/fakeminer -name '*.go') $(shell find pkg -name '*.go')
	@$(MKDIR_P) $(BIN_DIR)
	@$(GO) build -o $@ ./cmd/fakeminer

stop_miner:
	@if [ ! -f minerip.txt ]; then echo "minerip.txt missing"; exit 1; fi
	@echo "Stopping miners..."
	@while read -r ip; do \
		[ -z "$$ip" ] && continue; \
		echo " - $$ip"; \
		$(SSH) -n $(SSH_OPTS) root@$$ip "pkill -f $(REMOTE_DIR)/[m]iner || pkill -f '[m]iner' || true" >/dev/null 2>&1 || echo "   (skip) ssh failed"; \
	done < minerip.txt

deploy_miner:
	@if [ ! -f minerip.txt ]; then echo "minerip.txt missing"; exit 1; fi
	@if [ -z "$(COUNT)" ]; then echo "COUNT is required"; exit 1; fi
	@$(MAKE) stop_miner
	@$(MAKE) compile
	@$(MKDIR_P) $(LOG_DIR)
	@$(RM) $(WALLET_DIR)
	@$(MKDIR_P) $(WALLET_DIR)
	@echo "Deploying first $(COUNT) miners with difficulty $(DIFFICULTY)" | tee -a $(DEPLOY_LOG)
	@echo "Wallets will be generated under $(WALLET_DIR)/" | tee -a $(DEPLOY_LOG)
	@selected_ips=$$(awk 'NR<=$(COUNT)' minerip.txt); \
		echo "Selected miner IPs:" | tee -a $(DEPLOY_LOG); \
		echo "$$selected_ips" | sed 's/^/  - /' | tee -a $(DEPLOY_LOG); \
		echo "" >> $(DEPLOY_LOG); \
		echo "Peer port: $(MINER_PORT)" | tee -a $(DEPLOY_LOG); \
		echo "" >> $(DEPLOY_LOG); \
		echo "$$selected_ips" | while read -r ip; do \
			[ -z "$$ip" ] && continue; \
			wallet_file="$(WALLET_DIR)/wallet_$$ip.json"; \
			./$(CLIENT_BIN) wallet | tee "$$wallet_file" >/dev/null; \
			wallet_addr=$$($(PYTHON) -c 'import json,sys; print(json.load(open(sys.argv[1]))["address"])' "$$wallet_file"); \
			peers=$$(echo "$$selected_ips" | grep -v -x "$$ip" | sed "s/$$/:$(MINER_PORT)/" | paste -sd, -); \
			echo "[wallet] $$ip -> $$wallet_file" | tee -a $(DEPLOY_LOG); \
			echo "[peers] $$ip -> $$peers" | tee -a $(DEPLOY_LOG); \
			echo "[deploy] $$ip" | tee -a $(DEPLOY_LOG); \
			$(SSH) -n $(SSH_OPTS) root@$$ip "mkdir -p $(REMOTE_DIR)" >> $(DEPLOY_LOG) 2>&1 || { echo "  mkdir failed" | tee -a $(DEPLOY_LOG); continue; }; \
			$(SSH) -n $(SSH_OPTS) root@$$ip "pkill -f $(REMOTE_DIR)/[m]iner || pkill -f '[m]iner' || true; rm -f $(REMOTE_DIR)/miner" >> $(DEPLOY_LOG) 2>&1 || true; \
			$(SCP) $(SCP_OPTS) $(MINER_BIN) root@$$ip:$(REMOTE_DIR)/miner.new >> $(DEPLOY_LOG) 2>&1 < /dev/null || { echo "  copy failed" | tee -a $(DEPLOY_LOG); continue; }; \
			$(SSH) -n $(SSH_OPTS) root@$$ip "mv -f $(REMOTE_DIR)/miner.new $(REMOTE_DIR)/miner && chmod +x $(REMOTE_DIR)/miner" >> $(DEPLOY_LOG) 2>&1 || { echo "  install failed" | tee -a $(DEPLOY_LOG); continue; }; \
			$(SSH) -n $(SSH_OPTS) root@$$ip "nohup $(REMOTE_DIR)/miner -id $$wallet_addr -address $(MINER_ADDR) -peers '$$peers' -difficulty $(DIFFICULTY) > $(REMOTE_DIR)/miner.log 2>&1 &" >> $(DEPLOY_LOG) 2>&1 && \
			echo "  started" | tee -a $(DEPLOY_LOG); \
		done

download_log:
	@if [ ! -f minerip.txt ]; then echo "minerip.txt missing"; exit 1; fi
	@$(RM) $(DOWNLOAD_DIR)
	@$(MKDIR_P) $(DOWNLOAD_DIR)
	@echo "Downloading logs to $(DOWNLOAD_DIR)/"
	@awk 'NR<=$(COUNT)' minerip.txt | while read -r ip; do \
		[ -z "$$ip" ] && continue; \
		echo " - $$ip"; \
		$(SCP) $(SCP_OPTS) root@$$ip:$(REMOTE_DIR)/miner.log $(DOWNLOAD_DIR)/miner_$$ip.log >/dev/null 2>&1 || echo "   (skip) no log"; \
	done


environment:
	@echo "Setting up Node.js environment with nvm and pnpm..."
	@bash -lc 'set -euo pipefail; \
		apt update; \
		apt install -y curl; \
		if [ ! -d "$${HOME}/.nvm" ]; then \
			curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.3/install.sh | bash; \
		fi; \
		. "$${HOME}/.nvm/nvm.sh"; \
		nvm install 24; \
		node -v; \
		corepack enable pnpm; \
		pnpm -v; \
		echo "Environment setup complete."'