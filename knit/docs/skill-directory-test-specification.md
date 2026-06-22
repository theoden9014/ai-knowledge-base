# Skill Directory Unit - Test Specification

要件: [skill-directory-requirements.md](./skill-directory-requirements.md)
概念モデル: [skill-directory-conceptual-model.md](./skill-directory-conceptual-model.md)
責務割当: [skill-directory-responsibility.md](./skill-directory-responsibility.md)
インタフェース設計: [skill-directory-interface-design.md](./skill-directory-interface-design.md)

本書は振る舞いを Given-When-Then で記述したテスト仕様。実装言語の syntax は書かない。

## 共通方針

- **集合等価**: skill 由来 Artifact の集合は順序保証無し。テストは sort や set-equality で書く。
- **`errors.Is`**: sentinel の判定は `errors.Is(err, sentinel)` で書く。エラーメッセージ文字列パターンマッチには依拠しない。
- **fs.FS**: ソース側は `testing/fstest.MapFS` を用いる。OS ファイルシステムは Loader 単体テストでは触らない（distribution 側 Installer テストでは tmp dir を使う既存パターンを踏襲）。
- **モック最小化**: validator は本物の JSON Schema validator を使う。LabelStore は既存の sidecar 実装か in-memory 実装を使う（過剰なモック禁止）。

---

## 1. SkillAsset コンストラクタ

### Test Specifications

| Behavior | Given | When | Then | Test Level | Notes |
|---|---|---|---|---|---|
| 正常な相対 path と内容で構築できる | `relPath="scripts/foo.sh"`, `content=非空 bytes` | `NewSkillAsset(relPath, content)` | 戻り値 SkillAsset の `Path() == "scripts/foo.sh"`、`Content()` が等価 byte 列、`IsZero() == false`、エラー nil | unit | 防御コピーの検証は不変条件側 |
| サブディレクトリを含む path で構築できる | `relPath="references/sub/bar.md"` | 同上 | エラー nil、`Path()` で復元可 | unit | |
| 空 path はエラー | `relPath=""` | 同上 | `errors.Is(err, ErrInvalidSkillAssetPath) == true` | unit | |
| 絶対 path はエラー | `relPath="/etc/foo"` | 同上 | `errors.Is(err, ErrInvalidSkillAssetPath) == true` | unit | |
| `..` を含む path はエラー | `relPath="../escape.sh"` | 同上 | `errors.Is(err, ErrInvalidSkillAssetPath) == true` | unit | |
| `SKILL.md` 完全一致はエラー | `relPath="SKILL.md"` | 同上 | `errors.Is(err, ErrInvalidSkillAssetPath) == true` | unit | 本体ファイルを sibling 扱いさせない |
| 区切りに OS 固有のバックスラッシュは禁止 | `relPath="scripts\\foo.sh"` | 同上 | `errors.Is(err, ErrInvalidSkillAssetPath) == true` | unit | 区切りは forward slash 限定。ファイル名にバックスラッシュを含むケースの仕様化はしない（実装上は単純に `\\` を含む path を弾く） |

### Invariant Tests

| Invariant | Example | Expected Result |
|---|---|---|
| Content の防御コピー | 構築後に元の byte slice を変更 | `Content()` 戻り値は影響を受けない |
| Path 不変 | 構築後の SkillAsset を別変数に代入 | 両方の `Path()` は同一値 |

---

## 2. SkillMeta コンストラクタ

### Test Specifications

| Behavior | Given | When | Then | Test Level | Notes |
|---|---|---|---|---|---|
| 正常な Root と空 Assets で構築できる | `root="skills/orchestrator"`, `assets=[]` | `NewSkillMeta(root, assets)` | 戻り値 `*SkillMeta` 非 nil、`Root() == root`、`Assets()` が空、エラー nil | unit | |
| 複数 Assets で構築できる | `root="skills/foo"`, `assets=[a1, a2]`（path 重複なし） | 同上 | エラー nil、`Assets()` の集合が `{a1, a2}` と等価 | unit | 順序保証なし |
| Root が空はエラー | `root=""` | 同上 | `errors.Is(err, ErrInvalidSkillRoot) == true` | unit | |
| Root に末尾スラッシュ | `root="skills/foo/"` | 同上 | `errors.Is(err, ErrInvalidSkillRoot) == true` | unit | |
| Root に `..` を含む | `root="skills/../escape"` | 同上 | `errors.Is(err, ErrInvalidSkillRoot) == true` | unit | |
| Root が絶対パス | `root="/skills/foo"` | 同上 | `errors.Is(err, ErrInvalidSkillRoot) == true` | unit | |
| Assets に Path 重複あり | `root="skills/foo"`, `assets=[a, a]`（同じ Path） | 同上 | `errors.Is(err, ErrDuplicateSkillAsset) == true` | unit | |

