# worklog

[English](README.md) | 繁體中文

**worklog 把你的 Git commit 整理成每天的工作日誌。**

[快速上手](#快速上手) ·
[安裝](#安裝) ·
[每天的用法](#每天的用法) ·
[設定](#設定) ·
[疑難排解](#疑難排解)

收工時執行 `worklog day`。它會整理你今天的 commit，先把要寫的日誌給你看，你回答 y 之後才存成 Markdown 檔。日誌很適合放進 Obsidian，而且只存在你自己的電腦上。

## 快速上手

```sh
cd ~/code/my-project
worklog setup          # 回答幾個問題，每題按 Enter 就是接受預設值

# ……照常工作、照常 commit……

worklog                # 看今天有幾個 commit、還有什麼沒做
worklog day            # 預覽今天的日誌，輸入 y 存檔
```

```text
$ worklog
  work   2026-10-27   2 commits   not written yet

Next: worklog day
```

## 功能

- **收工只要一個指令。** `worklog day` 會先預覽日誌，你輸入 `y` 才會寫入。
- **重跑也安全。** 同一天再跑，只會顯示有變動的行。你自己在日誌裡加的內容永遠不會被動到。
- **工作和私人分開。** 每個 profile 有自己的 repo、email、時區、語言和日誌資料夾。
- **AI 可有可無。** 可以用完全不需要 AI 的 Built-in，也可以讓 Claude Code 或 Codex 用你的語言寫日誌。
- **預設就保護隱私。** AI 只會看到整理過的工作清單，看不到程式碼、diff、檔案路徑或 email。你也可以額外指定要隱藏的字。
- **AI agent 能補上更多細節。** agent 可以記下每個改動的作法和原因，日誌會一起收錄。
- **Obsidian 友善的 Markdown。** 一天一個檔案，附 frontmatter 和標籤。

## 運作方式

1. **你照常 commit。** 在已加入的 repo 裡做的每個 commit 都會算進去，commit 的標題會成為日誌裡的一個項目。
2. **AI agent 可以順便補上細節（選用）。** 用 Claude Code 或 Codex 這類 agent commit 時，它可以記下你做了什麼、怎麼做、為什麼這樣做。worklog 把這些存成該 commit 的 Git note，日誌內容就會更完整。請看[記錄更多細節](#記錄更多細節)。
3. **你執行 `worklog day`。** 每個 profile 每天寫一份日誌，例如 `~/Documents/worklog/work/2026/2026-10/2026-10-27.md`。

profile 用來把生活的不同部分分開。常見的設定是一個 `work`、一個 `personal`，各自有自己的 repo、email、時區、語言和日誌資料夾，兩邊的日誌永遠不會混在一起。

## 安裝

### 環境需求

- macOS 或 Linux，Apple 晶片／arm64 或 Intel／amd64 都可以
- Git
- 選用：`claude` 或 `codex` CLI，想讓 AI 寫日誌時才需要
- Go 1.23 以上，只有從原始碼編譯時才需要

### 從發布版本安裝

每個發布版本都為各平台準備好可以直接執行的檔案，不需要安裝 Go。

1. 下載你電腦對應的檔案和檢查碼檔。這個 repo 是私人的，所以要用有讀取權限的帳號搭配 GitHub CLI 下載：

   ```sh
   version=v0.1.0
   platform=darwin_arm64   # 或 darwin_amd64、linux_amd64、linux_arm64
   gh release download "$version" --repo Vickyciou/worklog \
     --pattern "worklog_${version}_${platform}" --pattern SHA256SUMS
   ```

2. 確認下載的檔案沒有損壞。你那個檔案的那一行結尾必須是 `OK`：

   ```sh
   shasum -a 256 -c SHA256SUMS --ignore-missing
   ```

3. 以 `worklog` 這個名字安裝到 `PATH` 裡的資料夾：

   ```sh
   install -m 755 "worklog_${version}_${platform}" ~/bin/worklog
   worklog version
   ```

   如果 macOS 的檔案是用瀏覽器下載的，還要再執行 `xattr -d com.apple.quarantine ~/bin/worklog`。

要升級時，用新版本重做一次這幾個步驟。你的設定、note 和日誌都存在別的地方，不會受影響。

### 從原始碼編譯

需要 Go 1.23 以上：

```sh
go build -o ./bin/worklog ./cmd/worklog
```

## 第一次設定

到一個你平常工作的 Git repo 裡執行：

```sh
cd ~/code/my-project
worklog setup
```

setup 會問幾個問題，大部分都有預設答案，按 Enter 就能接受：

- **Profile。** Work、Personal，或你自己取的名字。
- **Repository。** 你目前所在的 repo。
- **Journal language（日誌語言）。** 預設是系統語言。它決定日誌的標題語言，以及 AI 摘要使用的語言。
- **Journal folder（日誌資料夾）。** `~/Documents/worklog/<profile>`。
- **Daily summary（每日摘要）。** 日誌要怎麼寫出來。請看[選擇每日摘要方式](#選擇每日摘要方式)。

email 會從 Git 讀取，時區從系統讀取，只有讀不到時才會問你。存檔前會把所有設定再列一次給你確認。

## 每天的用法

### 看目前的狀態

```sh
worklog
```

直接執行 `worklog` 會顯示每個 profile 今天的 commit 數和日誌狀態，最後一行告訴你下一步該做什麼。它只會讀取資料，隨時執行都安全。

### 寫今天的日誌

```sh
worklog day
```

每個今天有 commit 的 profile，都會先顯示一行概覽，再顯示完整的日誌。輸入 `y` 存檔；輸入其他任何內容都不會寫入，之後可以再執行一次 `worklog day`。

profile 語言是 `zh-TW` 時，日誌長這樣：

```markdown
---
date: 2026-10-27
type: worklog-daily
tech: [Go]
tags: [工作日誌]
---

<!-- worklog:managed:start source-digest=sha256:… -->
# 工作日誌 - 2026-10-27

## 今日重點

- 修正 token 更新的競態問題
- 新增登入重試與退避機制

## 完成事項

### 工作成果

- 修正 token 更新的競態問題
- 新增登入重試與退避機制
  - 作法：以指數退避重試三次

## 技術決策

- 重試上限為三次（原因：避免帳號被鎖定）

## 後續行動

- 加上重試的監控指標

<!-- worklog:managed:end -->
```

profile 語言以 `zh` 開頭時（例如 `zh-TW`），標題是繁體中文，其他語言則是英文。`tags` 那一行不論哪種語言都一樣，所以你在 Obsidian 的搜尋不會因此失效。

### 晚點又 commit 了？再跑一次就好

同一天再執行 `worklog day`，只會顯示有變動的地方，`+` 是新增的行，`-` 是刪除的行，並在更新前先問你：

```text
  - 修正 token 更新的競態問題
  - 新增登入重試與退避機制
+ - 加上重試的監控指標
```

如果沒有任何變動，它會告訴你日誌已經是最新的。兩行 `worklog:managed` 標記以外的內容都屬於你，worklog 永遠不會碰。

### 補寫以前的日子，或重新產生

```sh
worklog day 2026-10-20            # 寫入或更新以前某一天
worklog day 2026-10-20 --refresh  # 即使 commit 沒變，也重新產生
```

改了 profile 的語言或每日摘要方式之後，可以用 `--refresh` 把既有的日誌用新設定重寫一次。

## 選擇每日摘要方式

| 選項 | 做什麼 | 需要什麼 |
| --- | --- | --- |
| **Built-in** | 照你記錄的內容列出工作，不改寫。 | 什麼都不需要，不用 AI 也不用網路。 |
| **Claude Code** | 請 Claude Code 用 profile 的語言寫日誌。 | 已登入的 `claude` CLI。 |
| **Codex** | 請 Codex 用 profile 的語言寫日誌。 | 已登入的 `codex` CLI。 |

worklog 直接沿用 CLI 已經登入的狀態，從不儲存或檢查你的帳號憑證。如果 AI 無法使用、執行失敗或回傳無法使用的內容，worklog 這一次會改用 Built-in，並告訴你原因。

AI 只會收到整理過的工作清單，看不到程式碼、diff、commit hash、檔案路徑或 email。想看它實際收到什麼，執行 `worklog inspect`。

## 設定

### 修改設定

| 我想要…… | 執行 |
| --- | --- |
| 加入我目前所在的 repo | `worklog repo add` |
| 把 repo 加到指定的 profile | `worklog repo add ~/code/api --profile work` |
| 修改語言、時區、資料夾或摘要方式 | `worklog settings` |
| 查看某個 profile 的設定 | `worklog profile show work` |
| 再新增一個 profile，例如 personal | `worklog profile add` |
| 為 profile 加入另一個 email | `worklog author add work me@company.example` |
| 列出 profile 裡的 repo | `worklog repo list work` |

只有一個 profile 時，worklog 會自動使用它。有好幾個時，它會問你要用哪一個，或者你也可以用 `--profile` 指定。

一個 repo 只能屬於一個 profile。更換日誌資料夾只影響之後的新日誌，既有的日誌會留在原處。

### 隱藏不該出現的字

如果有些字不該出現在日誌裡，也不該送給 AI，例如客戶名稱或內部代號，可以替 profile 指定：

```sh
worklog profile set work --deny-literal client-codename --deny-literal internal-host
```

worklog 會在做摘要之前把它們替換掉，也會自動移除常見的密鑰、token、路徑和 email。`profile show` 只會顯示隱藏了幾個字，不會顯示字本身。

## 記錄更多細節

commit 標題只說了改了什麼。note 還可以補上怎麼做、為什麼這樣決定、接下來要做什麼。agent 和腳本用 `worklog capture` 記錄 note：

```sh
echo '{"version":1,"outcome":"新增登入重試與退避機制",
       "decisions":[{"decision":"重試上限為三次","reason":"避免帳號被鎖定"}],
       "next_steps":["加上重試的監控指標"]}' \
  | worklog capture . HEAD
```

`worklog capture --help` 會列出所有欄位。同一筆 note 記錄兩次不會有問題；但如果某個 commit 已經有 note，內容不同的新 note 會被拒絕，所以不會有東西被覆蓋。`gen-commit` skill 每次 commit 後都會自動做這件事。

## 疑難排解

```sh
worklog doctor     # 找出設定問題，並告訴你每個問題怎麼修
worklog inspect    # 顯示 worklog 今天收集到的內容
worklog inspect work 2026-10-20
```

`doctor` 不會印出你的 commit、路徑或隱藏的字，也不會呼叫 AI。每個指令都有 `--help`；`day`、`inspect`、`doctor` 另外還有 `--details`，列出完整的隱私與檔案安全規則。

**我的日誌少了某個 commit。** worklog 只會計入同時符合以下條件的 commit：

- commit 在已加入該 profile 的 repo 裡。用 `worklog repo list` 確認。
- commit 的作者 email 屬於該 profile。用 `worklog author list <profile>` 確認。
- 以該 profile 的時區來看，commit 時間落在那一天。
- 不是 merge commit。

## 資料存放位置

| 項目 | 位置 | 修改方式 |
| --- | --- | --- |
| 日誌 | 各 profile 的日誌資料夾 | `worklog settings` |
| agent 記錄的 note | 你的 repo 裡，Git ref `refs/notes/worklog` 底下 | |
| 設定 | 使用者設定資料夾，macOS 上例如 `~/Library/Application Support/worklog` | `WORKLOG_CONFIG_HOME` |
| 暫存檔 | `~/.local/state/worklog` | `WORKLOG_STATE_HOME` |
| AI 時間上限 | 每份日誌兩分鐘 | `WORKLOG_SUMMARIZER_TIMEOUT`，例如 `30s` |

worklog 的原始碼資料夾裡不會存放任何個人資料。

## 什麼情況不適合用 worklog

- **你的工作不在 Git 裡。** worklog 只讀 Git commit，會議、工單和聊天紀錄除非被 commit，否則不會出現。
- **你需要在多台電腦之間同步 note。** note 只存在本機的 repo 裡，worklog 不會替你 push 或 fetch `refs/notes/worklog`。
- **你用 Windows。** 目前沒有 Windows 版本。從原始碼編譯也許可以用，但沒有測試過。
- **你想知道以前工作的原因。** 從沒被記錄下來的決策，worklog 沒辦法憑空補上，較早的 commit 只會以標題出現。

## 進階：使用你自己的摘要指令

除了 Built-in、Claude Code、Codex，profile 也可以使用你自己寫的任何指令：

```sh
worklog profile add local --timezone Asia/Taipei --language zh-TW \
  --journal-destination ~/Documents/worklog/local \
  --summarizer-command '/path/to/summary-adapter --format json'
```

這個指令會從 stdin 收到和 `worklog inspect` 印出的相同 JSON，並必須在 stdout 印出一個摘要 JSON 物件，包含 `version`、`focus`、`completed`、`decisions`、`next_steps` 和 `technologies`。檔案路徑和 Markdown 仍然由 worklog 自己決定與產生。指令預設有兩分鐘的時間上限，可以用 `WORKLOG_SUMMARIZER_TIMEOUT` 調整，例如 `30s`。帳號憑證請放在指令的環境變數或系統鑰匙圈裡，不要寫在指令本身。完整規則請看 `worklog day --details`。

## 給開發者

執行測試：

```sh
go test ./...
```

驗收測試會先編譯 CLI，再把它當成獨立程式執行，使用暫時的 home、設定、repo 和日誌資料夾，不會碰到你的真實資料。

從 `main` 推送版本 tag 就能發布新版本：

```sh
git tag v0.1.0
git push origin v0.1.0
```

接著 GitHub Actions 會在 Linux 和 macOS 上執行測試，用 `scripts/release.sh` 建出四個平台的版本，再用 `scripts/smoke-test.sh` 在各自的平台、沒有 Go 的乾淨環境中逐一試跑。所有步驟都通過才會發布。想在本機試建發布版本：

```sh
scripts/release.sh v0.0.0
scripts/smoke-test.sh dist/worklog_v0.0.0_darwin_arm64 v0.0.0
```

## 授權

worklog 以 MIT 授權發布，詳見 [LICENSE](LICENSE)。
