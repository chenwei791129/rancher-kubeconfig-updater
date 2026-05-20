## Summary

擴充 `install-script` capability 至 Windows：在 repo 根目錄新增 `install.ps1`，使 Windows 使用者能以一行 `irm ... | iex` 完成安裝；同時要求腳本自動寫入使用者 PATH（這是 Windows 與 Unix 必要的差異）。

## Motivation

目前 README 的 Windows 安裝段落仍是 4 步驟 PowerShell snippet（curl.exe 下載、ren 改名、move 搬到 System32），需要 admin 權限且體驗與 Linux / macOS 的單行 `curl ... | sh` 不對等。改為一行 `irm | iex` 後：

- 跨平台契約對齊：使用者在三平台都看到單行命令 + 相同的 `VERSION` / `INSTALL_DIR` 環境變數模型。
- 預設不需要 admin：與 Unix 端「預設裝到使用者目錄」的精神一致，把 system-wide 安裝改成顯式 opt-in。
- 自動 PATH 寫入：Windows 上 `~\.local\bin` 永遠不會被自動加入 PATH（不像 Linux 上有 `~/.profile` / pam_env 慣例），若不寫入 registry，使用者裝完仍無法執行二進位，等於沒裝。

`~\.local\bin`（即 `$env:USERPROFILE\.local\bin`）已是主流 CLI 工具在 Windows 的事實標準（uv、pipx、bun、deno、rustup 都使用 user-profile 下的 `bin` 目錄結構），cross-platform 一致是低成本選擇。

## Proposed Solution

1. 新增 `install.ps1`（PowerShell 5+ 相容）於 repo 根目錄，職責對齊 `install.sh`：
   - 開頭強制 TLS 1.2（給舊 Windows 10 / 預設 PS 5 環境）。
   - 偵測架構，allowlist 僅 `windows-amd64`；其他組合（如 windows-arm64）在任何網路呼叫之前 fail-fast，指向 Building from Source。
   - 守衛 `$env:USERPROFILE` 未設或為空。
   - `$env:VERSION` 預設 `latest`，HEAD redirect 解析實際 tag 供 confirmation line 顯示。
   - `$env:INSTALL_DIR` 預設 `$env:USERPROFILE\.local\bin`。預設值不存在時自動建立；建立失敗 → 退出非零並點名目錄。
   - 非預設 `INSTALL_DIR` 不存在 → fail-fast；存在但不可寫 → **不嘗試 UAC 自動提權**，退出非零並指引使用者用 admin PS 重跑。
   - 下載走 `Invoke-WebRequest` 至暫存路徑（`New-TemporaryFile` / `[System.IO.Path]::GetTempFileName()`），完成後 `Move-Item` 到目標路徑，失敗不留 partial file。
   - 若目標 `INSTALL_DIR` 不在使用者 PATH 中，透過 `[Environment]::SetEnvironmentVariable('PATH', ..., 'User')` 寫入，並印明確 notice（包含「需重啟 shell 才生效」說明）。
   - 訊息純 ASCII；以 `exit 0` / `exit 1` 對應 `$LASTEXITCODE`。
2. README 修改：
   - `Installation > Windows` 子段落改寫為單行 `irm ... | iex` + 環境變數表格 + system-wide 安裝範例（admin PS + `INSTALL_DIR="C:\Program Files\..."`）。
   - 不合併 Linux/macOS 與 Windows 段落（shell 語法差異大）。
   - `Building from Source` 段落範圍延伸到也支援 windows-arm64 等未發佈 Windows 平台。
3. `openspec/specs/install-script/spec.md` 修改：
   - `One-line installation entry point` requirement 增加 Windows scenario。
   - `Platform allowlist with fail-fast behavior` requirement allowlist 加入 `windows-amd64`，decision matrix 增加 windows 行（含 windows-arm64 fail-fast 行）。
   - `Version selection via VERSION environment variable` requirement 補充 PowerShell `$env:VERSION` 設定方式（純文字補充，不變語意）。
   - `Installation directory via INSTALL_DIR environment variable` requirement 新增「Windows 預設 `$env:USERPROFILE\.local\bin`」「非預設不可寫退出非零不提權」「PATH 自動寫入」三條條款，並補對應 scenarios。
   - `README installation section uses the script` requirement 增加 Windows scenario。

## Non-Goals

- 不在 release-please.yml 增加 windows-arm64 build matrix。
- 不簽署 install.ps1（無 code signing certificate）。
- 不整合 Chocolatey、Scoop、WinGet。
- 不支援 PowerShell 2.0 / Windows 7。
- 不嘗試 UAC 自動提權（避免跨 session boundary 的 elevation 複雜度）。
- 不偵測或修改系統 PATH（僅修改使用者 PATH）。
- 不自動修改現行 shell session 的 `$env:PATH`（registry 寫入只對新開的 session 生效，由使用者重啟）。
- 不變動 install.sh 的既有行為或測試。

## Alternatives Considered

- **預設 `$env:LOCALAPPDATA\Programs\rancher-kubeconfig-updater\`**（Windows-native）：被否決。違背跨平台 spec 契約一致原則；該路徑同樣不在 PATH，仍需自動寫入，且使用者習慣不如 `~\.local\bin` 統一（uv / pipx / bun / deno / rustup 都向 `~\.<name>\bin` 收斂）。
- **UAC 自動提權**：被否決。`Start-Process -Verb RunAs` 觸發提權後，與當前 session 跨進程，輸出與 exit code 回傳機制複雜，error reporting 不可靠；rustup 也選擇報錯讓使用者重跑。
- **沿用既有 4 步驟 README snippet**：被否決。違背一行安裝的初衷，且要求 System32 寫入需要 admin，使用體驗劣於現有方案。
- **新開 `windows-install-script` capability**：被否決。同樣的 env var 契約 + fail-fast 模型只是換 shell 實作，拆兩個 capability 會帶來 cross-capability 雙重維護成本。
- **修改現行 shell session 的 `$env:PATH`**：被否決。`irm | iex` 在 caller scope 執行，理論上可改，但會引入 user-visible side effect 不易移除；改 registry + 提示重啟 shell 是業界慣例（rustup / Bun / Deno）。

## Impact

- Affected specs: `install-script` capability（MODIFIED — 5 個現有 requirements 全部觸及；不新增 requirement）。
- Affected code:
  - New:
    - `install.ps1`
  - Modified:
    - `README.md`
  - Removed:
    - (none)
- Affected release flow: 無變動。沿用既有 `releases/latest/download/rancher-kubeconfig-updater-windows-amd64.exe` URL 慣例。
- Affected change ordering: 本 change 依賴 `add-install-script` 與 `change-default-install-dir` 已 archive（皆完成）；無 ordering 阻擋。