### Invariant Tests

| Invariant | Example | Expected Result |
|---|---|---|
| Assets 防御コピー | 構築後に元 assets slice を変更（要素追加） | `Assets()` 戻り値の長さは不変 |

---

## 3. Manifest schema validator

### Test Specifications

| Behavior | Given | When | Then | Test Level | Notes |
|---|---|---|---|---|---|
| skill 新形式は受け入れ | id=`p.skill.foo`, path=`skills/foo` | validator が manifest 全体を検証 | エラー nil | unit | |
| skill で SKILL.md を含む path は拒否 | id=`p.skill.foo`, path=`skills/foo/SKILL.md` | 同上 | schema 違反エラー | unit | 旧形式拒否 |
| skill で末尾スラッシュ付き path は拒否 | id=`p.skill.foo`, path=`skills/foo/` | 同上 | schema 違反エラー | unit | |
| agent でディレクトリ形式 path は拒否 | id=`p.agent.bar`, path=`agents/bar` | 同上 | schema 違反エラー | unit | kind=agent はファイル必須 |
| rule でディレクトリ形式 path は拒否 | id=`p.rule.baz`, path=`rules/baz` | 同上 | schema 違反エラー | unit | |
| prompt でディレクトリ形式 path は拒否 | id=`p.prompt.qux`, path=`prompts/qux` | 同上 | schema 違反エラー | unit | |
| kind と path の不整合（id=skill, path=agents/X.md） | id=`p.skill.foo`, path=`agents/foo.md` | 同上 | schema 違反エラー | unit | |
| kind と path の不整合（id=agent, path=skills/X） | id=`p.agent.bar`, path=`skills/bar` | 同上 | schema 違反エラー | unit | |
| skill path に `..` を含む | id=`p.skill.foo`, path=`skills/../escape` | 同上 | schema 違反エラー | unit | |
| 非 skill 既存形式は引き続き受け入れ | id=`p.agent.bar`, path=`agents/bar.md` | 同上 | エラー nil | unit | 後方互換確認 |

---

## 4. Loader

すべて `fstest.MapFS` で fs.FS を構築する unit テスト。

### 4.1 正常系

| Behavior | Given | When | Then | Notes |
|---|---|---|---|---|
| sibling 無しの skill を読める | manifest に skill 1 件、`skills/foo/SKILL.md` のみ存在 | `loader.LoadPack(ctx, fsys, packDir)` | Entry 1 件、`Kind=KindSkill`、`Body` が **既存 Loader と同じセマンティクスで frontmatter 除去後の本文 byte 列**、`Skill != nil`、`Skill.Root() == "skills/foo"`、`Skill.Assets()` が空集合 | |
| sibling 1 件付き skill を読める | `skills/foo/SKILL.md` + `skills/foo/scripts/run.sh` | 同上 | Entry の `Skill.Assets()` が `{(Path="scripts/run.sh", Content=run.sh の中身)}` と等価 | 集合等価 |
| sibling 複数階層付き skill を読める | `skills/foo/SKILL.md` + `skills/foo/scripts/run.sh` + `skills/foo/refs/a/b.md` | 同上 | `Skill.Assets()` 集合が 2 要素で正しい (Path, Content) を持つ | 階層保持 |
| non-skill エントリは従来どおり読める | manifest に agent 1 件、`agents/bar.md` | 同上 | Entry の `Kind=KindAgent`、`Skill == nil`、`Body` が agent ファイル | 後方互換 |
| skill と non-skill 混在 manifest | skill 1 件 + agent 1 件 | 同上 | 2 Entry が manifest 順で返り、それぞれの Skill/Agent フィールドが正しく nil/非 nil |  |

### 4.2 異常系（skill 解決）

