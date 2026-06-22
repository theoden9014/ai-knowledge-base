# Skill Directory Unit - Requirements Specification

## 1. Requirement Summary

`knit` の skill エントリを「ディレクトリ単位」として扱えるよう拡張する。manifest スキーマの説明は既に「skill は per-skill ディレクトリで sibling assets (scripts/, references/, ...) を伴える」と宣言しているが、実装は manifest path・loader・renderer すべてが SKILL.md という単一ファイルしか扱えない。設計上の意図と実装を一致させ、skill ディレクトリ配下の任意の補助ファイルを skill とともに配置・列挙・削除できるようにする。

agent / rule / prompt は単一ファイル単位のまま据え置く。

## 2. Functional Requirements

### F1. Manifest 表現

- skill entry の `path` は「ソースリポジトリ内の skill ディレクトリ（pack 相対）」を指す。
- 正規形は末尾スラッシュ無しの相対ディレクトリパス（例: `path: skills/orchestrator`）。末尾スラッシュ付きの表現は受け付けない。
- agent / rule / prompt の `path` は従来どおり単一の `.md` ファイル指定。
- skill の場合、`path` で指したディレクトリ直下に固定名 `SKILL.md` が存在することが暗黙の前提（解決規則は F2）。

### F2. ソース解釈（Loader 側）

- skill エントリの `path` で指されたディレクトリは「skill ルート」と呼ぶ。
- skill ルート直下の固定名 `SKILL.md` が skill の本体ファイル。これは従来どおり frontmatter + body として解釈され、Entry の `Body` に入る。
- skill ルートの同一サブツリー配下にある通常ファイル（ディレクトリでもシンボリックリンクでもない）のうち、本体ファイルである `SKILL.md` を除く全ファイルが「sibling 集合」を構成する。
- sibling 集合の各要素は次の 2 値を持つ: ① skill ルートからの相対パス（forward slash 区切り、`..` を含まない）、② 中身の不透明な byte 列。
- sibling は frontmatter 解析や schema 検証の対象としない。
- **neutral 表現の契約**: skill Entry は本体 `Body` に加えて sibling 集合を保持する。sibling 集合は Entry から取得可能な集合として扱われ、Builder 以降の処理は sibling を Entry の一部として参照する（具体的な保持形態は設計判断に委ねる）。
- agent / rule / prompt の Entry は従来どおり sibling 集合を持たない（常に空とみなされる）。

### F3. ビルド（Builder / Renderer 側）

- skill エントリは、SKILL.md を起点とする 1 件の主 Artifact と、sibling 1 ファイルにつき 1 件の sibling Artifact を生成する。
- **主 Artifact と全 sibling Artifact は同一の `SourceEntryIDs = [<pack>.skill.<name>]` を持つ**。これは F6（pack 指定 uninstall）が `Provenance.BelongsToPack` に依拠して sibling まで一括削除するための契約。
- 主 Artifact / sibling Artifact の集合は集合として等価とみなされる（順序保証なし）。
- 配置先パスは target の「skills ディレクトリ規約」配下に、ソース側ディレクトリ構造をそのまま再現したものとする。`SKILL.md` → `skills/<name>/SKILL.md`、`scripts/foo.sh` → `skills/<name>/scripts/foo.sh`、など。
- target ごとの「skills 規約」は既存のものを踏襲する（claude / codex / gemini 各 path_policy）。
- sibling Artifact の `Mode` はゼロ値（= Installer/ArtifactStore のレギュラーファイル既定）に統一する。ソース側の実行ビット等は持ち越さない。
- skill エントリの enabled 解決結果（`default_tools` または `tools.<target>.enabled`）に従い、enabled でない target では skill 本体・sibling のいずれも Artifact を生成しない。

### F4. 配置（Installer 側）

- 配置の粒度は従来どおり Artifact 1件 = Installation 1件。
- skill とその sibling は「複数の Installation の集合」として配置される。
- 集合内のいずれかが既に存在する場合の挙動は、現行の Installer のエラー優先順位を踏襲する（label が既にあれば `ErrAlreadyInstalled`、未管理ファイルが既にあれば `ErrUnmanagedArtifactExists`）。
- 集合のうち順に install され、途中で失敗した場合は失敗時点までに書き込まれた Installation は残る（既存の install command の semantics と同じ）。集合全体のロールバックは非ゴール。
- 失敗時のエラーには、失敗した Artifact のパス（ソース上の skill 相対 path もしくは生成先 path）が含まれ、どの sibling で失敗したかが識別できる。

### F5. 列挙（Lister 側）

