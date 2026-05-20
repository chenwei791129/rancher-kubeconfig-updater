## 1. 實作 install.sh

- [x] 1.1 建立 `install.sh`（POSIX `sh` shebang、`set -eu`、info/error 訊息 helper），構成 one-line installation entry point 腳本。Verified by `shellcheck install.sh` 退出碼 0。
- [x] 1.2 實作 platform allowlist with fail-fast behavior：正規化 `uname -s` / `uname -m`，allowlist `linux-amd64` 與 `darwin-arm64`，其他組合在 download 之前 fail-fast 並印出含平台字串與 Building from Source 指引的錯誤訊息。Verified by 透過環境變數注入（或內建 `_OS_OVERRIDE` / `_ARCH_OVERRIDE` hook）模擬 `linux-arm64`、`darwin-amd64`、`FreeBSD-amd64`，每組均回傳非零、stderr 含對應平台字串、且整個過程未發出任何網路請求（可用 `strace`/`dtruss` 或斷網執行確認）。
- [x] 1.3 實作 version selection via VERSION environment variable：未設或為 `latest` 時走 `releases/latest/download/...`，否則走 `releases/download/${VERSION}/...`；HTTP 4xx/5xx 時非零退出、印出失敗 URL、且不於目標路徑留下 partial file。Verified by 將 `VERSION` 設為不存在的 tag（例如 `v0.0.0-nonexistent`）執行腳本，斷言退出碼非零、stderr 含完整下載 URL 與 HTTP 錯誤碼、`INSTALL_DIR` 內無新增檔案。
- [x] 1.4 實作 installation directory via INSTALL_DIR environment variable：預設 `/usr/local/bin`、非預設值不可觸發 sudo、最終檔名固定 `rancher-kubeconfig-updater`。Verified by (a) `INSTALL_DIR=$(mktemp -d) sh install.sh` 完成後該目錄出現可執行的 `rancher-kubeconfig-updater` 且過程未呼叫 sudo；(b) `INSTALL_DIR=/no-such-path sh install.sh` 退出非零、錯誤訊息含 `/no-such-path`、且未提權。

## 2. 更新 README

- [x] 2.1 改寫 `Installation > Linux / macOS` 子段落使 README installation section uses the script：呈現單行 `curl -fsSL https://raw.githubusercontent.com/chenwei791129/rancher-kubeconfig-updater/main/install.sh | sh`、列出 `VERSION` / `INSTALL_DIR` 變數用法、明示支援平台僅 `linux-amd64` 與 `darwin-arm64`。Verified by 閱讀更新後的 `README.md`，確認命令字串與 spec 一致、且 Windows 子段落與本 change 之前的內容 byte 級相同（`git diff README.md` 在 Windows 區塊內無差異）。
- [x] 2.2 新增 `Installation > Building from Source` 子段落，內含 `git clone` 與 `go build .` 指令，供 `linux-arm64` / `darwin-amd64` 等未發佈平台使用者使用。Verified by 在乾淨工作目錄依照新段落指令依序執行，產生可運行的 `rancher-kubeconfig-updater` binary 且 `./rancher-kubeconfig-updater --help` 退出碼為 0。

## 3. 端到端驗證

- [x] 3.1 在 `darwin-arm64` 主機以 `VERSION=latest`、`INSTALL_DIR=$(mktemp -d)` 執行完整 `install.sh`，確認下載、置放、執行權限三步均成功，並能呼叫 `./rancher-kubeconfig-updater --help` 取得 usage 輸出。Verified by 手動執行並截錄安裝路徑與 help 輸出，確認與 spec 中 Default install on supported platform scenario 描述一致。