| Behavior | Given | When | Then | Notes |
|---|---|---|---|---|
| skill path 不在 | manifest の skill path=`skills/missing`、ファイルシステムに当該 dir 無し | `LoadPack` | `errors.Is(err, ErrSkillPathNotFound) == true` かつ `errors.Is(err, ErrSkillResolution) == true` | sentinel 識別 |
| skill path がファイル | manifest の skill path=`skills/file`、fs に `skills/file`（ファイル）が存在 | 同上 | `errors.Is(err, ErrSkillPathNotDirectory) == true` かつ `errors.Is(err, ErrSkillResolution) == true` | |
| SKILL.md 欠落 | `skills/foo/` ディレクトリは存在するが直下に SKILL.md 無し | 同上 | `errors.Is(err, ErrSkillBodyNotFound) == true` かつ `errors.Is(err, ErrSkillResolution) == true` | |
| 3 種 sentinel は互いに独立 | 直前ケース（「skill path 不在」）で Loader が返したエラーオブジェクト | `errors.Is(err, ErrSkillPathNotDirectory)` / `errors.Is(err, ErrSkillBodyNotFound)` | いずれも false | 識別性。Loader 駆動経由で得た err 値を使う |
| `ErrSkillResolution` は SkillAsset/SkillMeta コンストラクタ系をカバーしない | `ErrInvalidSkillAssetPath` を返した err | `errors.Is(err, ErrSkillResolution)` | false | 別カテゴリ |
| skill エントリで manifest schema 違反 | manifest path=`skills/foo/SKILL.md`（旧形式） | 同上 | schema 違反エラー（`ErrSkillResolution` には該当しない） | |
| 複数 skill 異常時は manifest 順で最初の異常を返す | skill 1=正常、skill 2=path 不在、skill 3=SKILL.md 欠落 | 同上 | エラーは skill 2 由来（`ErrSkillPathNotFound`）。skill 3 の事象は観測されない | fail-fast |

### 4.3 ファイル種別の扱い

| Behavior | Given | When | Then | Notes |
|---|---|---|---|---|
| SKILL.md と完全一致でないファイルは sibling | `skills/foo/SKILL.md` + `skills/foo/skill.md` | 同上 | `skill.md` は sibling 集合に **含まれる** | 要件 EE4（完全一致のみ本体扱い） |
| ディレクトリエントリ（直下が空サブディレクトリ）は無視 | `skills/foo/SKILL.md` + `skills/foo/empty_dir/` のみ | 同上 | `Skill.Assets()` は空集合 | 要件 EE1 系（ディレクトリは sibling 集合に含めない） |
| シンボリックリンク扱い | OS 依存・`fstest.MapFS` で再現困難 | — | テスト対象外 | 要件 EE3（除外） |

---

## 5. Builder + skill_renderer (claude / codex / gemini それぞれで実施)

### 5.1 振る舞い

| Behavior | Given | When | Then | Notes |
|---|---|---|---|---|
| sibling 無しの skill を build | `Entry.Kind=KindSkill`, `Entry.Path="skills/<name>"`, `Entry.Skill = NewSkillMeta("skills/<name>", nil)`（Assets は空集合の非 nil SkillMeta） | `builder.Build(ctx, pack)` | Artifact 集合に 1 件、Path=`skills/<name>/SKILL.md`、Content が SKILL.md frontmatter + body | `Entry.Skill == nil` のケースは契約違反（テスト対象外） |
| sibling 1 件の skill を build | `Entry.Skill = NewSkillMeta("skills/<name>", [NewSkillAsset("scripts/run.sh", X)])` | 同上 | Artifact 集合が 2 件: { Path=`skills/<name>/SKILL.md`, Path=`skills/<name>/scripts/run.sh` (Content=X) }、両者の `SourceEntryIDs` は長さ 1 で `[entry.ID]` と値等価 | 集合等価、複数 entry のマージは行われない |
| sibling 多件・サブディレクトリ付きの skill を build | Assets=[NewSkillAsset("scripts/run.sh", X), NewSkillAsset("refs/a/b.md", Y)] | 同上 | Artifact 3 件、Path が `skills/<name>/scripts/run.sh` / `skills/<name>/refs/a/b.md` / `skills/<name>/SKILL.md`（集合等価） | サブディレクトリ保持 |
| sibling Artifact の Mode はゼロ値 | Assets=[NewSkillAsset("scripts/run.sh", 非空 bytes)] | 同上 | 該当 Artifact の `Mode == 0` | F3 契約 |
| skill が target enabled でない | `Entry.Kind=KindSkill`、`Entry.Skill` あり（Assets=[scripts/run.sh]）、`Entry.Tools[target].Enabled=false` | 同上 | その skill 由来 Artifact は 0 件（本体・sibling とも） | |
| skill が `default_tools` 非対象 | `Entry.Kind=KindSkill`、`Entry.Skill` あり、`Pack.DefaultTools` に当該 target 無し、`Entry.Tools[target].Enabled` 未指定 | 同上 | その skill 由来 Artifact は 0 件 | |
| 既存 agent/prompt は単一 Artifact | `Entry.Kind=KindAgent`、`Entry.Skill=nil` | 同上 | Artifact 1 件、`SourceEntryIDs=[entry.ID]` | KindRenderer 戻り型変更影響無し |