- Lister は従来どおり Installation 単位で列挙する。
- skill とその sibling は別々の Installation として現れる。
- `Provenance.BelongsToPack` は SKILL.md / sibling の双方に対して同じ `<pack>.skill.<name>` を返す（同一の `SourceEntryIDs` を持つため）。

### F6. 削除（Uninstaller 側）

- パック名指定の `knit uninstall` は、`Provenance.BelongsToPack(packName)` で該当するすべての Installation を削除する。SKILL.md とその sibling はこの判定ですべて拾われる（F3 の `SourceEntryIDs` 契約に依拠）。
- 削除順序は Lister の返却順序に従う（既存仕様どおり）。
- 部分失敗時は、失敗時点までに削除された Installation は残らない（既存の uninstall semantics と同じ）。
- 失敗時のエラーには、失敗した Installation の ID が含まれ、どの sibling で失敗したかが識別できる。

### F7. 再配置（Reinstaller 側）

- 既存の reinstaller / refresh の挙動が、sibling の増減を含む skill に対しても破綻しないこと（最低保証）。
- sibling 増減時のあるべき挙動の規範化は本要件のスコープ外（R7 を参照）。

### F8. 検証（Validator 側）

- manifest スキーマは `id` の kind 部分（`skill` / `agent` / `rule` / `prompt`）と `path` 形状（ディレクトリ / 単一 .md ファイル）の整合を強制する。
  - `id.kind = skill` の場合: `path` は `skills/<kebab-case>` 形式のディレクトリ参照（末尾スラッシュ無し、`SKILL.md` を含まない）。
  - `id.kind = agent` の場合: `path` は `agents/<kebab-case>.md`。
  - `id.kind = rule` の場合: `path` は `rules/<kebab-case>.md`。
  - `id.kind = prompt` の場合: `path` は `prompts/<kebab-case>.md`。
- いずれの path も pack 相対で、`..` を含まず、絶対パスでない。

## 3. Non-Functional Requirements

### N1. 後方互換性

- 既存の `knowledge/structure-behavior-design/manifest.yaml` は本リリースで新形式に書き換える。
- 旧形式 `path: skills/<name>/SKILL.md` は **受け入れない**（スキーマ違反としてエラー）。本リポジトリ内に新形式へ書き換える対象は同 manifest 1 件のみ。
- 既に inventory に置かれた旧形式の Installation の自動移行は行わない（ユーザーが必要に応じて `knit uninstall` → `knit install` を実行する想定）。マイグレーションコマンドは提供しない。

### N5. neutral 識別子の不変

- skill の neutral 識別子 `<pack>.skill.<name>` は本変更でも不変。agent.uses_skills が参照するキーは影響を受けない。

### N2. 性能

- 1 skill あたりの sibling ファイル数は 10 〜 100 オーダーを想定。ファイル合計サイズは 数 MB オーダー。
- すべての sibling 内容は in-memory で扱う。ストリーミングは今回考慮しない。

### N3. プラットフォーム

- skill ディレクトリ走査時のパス区切りはソース側 `fs.FS` のセマンティクス（forward slash）で統一する。target 配置時の OS 固有区切りは既存 path policy 層の責務。

### N4. 可観測性

- ログ出力・エラーメッセージは「どの skill のどの sibling か」が分かる粒度で出す。

## 4. Inputs and Outputs

### 入力

- `manifest.yaml`: skill エントリの `path` がディレクトリ指定。
- `<pack>/skills/<name>/`: ディレクトリ。直下に `SKILL.md`、任意のサブパスに sibling ファイル。

### 出力

- 各 target の Inventory ルート配下の `skills/<name>/SKILL.md` + sibling ファイル群。
- 対応する label sidecar 群（Installation ごと）。

## 5. Normal Cases

### NC1. sibling 無しの skill を install

- skill ディレクトリに SKILL.md のみ。
- 結果: 1 Artifact が install される。従来挙動と同じ。

### NC2. sibling 有りの skill を install

- skill ディレクトリに SKILL.md と `scripts/foo.sh` と `references/bar.md`。
- 結果: 3 Installation がそれぞれ作成され、各 sidecar label が書かれる。

### NC3. sibling 有りの skill を uninstall（pack 指定）

- すべての関連 Installation が削除される。

### NC4. sibling 有りの skill を list

- SKILL.md / scripts/foo.sh / references/bar.md がそれぞれ別行として現れる。いずれも同じ ENTRY_ID と PACK を持つ。

## 6. Error Cases

本セクションでは、loader の段階で skill 解決時に生じる失敗を 3 種に区別する。具体的エラー識別子の命名は設計に委ねるが、それぞれが互いに識別可能であること（テストで分岐可能）が要件。

### EC1. SKILL.md 欠落

