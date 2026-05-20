## Summary

把 `install.sh` 的 `DEFAULT_INSTALL_DIR` 從 `/usr/local/bin` 改為 `$HOME/.local/bin`，並讓預設目錄在不存在時自動建立；sudo 路徑從預設行為退化為「使用者顯式指定 `/usr/local/bin`」才觸發的 opt-in。

## Motivation

目前預設 `/usr/local/bin` 在 macOS 上是 root 擁有的系統目錄，寫入需要 sudo；當使用者透過 `curl ... | sh` 安裝時，stdin 是 pipe 而非 tty，`sudo` 幾乎必然失敗（`sudo: a password is required`），導致開箱即失敗的體驗。已透過 README NOTE 描述此限制，但這只是說明問題，沒有解決問題。

`~/.local/bin` 在主流系統都是合宜的使用者級 CLI 安裝路徑：

- **Linux**：XDG Base Directory 衍生規範與 systemd file-hierarchy(7) 所推薦的 user binaries 路徑；Fedora、Ubuntu 20.04+、Arch、Debian 12+ 在 `~/.profile` 或 pam_env 中會自動把它加入 `PATH`（若該目錄存在）。
- **macOS**：系統本身不會自動加 `PATH`，但 `uv tool install`、`rustup`、`pipx`、`pip --user` 等主流 CLI 工具預設都裝這裡，已是事實標準；腳本既有的「`INSTALL_DIR` 不在 `PATH`」warn 已能提示使用者補設定。

切換預設能讓多數使用者**零 sudo、跨平台一致**地完成安裝，同時保留想做 system-wide 安裝者透過 `INSTALL_DIR=/usr/local/bin`（或其他系統路徑）opt-in 的退路。

## Proposed Solution

1. `install.sh` 修改：
   - `DEFAULT_INSTALL_DIR` 由 `/usr/local/bin` 改為 `$HOME/.local/bin`。
   - 預設目錄不存在時自動 `mkdir -p` 一次（僅當 `INSTALL_DIR` 等於預設值時），這樣使用者第一次裝不必預先建立。
   - 非預設 `INSTALL_DIR` 維持現行 fail-fast 行為（不存在或不可寫 → 退出，無 sudo）。
   - sudo 分支邏輯維持但只在使用者顯式設 `INSTALL_DIR=/usr/local/bin`（或任一系統目錄）且該目錄不可寫時觸發；不可從預設值走到 sudo path。
   - 確認 confirmation message 仍正確報告 `display_version` 與 `target`。
2. `README.md` 修改：
   - 主要範例不再需要 `mkdir -p`；改為純 `curl ... | sh` 一行。
   - 環境變數表格更新 `INSTALL_DIR` 的 default 為 `$HOME/.local/bin`，並新增「預設值會自動建立；非預設值必須事先存在」說明。
   - 既有的「sudo + piped stdin」NOTE 改寫為：「如果想做 system-wide 安裝，可顯式設 `INSTALL_DIR=/usr/local/bin`；該路徑寫入需 sudo，建議先 `curl -o install.sh` 再執行以便輸入密碼」。
3. `openspec/specs/install-script/spec.md` 修改：
   - `Installation directory via INSTALL_DIR environment variable` requirement 改寫，預設由 `/usr/local/bin` 改為 `$HOME/.local/bin`，新增「預設目錄不存在時 SHALL 自動建立」條款。
   - Scenarios 對應調整：刪除「Default directory requires elevation」（不再是預設行為），改為新的「Install with default into auto-created home directory」與「Explicit /usr/local/bin requires elevation」兩個 scenarios。

## Non-Goals

- 不改 Windows 安裝段落。
- 不改 platform allowlist 或 release matrix。
- 不改 `VERSION` 環境變數的語意或解析邏輯。
- 不引入第二層 fallback（例如 `~/.local/bin` 不可建立時改試 `~/bin` 或 `/opt/...`）；只有單一預設。
- 不偵測 Homebrew 並改用 `/opt/homebrew/bin` 之類的平台特化 fallback。
- 不自動把 `~/.local/bin` 加進使用者的 shell rc；既有的 PATH warn 已足以提示。
- 不變動 `_OS_OVERRIDE` / `_ARCH_OVERRIDE` 測試 hook 行為。
- 不變動既有 trap / `command -v curl` preflight / curl flags 等 install.sh 內部結構。

## Alternatives Considered

- **macOS-only 改預設、Linux 維持 `/usr/local/bin`**：被否決。會讓腳本含平台分支邏輯，且 Linux 上 `/usr/local/bin` 一樣常需要 sudo（除非使用者改過權限），同樣的可用性問題。
- **改預設為 `/opt/homebrew/bin`（macOS Apple Silicon）/ `/usr/local/bin`（Intel Mac）/ `/usr/local/bin`（Linux）**：被否決。需偵測架構與 Homebrew 是否存在，邏輯複雜且把假設綁到 Homebrew，違反 install.sh 的「最小依賴」設計目標。
- **保留 `/usr/local/bin` 預設，靠 README 教育使用者改 `INSTALL_DIR`**：被否決。預設值的功能就是「不設定也能用」，若預設值幾乎必失敗，等於沒有預設。
- **完全移除 sudo 分支**：被否決。仍有 system-wide 安裝的合理用例（共享開發機、容器內 root user 安裝），保留 opt-in 路徑成本低。

## Impact

- Affected specs: `install-script` capability（MODIFIED requirement）。
- Affected code:
  - Modified:
    - `install.sh`
    - `README.md`
  - New:
    - (none)
  - Removed:
    - (none)
- Affected change ordering: 本 change 與 `add-install-script` 修改同一 capability。`add-install-script` 目前 tasks 已完成但尚未 archive；建議先 archive `add-install-script`（讓 `install-script` capability 進入 `openspec/specs/`），再 apply 本 change，避免 delta-on-delta 的分析疑慮。若需求變動，亦可把本 change 視為對 `add-install-script` 的補丁並重新 ingest，但首選為先後 archive。
