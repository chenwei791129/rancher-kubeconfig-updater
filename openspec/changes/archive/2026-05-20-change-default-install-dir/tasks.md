## 1. 修改 install.sh

- [x] 1.1 把 `DEFAULT_INSTALL_DIR` 由 `/usr/local/bin` 改為 `$HOME/.local/bin`，並在 `INSTALL_DIR` 未顯式指定（resolves to default）且目錄不存在時執行單次 `mkdir -p "${INSTALL_DIR}"`，建立失敗時退出非零並以錯誤訊息點名該目錄。Verified by (a) `rm -rf "$HOME/.local/bin"` 後執行 `sh install.sh`，斷言目錄被建立、binary 落在 `$HOME/.local/bin/rancher-kubeconfig-updater`、過程無 sudo 呼叫；(b) 把 `HOME` 暫時指到唯讀路徑（例如 `HOME=/usr/sbin sh install.sh`）再執行，斷言退出非零、錯誤訊息提到該無法建立的目錄、且無 `sudo` 呼叫。
- [x] 1.2 重整 INSTALL_DIR validation 邏輯使其符合修改後的 Installation directory via INSTALL_DIR environment variable requirement：預設值走 auto-mkdir + 永不 sudo；非預設值不存在 → fail-fast；非預設值存在但不可寫 → 進入 sudo 分支並印 notice。Verified by (a) `INSTALL_DIR=/opt/bin sh install.sh`（不存在）→ 退出非零、無 mkdir、無 sudo、stderr 含 `/opt/bin`；(b) `INSTALL_DIR=/usr/local/bin sh install.sh`（存在且不可寫）→ stderr 出現 `Elevation required` notice 且呼叫 sudo（透過 sudo-shim 捕捉）。
- [x] 1.3 加入 `$HOME` 空值守衛：在使用 `${HOME}` 之前若 `HOME` 未設或為空，輸出明確錯誤並在任何網路呼叫之前退出非零，符合新 spec 中 HOME is unset scenario。Verified by `env -i PATH=$PATH sh install.sh` 執行，斷言退出非零、stderr 含 `HOME` 字樣、且未出現 `Downloading` info 訊息（確保未發網路請求）。
- [x] 1.4 保留現有不在本 change 範圍的行為（platform allowlist、`_OS_OVERRIDE` / `_ARCH_OVERRIDE` hooks、`command -v curl` preflight、trap `EXIT INT TERM`、`VERSION` 解析與 display_version、trailing-slash normalization）。Verified by `shellcheck install.sh` 退出碼 0，且 `add-install-script` change 中既有的所有 task 驗證指令（platform fail-fast 三組、`VERSION=v0.0.0-nonexistent`、`INSTALL_DIR=/no-such-path`）重跑後行為一致。

## 2. 更新 README

- [x] 2.1 改寫 `Installation > Linux / macOS` 子段落使 README installation section uses the script 在新預設下仍精準：主要範例維持單行 `curl ... | sh`，移除已不必要的 `mkdir -p ~/.local/bin` 預備步驟；環境變數表格更新 `INSTALL_DIR` default 為 `$HOME/.local/bin` 並標明「預設值會自動建立；非預設值必須事先存在」。Verified by 閱讀更新後的 `README.md`，確認預設範例不含 mkdir、變數表格 default 欄為 `$HOME/.local/bin`、且 Windows 子段落與本 change 之前的內容 byte 級相同（`git diff` 在 Windows 區塊內無差異）。
- [x] 2.2 改寫既有的「sudo + piped stdin」NOTE callout：把焦點從「預設行為的限制」改為「想做 system-wide 安裝時的指引」，提供 `INSTALL_DIR=/usr/local/bin` 顯式 opt-in 範例與「先 `curl -o install.sh` 再執行以便輸入密碼」的替代路徑，符合 Explicit /usr/local/bin requires elevation scenario。Verified by 閱讀更新後的 `README.md`，確認 NOTE 標題或前言提到 system-wide 安裝、且範例命令顯式設定 `INSTALL_DIR=/usr/local/bin`。

## 3. 端到端驗證

- [x] 3.1 在 `darwin-arm64` 主機以乾淨環境（先 `rm -rf "$HOME/.local/bin"`）執行 `./install.sh`，確認目錄被建立、binary 落地、`$HOME/.local/bin/rancher-kubeconfig-updater --help` 退出碼 0、過程無 sudo prompt — 同時驗證更新後的 One-line installation entry point 與 Installation directory via INSTALL_DIR environment variable requirements 的對應 Default install on supported platform / Install into default directory that does not exist scenarios。Verified by 手動執行並截錄輸出，比對兩個 scenario 的 THEN 條款。
- [x] 3.2 在同一主機以 `INSTALL_DIR=/usr/local/bin sh install.sh` 執行 Explicit /usr/local/bin requires elevation scenario：斷言 stderr 出現 `Elevation required` notice 且觸發 sudo 呼叫（可用 PATH 注入 sudo-shim 捕捉）。Verified by 手動執行並比對 spec scenario 的 THEN 條款。