- skill ルート（`path` で指したディレクトリ）は存在するが、その直下に `SKILL.md` が無い場合、loader は専用のエラー（仮称 `ErrSkillBodyNotFound`）を返す。
- 受け入れ条件: テストから当該エラーを `errors.Is` で識別できる。

### EC2. skill path がファイルを指している

- skill エントリの `path` がディレクトリではなくファイルを指している場合、loader は専用のエラー（仮称 `ErrSkillPathNotDirectory`）を返す。
- 受け入れ条件: テストから当該エラーを `errors.Is` で識別できる。

### EC3. skill path 自体が不在

- skill エントリの `path` が存在しない場合、loader は既存の `ErrEntryNotFound` 相当を返す（path 不在の意味で従来挙動を踏襲）。

### EC4. manifest path 形式違反（schema 違反）

- skill エントリの `path` が新スキーマに合致しない場合（例: 末尾スラッシュ付き、`SKILL.md` を含む、`..` を含む）、validator がスキーマ違反エラーを返す。
- agent / rule / prompt の path がディレクトリ形状である場合も同様にスキーマ違反。

### EC5. sibling 配置時の競合

- 配置先に未管理のファイルがある場合は `ErrUnmanagedArtifactExists`。
- 配置先に既に knit 管理の同じ Installation がある場合は `ErrAlreadyInstalled`。

### EC6. sibling 配置時の途中失敗

- 複数 Artifact を install 中に途中の Artifact で失敗した場合、その時点までに書かれた Installation はそのまま残る（既存仕様と同じ）。
- スコープ外: 集合全体のロールバック。受け入れる。

### EC7. sibling 同士の target path 衝突

- skill 集合内の Artifact 同士が target 上の同一 path に解決される場合、path_policy または Installer 段階で `ErrUnmanagedArtifactExists` / `ErrAlreadyInstalled` のいずれかでエラーになる（後勝ち書き込みは行わない）。

## 7. Edge Cases

### EE1. 空ディレクトリ（SKILL.md のみ）

- F2 のとおり、sibling 0 件として正常に処理される。

### EE2. 深いサブディレクトリ

- 3 階層以上のネストでも、ソース構造をそのまま target に再現する。

### EE3. シンボリックリンク・特殊ファイル

- 本要件のサポート対象は `fs.WalkDir` の結果のうち通常ファイルのみ。ディレクトリエントリでもシンボリックリンクでもない通常ファイルだけが sibling 集合に含まれる。
- シンボリックリンクや特殊ファイルは黙って除外する。テストは作らない。

### EE4. SKILL.md と完全一致する本体の特定

- 本体ファイルの特定は `SKILL.md` の **完全一致** のみ。`skill.md` や `SKILL.MD` 等は本体扱いせず sibling 集合に流れる。
- 大文字小文字を区別しないファイルシステム上で複数候補が同居するケースは fs.FS 側の事前条件違反として扱い、本要件ではテスト対象外。

### EE5. sibling target path 衝突

- ソース側の sibling 同士は fs.FS が一意性を保証するため通常は衝突しない。
- ただし target 側 path_policy による正規化（大小区別の有無など）で衝突が発生する可能性は残る。これは EC7 で扱う。

### EE6. 配置先 target が skills 配下にサブディレクトリ非対応の場合

- 現状の claude / codex / gemini はいずれも `skills/<name>/...` のサブディレクトリを許容する想定。target 側で許容できない場合は path_policy がエラーを返すことで検知される。

## 8. Acceptance Criteria

### AC1. Manifest スキーマ

- **Given** id `<pack>.skill.orchestrator` で `path: skills/orchestrator` の skill エントリ
- **When** loader が validator を通す
- **Then** 検証が成功する

- **Given** id `<pack>.skill.orchestrator` で `path: skills/orchestrator/SKILL.md` の skill エントリ（旧形式）
- **When** loader が validator を通す
- **Then** 検証がエラーになる

- **Given** id `<pack>.skill.orchestrator` で `path: skills/orchestrator/` の skill エントリ（末尾スラッシュ）
- **When** loader が validator を通す
- **Then** 検証がエラーになる

- **Given** id `<pack>.agent.reviewer` で `path: agents/reviewer` の agent エントリ（ディレクトリ指定）
- **When** loader が validator を通す
- **Then** 検証がエラーになる

- **Given** id `<pack>.skill.orchestrator` で `path: agents/reviewer.md` の skill エントリ（kind と path 形式の不整合）
- **When** loader が validator を通す
- **Then** 検証がエラーになる

### AC2. Loader