### 5.2 ターゲット固有 path

| target | 本体 path | sibling path |
|---|---|---|
| claude | `skills/<name>/SKILL.md` | `skills/<name>/<asset.Path>` |
| codex | `skills/<name>/SKILL.md` | `skills/<name>/<asset.Path>` |
| gemini | `skills/<name>/SKILL.md` | `skills/<name>/<asset.Path>` |

3 target すべて同形（distribution 内で同一テストを書く）。

### 5.3 path_policy（既存テストが境界として確立しているもののみ）

codex の `validateSkillPath` 拡張は既存 path_policy_test.go パターンが存在する場合のみそこで unit テストを増やす。**存在しない場合は Builder 経由のテスト（5.1）で sibling Artifact が拒否されないことを確認することで代替する**。

| Behavior | Given | When | Then | Notes |
|---|---|---|---|---|
| codex の path policy 拡張テスト（既存 path_policy_test.go がある場合のみ） | path=`skills/foo/scripts/run.sh`, `skills/foo`, `skills/foo/notes.txt` | `pathPolicy.Validate(path)` | サブパス付きは受け入れ、bare top は拒否 | path_policy が既に外向き境界として確立している前提 |

---

## 6. Installer / Lister / Uninstaller（各 target）

各 target で同様のテストを書く。LabelStore は in-memory または既存 sidecar 実装。ファイル書き込み先は `t.TempDir()`。

### 6.1 Installer

| Behavior | Given | When | Then | Notes |
|---|---|---|---|---|
| sibling 付き skill を install | Builder が返した skill 由来 Artifact 3 件 | install command が順に `installer.Install` を呼ぶ | 3 Installation がそれぞれ作成され、対応する label sidecar 3 個と placement file 3 個が存在する | 既存 install command の semantics |
| sibling Artifact の `SourceEntryIDs` は label に保存 | 同上 | 同上 | 各 Installation の Provenance.SourceEntryIDs に `[entry.ID]` が含まれる | F3 検証 |
| 既存ファイル（unmanaged）と衝突した sibling | テスト前段で sibling 1 件目の target 配置 path に未管理ファイルを直接書き込んでおく | install | `errors.Is(err, ErrUnmanagedArtifactExists)` | 既存 sentinel |
| 既に install 済みと衝突した sibling | テスト前段で当該 EntryID + sibling 1 件目の path で別途 install 済みにする（同一 LabelStore + ArtifactStore に対し直接 install） | install | `errors.Is(err, ErrAlreadyInstalled)` | 既存 sentinel |
| 部分失敗（途中の sibling で衝突） | テスト前段で sibling 2 件目（`scripts/run.sh`）の target path に knit 管理の Installation を作っておき、本体 + sibling 1 件目は未配置 | install command 順次実行（本体 → sibling[0] → sibling[1]） | 本体と sibling 1 件目までは Installation が残る。sibling 2 件目で `ErrAlreadyInstalled`。3 件目以降は実行されない | 集合ロールバックは非ゴール。setup は in-memory または既存 sidecar LabelStore + tmp dir |

### 6.2 Lister

| Behavior | Given | When | Then | Notes |
|---|---|---|---|---|
| sibling 付き skill を list | install 済み（本体 + sibling 2） | `lister.List(ctx, scope)` | 3 Installation が返る。すべて `Label.Target` 一致、Provenance.SourceEntryIDs が同一 `[entry.ID]` | |
| `BelongsToPack` で skill 集合を一括抽出 | install 済み skill 3 + 他 pack 1 | List してから `Provenance.BelongsToPack(packName)` でフィルタ | skill 由来 3 件のみが返る | F5 検証 |

### 6.3 Uninstaller

