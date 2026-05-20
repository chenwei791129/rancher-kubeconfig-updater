## Why

目前 README 的 Linux / macOS 安裝步驟是 22 行的 shell snippet（OS / arch 偵測、curl、rename、chmod、sudo mv），複製貼上時容易出錯，且現有 snippet 對未發佈的平台（如 linux-arm64、darwin-amd64）會悄悄產生 404 下載連結而無錯誤提示。把整段壓縮成單行 `curl ... | sh`，並用環境變數控制版本與安裝目錄，能大幅降低新使用者的上手摩擦。

## What Changes

- 新增 repo 根目錄的 `install.sh`，POSIX `sh` 相容，負責偵測 OS / arch、下載對應 binary、`chmod +x`、搬到 `INSTALL_DIR`。
- 支援 `VERSION` 環境變數（預設 `latest`）讓使用者釘到特定 release tag。
- 支援 `INSTALL_DIR` 環境變數（預設 `/usr/local/bin`）；非預設值時不觸發 sudo。
- 僅 allowlist `linux-amd64` 與 `darwin-arm64`（與 `release-please.yml` 的 build matrix 一致）；其他平台組合一律 fail-fast，並在錯誤訊息中指向 build-from-source 段落。
- README 的 `Installation > Linux / macOS` 區段改為單行 `curl -fsSL .../install.sh | sh`，並補一個簡短的 `Building from Source` 區段給未發佈平台的使用者。
- Windows 安裝段落不動。

## Non-Goals

- 不擴充 `release-please.yml` 的 build matrix 補 `linux-arm64` / `darwin-amd64`。
- 不產生 / 不驗證 SHA256 checksums（release 流程目前未產 checksum 檔）。
- 不引入 GoReleaser；未來討論備忘已記於 `.local/goreleaser-future-plan.md`。
- 不發佈 Homebrew tap、Scoop bucket、AUR 等套件管理器集成。
- 不改變 Windows 安裝流程。
- 不在 `install.sh` 內做下載內容的可執行驗證（例如 `--version` 自我測試），交給使用者執行。

## Capabilities

### New Capabilities

- `install-script`: 提供 POSIX shell 安裝腳本，封裝 release binary 的下載 / 平台偵測 / 安裝路徑處理，使 Linux 與 macOS 使用者能透過單行 `curl ... | sh` 完成安裝。

### Modified Capabilities

(none)

## Impact

- Affected specs: 新增 `install-script` capability spec。
- Affected code:
  - New:
    - `install.sh`
  - Modified:
    - `README.md`（Installation > Linux / macOS 段落改寫；補 Building from Source 段落）
  - Removed:
    - (none)
- Affected release flow: 不變動 `.github/workflows/release-please.yml`；`install.sh` 依賴既有的 `releases/latest/download/rancher-kubeconfig-updater-${OS}-${ARCH}` URL 慣例。
- Affected docs: `CHANGELOG.md` 由 release-please 自動產生，本 change 不手動編輯。