- **Given** skill ディレクトリに SKILL.md + scripts/foo.sh + references/bar.md
- **When** loader が pack を読む
- **Then** skill Entry 1 件が得られ、`Body` に SKILL.md の本文が入り、sibling 集合に `scripts/foo.sh` と `references/bar.md` の (相対 path, 内容) が（順序不問で）両方含まれる

- **Given** skill ディレクトリは存在するが SKILL.md が無い
- **When** loader が pack を読む
- **Then** `ErrSkillBodyNotFound` 相当のエラーが返り、`errors.Is` で識別できる

- **Given** skill エントリの path がディレクトリではなく既存ファイルを指している
- **When** loader が pack を読む
- **Then** `ErrSkillPathNotDirectory` 相当のエラーが返り、`errors.Is` で識別できる

- **Given** skill エントリの path が存在しない
- **When** loader が pack を読む
- **Then** `ErrEntryNotFound` 相当のエラーが返る

### AC3. Builder

- **Given** sibling 2 件付きの skill Entry が enabled な target
- **When** Builder が pack を build する
- **Then** Artifact 集合に SKILL.md（target 規約上の path）と sibling 2 件（同様に target 規約上の path）が（順序不問で）含まれ、すべて同一の `SourceEntryIDs = [<pack>.skill.<name>]` を持つ

- **Given** sibling 付き skill Entry だが、ある target で skill が enabled でない
- **When** その target の Builder が pack を build する
- **Then** その skill 由来の Artifact は 1 件も生成されない（本体・sibling 双方）

### AC4. Installer

- **Given** sibling 付き skill を build した Artifact 群
- **When** install command が順に Installer.Install を呼ぶ
- **Then** target Inventory 配下に SKILL.md と sibling が正しい相対構造で書き込まれ、それぞれの label sidecar が作成される

### AC5. Uninstaller（pack 指定）

- **Given** sibling 付き skill が install 済み
- **When** pack 名で `knit uninstall` を実行する
- **Then** SKILL.md と sibling の Installation が全て削除され、対応する label sidecar も削除される

### AC6. 後方互換性違反検知

- **Given** 旧形式（path に SKILL.md を含む）の manifest.yaml
- **When** loader が読む
- **Then** スキーマ違反エラーが返る

## 9. Non-Goals

- N1: agent / rule / prompt のディレクトリ単位化
- N2: `knit list` の集約表示（1 skill = 1 行）
- N3: 既存 Installation の自動マイグレーション
- N4: バイナリ・大型アセット向けの効率的取り扱い（streaming, mmap など）
- N5: 集合インストールの全体ロールバック・トランザクション境界の引き上げ
- N6: シンボリックリンク／特殊ファイルのサポート
- N7: target 側に「skills 配下サブディレクトリ非対応」のものを追加する場合の特別扱い

## 10. Risks and Open Questions

### R1. SourceEntryIDs と Installation の対応関係

- 現状: 1 Artifact = 1 Installation、各 Installation は単一の `<pack>.<kind>.<name>` を持つ場合と、rule の集約のように複数 Entry を持つ場合がある。
- 新仕様: 1 skill Entry → 複数 Artifact → 複数 Installation。各 Installation は同一の `<pack>.skill.<name>` を持つ。
- 影響: `BelongsToPack` ベースの uninstall は素直に動作する。Lister 出力では同一 ENTRY_ID が複数行に現れる（許容）。

### R2. label sidecar 数の膨張

- sibling 100 件 × scope 2 × target 3 = 600 sidecar が 1 skill に対して発生する可能性。性能影響はあるが今回のスコープ外。

### R3. 旧 manifest 互換の打ち切りタイミング

- 本要件では旧形式を即時拒否する。`structure-behavior-design` 以外のパックを抱えるユーザーがいた場合は同時に更新が必要。
- 現リポジトリ内には新形式に書き換える対象は `knowledge/structure-behavior-design/manifest.yaml` のみと想定。

### R4. シンボリックリンクや実行ビット

- F3 で「sibling Artifact の Mode はゼロ値」と固定したので解消。シンボリックリンクは EE3 で除外と決定。

### R5. target 側の path policy

- target ごとの path_policy で `skills/<name>/<subpath>` の subpath を許容しているかを実装着手前に確認する必要がある（特に gemini）。許容していなければ policy 側も拡張対象になる。

### R6. 既存テストの破壊範囲

- distribution/{claude,codex,gemini}/skill_renderer_test.go や builder_test.go は「skill は SKILL.md 単一」を前提にしている可能性が高い。テストの書き換えと、新ケースの追加が必要。

### R7. Reinstaller の sibling 増減仕様

- ソース側で sibling が増減した skill を refresh した時、削除された sibling を inventory からも消すかは未決。本要件のスコープ外。挙動は別タスクで規範化する。