| Behavior | Given | When | Then | Notes |
|---|---|---|---|---|
| pack 指定で sibling まで一括 uninstall | install 済み skill 3 件（本体 + sibling 2） | `cmd_uninstall` を当該 pack で実行 | 3 Installation と対応 label が全削除。target 配置ファイルも全削除 | F6 検証 |
| pack 指定で他 pack は残る | 当該 pack 3 + 他 pack の skill 1 | 同上 | 当該 pack の 3 件のみ削除、他 pack 1 件は残る | |
| 個別 Installation の uninstall は従来どおり | install 済み skill の sibling 1 件のみを `uninstaller.Uninstall(ctx, inst)` | uninstall | その 1 件のみ削除、他は残る | 個別単位の動作確認 |
| 部分失敗時の振る舞い（観測契約のみ） | install 済み skill 3 件のうち 2 件目を手動で外部削除 → label と file が乖離した状態 | `cmd_uninstall` で当該 pack 全件 uninstall を実行 | 既存挙動どおり「`ErrInstallationNotFound` は warning で継続」され、結果として label/file 双方が無くなる。返るエラーは無し | IO error 注入は ArtifactStore モックの新規導入が必要になるため自動テストはこの形に留める。任意のフェイル注入は将来の課題 |

---

## 7. 結合（end-to-end ライクな install → list → uninstall）

CLI レベルの結合テスト。`fstest.MapFS` でソースを作り、`t.TempDir()` で各 target のルートを切る。

| Behavior | Given | When | Then | Notes |
|---|---|---|---|---|
| sibling 付き skill を pack install → list → uninstall | manifest + skill ディレクトリ + sibling 2 件 | `cmd_install` → `cmd_list` → `cmd_uninstall` | (a) install で 3 Installation 作成 / (b) list で 3 行（PACK = pack 名、ENTRY_ID 同一） / (c) uninstall で全削除 | パイプライン |
| 集合の Lister 出力は同一 ENTRY_ID を複数行で含む | 上記 install 後 | list | ENTRY_ID が `<pack>.skill.<name>` の行が 3 つ | 集合の動的ビュー |

---

## 8. KindRenderer 戻り型変更と既存テストへの影響

### 8.1 既存テストの破壊範囲

新仕様により既存テストでは下記 2 点を同時に変更する必要がある:

1. **skill Entry の `Path` 形式変更**: 既存テストは `Path: "skills/<name>.md"` のような単一ファイル形式で Entry リテラルを組んでいる。新仕様では skill の `Path == Skill.Root() == "skills/<name>"`（ディレクトリ・末尾スラッシュ無し）に揃える。
2. **`Entry.Skill` フィールドの構築**: skill Entry を直接リテラルで作るすべてのテストで `Skill: &SkillMeta{Root: "skills/<name>", Assets: []SkillAsset{...}}` を追加。Assets 空でも非 nil で構築する（後述 8.3 の不変条件）。
3. **KindRenderer.Render 戻り型**: `Artifact` から `[]Artifact` へ変更。既存テストの assertion は `art := renderer.Render(...)` を `arts := renderer.Render(...)`、`art.Path` を `arts[0].Path` のように書き換える。

### 8.2 対象ファイル（実在確認済み・テスト本体）

| ファイル | 主な修正点 |
|---|---|
| `internal/source/loader_impl_test.go` | skill Entry の `Path` 期待値、`Skill` フィールドの組み立て、skill 用 fixture の MapFS 構造（ディレクトリ + SKILL.md）への変更 |
| `internal/source/validator_test.go` | skill の path 形式（新形式・旧形式）に対する schema 検証テスト追加 |
| `internal/distribution/claude/builder_test.go` | skill Entry リテラル変更、`renderer.Render` 戻り型 `[]Artifact` への対応 |
| `internal/distribution/codex/builder_test.go` | 同上 |
| `internal/distribution/gemini/builder_test.go` | 同上 |
| `internal/distribution/claude/installer_test.go` / `lister_test.go` / `uninstaller_test.go` | skill Entry fixture が sibling を含む場合のテスト追加 |
| `internal/distribution/codex/{installer,lister,uninstaller}_test.go` | 同上 |
| `internal/distribution/gemini/{installer,lister,uninstaller}_test.go` | 同上 |
| `internal/distribution/codex/path_policy_test.go`（あれば） | `validateSkillPath` 拡張に対応 |

### 8.3 振る舞いテスト

| Behavior | Given | When | Then | Notes |
|---|---|---|---|---|
| 既存の agent_renderer / prompt_renderer は 1 要素スライスを返す | 任意の agent Entry（Skill=nil） | `agentRenderer.Render(entry, pack)` | `len(arts) == 1`、`arts[0]` の値は既存テストと同等 | 既存テストの assertion を `arts[0]` 経由に更新 |
| RendererRegistry.Build は要素数非依存で append | Pack 内に skill (Skill.Assets=[a1, a2] で計 3 件返す) + agent (1 件) + rule (1 件、aggregator 経由) | `registry.Build(ctx, pack)` | 合計 5 件の Artifact が返る（集合等価） | OCP 検証 |
| Render が空スライス + nil error を返した場合 | テスト用 fake renderer が空 + nil を返す | `registry.Build` | **契約違反扱い**。実装側で本ケースを生まないことが LSP 契約。フレームワーク側の追加 sentinel は導入しない。本ケースのテストは書かない | LSP 注記 |

---

## Invariant Tests（要約）

| Invariant | Example | Expected Result |
|---|---|---|
| SkillAsset.Path に SKILL.md は入らない | NewSkillAsset("SKILL.md", ...) | `ErrInvalidSkillAssetPath` |
| SkillMeta.Assets に重複は無い | NewSkillMeta(root, [a, a]) | `ErrDuplicateSkillAsset` |
| Entry.Skill は Kind 連動で nil / 非 nil | Loader が組み立てた Entry | KindSkill のみ非 nil |
| skill 由来 Artifact の SourceEntryIDs は全要素同一 | Builder が返した skill 集合 | 全要素 `[entry.ID]` |
| sibling Artifact の Mode はゼロ値 | Builder が返した sibling | `Mode == 0` |
| Loader fail-fast | manifest に複数 skill 異常 | 最初の異常 entry のエラーのみ観測 |
| `errors.Is(err, ErrSkillResolution)` は skill 解決 3 種で true | `ErrSkillPathNotFound`, `ErrSkillPathNotDirectory`, `ErrSkillBodyNotFound` のいずれか | すべて true |

---

## Error / Edge Case Tests（要約）

| Case | Given | When | Then |
|---|---|---|---|
| 空 manifest | `entries: []` | `LoadPack` | schema validator が `minItems: 1` 違反を返す（既存挙動） |
| skill ディレクトリが空（ファイル 0） | `skills/foo/` のみ存在、中身ゼロ | `LoadPack` | `ErrSkillBodyNotFound` |
| sibling のみ存在し SKILL.md 無し | `skills/foo/scripts/run.sh` 存在、SKILL.md 無し | `LoadPack` | `ErrSkillBodyNotFound` |
| シンボリックリンク | OS 依存。MapFS では再現困難 | — | テストしない（要件 EE3 で除外） |
| 大文字小文字違いの SKILL.md（`Skill.md`） | `skills/foo/Skill.md` のみ存在 | `LoadPack` | `ErrSkillBodyNotFound`（`SKILL.md` 完全一致を要求） |
| skill 名に kebab-case 違反 | id=`p.skill.Foo_Bar` | `LoadPack` | manifest schema 違反 |

---

## Testability Feedback

### Interface concerns

- `Entry.Skill` を Kind 連動で `nil` / 非 nil にする方針は、skill_renderer が `entry.Skill != nil` を assume できることでテストが書きやすくなる。逆に、Builder/Registry のテストで「Entry.Kind=Skill かつ Entry.Skill=nil」という不正状態を作りたくなったら、それは概念モデル違反なので **テストしない**（コンストラクタで防ぐ責務）。
- `SkillAsset` / `SkillMeta` のコンストラクタが値オブジェクトの不変条件を吸収するため、上流の Loader / Renderer テストは「正常な値オブジェクト」を前提に書ける。

### Responsibility concerns

- skill ルート解決の 3 種異常は、Loader 内の単一関数で発生する想定。テストは「Loader を駆動して error の sentinel を判定」までで十分。内部関数を直接テストしない。
- SkillAssetCollector を internal にすることで、テストは Loader 経由でカバーする方針が成立する。Collector 単体テストは不要（カバレッジは Loader テストで足りる）。

### Coupling concerns

- skill_renderer のテストは Builder 経由で書く（既存テストパターンの踏襲）。skill_renderer 単体の白箱テストは避ける。
- Installer / Uninstaller / Lister のテストは distribution 層既存テストの skill ケースに「sibling を含む Entry」を追加する形で書く。in-memory LabelStore があれば優先使用。なければ既存の SidecarLabelStore + tmp dir。

### Mocking concerns

- fs.FS のモックは `fstest.MapFS` で完結する（外部依存無し）。
- validator は本物の JSON Schema validator を使う。シミュレーション用の fake validator は作らない。
- LabelStore も in-memory または本物 sidecar を使う。fake は作らない。
