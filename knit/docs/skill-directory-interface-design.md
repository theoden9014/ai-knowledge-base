# Skill Directory Unit - Interface Design

要件: [skill-directory-requirements.md](./skill-directory-requirements.md)
概念モデル: [skill-directory-conceptual-model.md](./skill-directory-conceptual-model.md)
責務割当: [skill-directory-responsibility.md](./skill-directory-responsibility.md)
既存インタフェース: [refactoring-interface-design.md](./refactoring-interface-design.md)

本書は型・関数の **契約レベル** の記述に留める。具体的な Go コード断片は載せない。

## 1. 追加される型

### 1.1 `source.SkillAsset`

- **目的**: skill ルートディレクトリ配下の本体以外のファイル 1 個を表す値オブジェクト。
- **状態**: skill ルートからの相対パス文字列 / 不透明な byte 列。
- **公開アクセサ**: `Path() string` / `Content() []byte`（防御コピー返却）/ `IsZero() bool`。
- **コンストラクタ契約**: `NewSkillAsset(relPath string, content []byte) (SkillAsset, error)`
  - 前提: `relPath` は空でない、`..` を含まない、絶対パスでない、`/` を含み得るが forward slash のみ、`SKILL.md`（完全一致）でない。
  - 違反時は `ErrInvalidSkillAssetPath`（仮称）を返す。
  - 事後: 構築済み SkillAsset の `Path()` 戻り値は引数を変更せず保持する。`Content()` は引数を防御コピーした内部表現を返す。
- **不変条件**: 一度構築されたら状態は不変。

### 1.2 `source.SkillMeta`

- **目的**: Entry の Kind=KindSkill のときのみ存在するサブ構造。skill ルートと SkillAsset 集合を一体として保持する。
- **状態**: `Root` (skill ルートの pack 相対 path 文字列) / `Assets` ([]SkillAsset 集合)。
- **公開アクセサ**: `Root() string` / `Assets() []SkillAsset`（防御コピー返却）。
- **コンストラクタ契約**: `NewSkillMeta(root string, assets []SkillAsset) (*SkillMeta, error)`
  - 前提: `root` は空でない pack 相対ディレクトリ path、末尾スラッシュ無し、`..` を含まない、絶対パスでない、forward slash 区切り。
  - 前提: `assets` 内の Path に重複が無い。
  - 違反時は `ErrInvalidSkillRoot` または `ErrDuplicateSkillAsset`（仮称）。
  - 事後: 戻り値は構築済みの不変オブジェクトへのポインタ。
- **不変条件**: `Root` は構築後不変、`Assets` のスライスは内部コピーを保持。

### 1.3 skill ルート解決（内部実装の指針）

- **概念モデルの SkillRootResolution は内部表現としても型化しない**。Loader 内の単一関数が fs.Stat + os.OpenFile 等で 3 種の異常を直接判定し、対応する sentinel をその場で返す。
- 中間 enum を持たない理由: enum → sentinel の二段マッピングが test 不能な内部抽象になり、provider 都合の型になりがち。
- 「3 状態を 1 関数で判定する」責務は残る。コードレビューと test specification 側で、判定パスが 1 箇所に集まっていることを確認する。

### 1.4 `source.SkillAssetCollector`（**内部** 型）

- **目的**: skill ルート配下から本体以外の通常ファイル集合を返す境界。
- **位置**: `internal/source` 内、unexported。
- **関数契約**: `(c *skillAssetCollector) Collect(fsys fs.FS, skillRoot string) ([]SkillAsset, error)`
  - 前提: `skillRoot` は SkillMeta コンストラクタ前提と同じ制約を満たす。
  - 事後: 戻り値の各 SkillAsset は本体 `SKILL.md` を含まない。ディレクトリ・シンボリックリンク・特殊ファイルは除外。
  - エラー: fs.FS が返したエラーをラップして返す。fs.FS のセマンティクスに依拠（具体例: open エラーは I/O エラーとして返す）。
- **interface 化しない**: 単一実装で十分。Loader の内部協力者として保持。

## 2. 変更される型

### 2.1 `source.Entry`

- **追加フィールド**: `Skill *SkillMeta`
- **不変条件追加**:
  - `Entry.Kind == KindSkill` のとき `Entry.Skill != nil`。
  - `Entry.Kind != KindSkill` のとき `Entry.Skill == nil`。
  - skill のとき `Entry.Path == Entry.Skill.Root()`（loader が組み立て時に同期）。
- **doc コメント変更**: `Entry.Path` の説明を「pack 相対の場所参照（kind に応じてファイル or ディレクトリ）」に更新。skill 用途では skill ルートディレクトリを指す例を追記。
- **後方互換**: agent/rule/prompt の Entry には影響無し（`Skill` は nil）。
- **影響**: Entry を値コピーする箇所は内部の `*SkillMeta` を共有することになる。SkillMeta は構築後不変なので問題なし。

### 2.2 `source.KindRenderer`

- **シグネチャ変更**: `Render(entry *Entry, pack *Pack) ([]Artifact, error)`（旧 `Artifact` を `[]Artifact` に）。
- **契約（事後条件）**:
  - **成功時に返るスライスは要素数 ≥ 1 でなければならない**。空スライス + nil error の組み合わせは **契約違反**（実装側で守るべき不変条件）。`Render` が空集合を返すべきケース（例: enabled でない）は呼び出し前に Builder/Registry 側で除外する。空集合返却は LSP 違反として扱う。
  - 返した全 Artifact は同一 `SourceEntryIDs`（= `[entry.ID]`）を持つ。
  - skill 用 renderer は本体 1 + sibling N の合計 1+N 個。agent/prompt 用 renderer は 1 要素。
  - rule の renderer は引き続き存在しない（RuleAggregator が担当）。
- **pack 引数の役割**: 呼び出し側互換のため引数として残す。skill renderer は pack を読まず entry のみで完結する。agent renderer 等で pack を参照する既存実装は変更なし。
- **既存 API 破壊**: source.KindRenderer 実装者は全員シグネチャ変更が必要。影響範囲は次の 9 ファイル（実在確認済み）:
  - `internal/distribution/claude/{skill,agent,prompt}_renderer.go`
  - `internal/distribution/codex/{skill,agent,prompt}_renderer.go`
  - `internal/distribution/gemini/{skill,agent,prompt}_renderer.go`

### 2.3 `source.RendererRegistry.Build`

- **シグネチャ不変**: `Build(ctx context.Context, pack *Pack) ([]Artifact, error)`
- **内部処理変更**:
  - 旧: `art, err := renderer.Render(...)` → `append(artifacts, art)`
  - 新: `arts, err := renderer.Render(...)` → `append(artifacts, arts...)`
- **要素数非依存**: 新 ループ は要素数 1 / N どちらでも動作。
- **エラー契約**: 既存どおり `ErrUnsupportedKind` / context cancel をそのまま伝播。

### 2.4 `source.Loader`

- **シグネチャ不変**: `LoadPack(ctx, fsys, packDir) (*Pack, LoadInfo, error)`
- **エラー契約の追加**:
  - 既存: `ErrManifestNotFound` / `ErrDuplicateEntryID` / `ErrEntryNotFound` / `ErrIDMismatch` / schema 違反。
  - 追加: skill ルート解決の異常 3 種。**skill 用は専用 sentinel を新設し `ErrEntryNotFound` 共有は避ける**（要件 AC2 の `errors.Is` 識別性を担保するため）。
- **新規エラー値**（`internal/source` で sentinel として公開）:
  - `ErrSkillPathNotFound`: skill エントリの `path` で指したディレクトリが存在しない。
  - `ErrSkillPathNotDirectory`: skill エントリの `path` がファイルを指している。
  - `ErrSkillBodyNotFound`: skill ルートディレクトリは存在するが直下に `SKILL.md` が無い。
  - 傘 sentinel `ErrSkillResolution`: skill 解決系のエラーを「まとめて判定」できるようにする上位 sentinel。上記 3 種は `errors.Is(err, ErrSkillResolution)` を満たす。
- **`errors.Is` 関係**:
  - 3 種の skill 専用 sentinel は互いに独立（`errors.Is` は相互 false）。
  - 3 種はいずれも `errors.Is(err, ErrSkillResolution)` で一括判定可能。
  - 既存 `ErrEntryNotFound` は agent/rule/prompt の path 不在で従来どおり使用。skill では使わない。
- **事後**: 成功時、skill エントリの Entry には Skill フィールドが組み立てられている。

### 2.5 `source.Artifact`

- **シグネチャ不変**: 構造体フィールドは変更なし。
- **doc コメント追記**: 「複数 Artifact が同一 `SourceEntryIDs` を共有する場合がある（skill の本体 + sibling 等）」という既存コメントを契約化する旨を強調する追補のみ。

### 2.6 Manifest JSON Schema

- **論理構造（具体 JSON ではなく構造の言葉）**:
  - `entries[]` の各要素は `id` と `path` を持つ。
  - `id` の正規表現は既存と同じ `^<pack>.<kind>.<name>$` パターン。
  - `path` の制約は `id.kind` の値で **分岐する** :
    - `id.kind = skill` の場合: `path = skills/<kebab-case>`（末尾スラッシュ無し、`SKILL.md` を含まない、`/` を 1 個のみ）。
    - `id.kind = agent` の場合: `path = agents/<kebab-case>.md`。
    - `id.kind = rule` の場合: `path = rules/<kebab-case>.md`。
    - `id.kind = prompt` の場合: `path = prompts/<kebab-case>.md`。
  - これは JSON Schema の `oneOf` / `allOf` + `if`/`then` の組み合わせで表現可能。
- **検証エラーの観測可能性**:
  - schema 違反は `source.Validator` 経由で `ErrSchemaViolation` 系として返る（既存）。新 sentinel は導入しない。
  - エラーメッセージには違反した entry の `id` と `path`、期待 path 形式が含まれる（既存 schema validator の挙動）。

## 3. エラー sentinel 階層

```
（top-level、source package）
├── ErrManifestNotFound        (既存)
├── ErrDuplicateEntryID         (既存)
├── ErrEntryNotFound            (既存・agent/rule/prompt の path 不在で使用)
├── ErrIDMismatch               (既存)
├── ErrPackDirNotFound          (既存)
├── ErrSkillResolution          ★新規・傘 sentinel（skill 解決系の親）
│   ├── ErrSkillPathNotFound    ★新規 (skill path 不在)
│   ├── ErrSkillPathNotDirectory ★新規 (skill path がファイル)
│   └── ErrSkillBodyNotFound    ★新規 (SKILL.md 欠落)
├── ErrInvalidSkillRoot         ★新規 (SkillMeta コンストラクタ)
├── ErrInvalidSkillAssetPath    ★新規 (SkillAsset コンストラクタ)
└── ErrDuplicateSkillAsset      ★新規 (SkillMeta コンストラクタ)
```

- skill 系 3 sentinel は **互いに独立**（`errors.Is` で相互 false）かつ **`ErrSkillResolution` に対しては true**（傘で一括判定可能）。実現方法は実装に委ねる（複数 sentinel を持つカスタムエラー、または `errors.Is` を実装した独自型）。
- Loader 経由のエラーは `fmt.Errorf("source: %s: %w", path, <sentinel>)` 形式でラップされ、`errors.Is` で識別可能。
- SkillMeta / SkillAsset コンストラクタは値オブジェクト単位の sentinel を返す（loader を経由しないテスト用途）。これらは `ErrSkillResolution` の傘下に **入れない**（コンストラクタの引数バリデーションと、ファイルシステム解決の異常は別カテゴリ）。

## 4. 既存 API への破壊的変更まとめ

| API | 変更 | 影響範囲 |
|---|---|---|
| `source.KindRenderer.Render` | 戻り型 `Artifact` → `[]Artifact` | 各 target の `skill_renderer.go` / `agent_renderer.go` / `prompt_renderer.go`（計 9 ファイル） |
| `source.Entry` | `Skill *SkillMeta` フィールド追加 | 構造体リテラルで Entry を組み立てるテスト等が複数 |
| `source.Entry.Path` | doc コメントの意味更新（skill ではディレクトリ） | doc のみ。挙動変化なし |
| `internal/source/schemas/manifest.schema.json` | skill の path 形式変更 + kind と path の整合スキーマ | 既存 manifest.yaml の書き換えが必要 |
| `knowledge/structure-behavior-design/manifest.yaml` | skill 各エントリの `path` 末尾 `/SKILL.md` を削除 | 1 ファイル |

## 5. Example Call Sites（疑似シグネチャ）

### 5.1 Loader 利用側

```
loader := source.NewLoader(validator)
pack, info, err := loader.LoadPack(ctx, fsys, "structure-behavior-design")
// err が errors.Is(err, source.ErrSkillBodyNotFound) を満たすなら、
// どこかの skill ルートに SKILL.md が無かったことを示す
```

### 5.2 skill renderer 利用側（distribution 内部）

```
arts, err := skillRenderer.Render(entry, pack)
// arts は 1 + len(entry.Skill.Assets()) 個
// 全要素の SourceEntryIDs は同一スライス値（順序保証なし）
```

### 5.3 RendererRegistry.Build

```
arts, err := registry.Build(ctx, pack)
// rule をまだ集約していない時点でも arts はフラットなスライスで返り、
// 要素数 1 / N の renderer どちらでも順序依存無く扱われる
```

### 5.4 manifest schema 違反のエラー観測

```
_, _, err := loader.LoadPack(ctx, fsys, "structure-behavior-design")
// err は schema validator が報告するメッセージを保持。
// メッセージには違反した entry の id と path、期待 path 形式が含まれる。
// errors.Is(err, source.ErrSkillResolution) は false（schema 違反は別カテゴリ）。
```

### 5.5 skill renderer が SkillMeta を読む典型

```
// skill renderer 内部（疑似）
assert entry.Kind == KindSkill
assert entry.Skill != nil   // Loader が保証する不変条件
root := entry.Skill.Root()           // 正本
assets := entry.Skill.Assets()       // 読み取り専用扱い
// 本体 Artifact: path = "skills/" + entry.Name + "/SKILL.md"
// sibling Artifact: path = "skills/" + entry.Name + "/" + assets[i].Path()
// 全 Artifact の SourceEntryIDs = [entry.ID]
// 全 Artifact の Mode はゼロ値
return artifacts, nil    // len(artifacts) >= 1 を保証
```

## 6. ManifestPathShapePolicy の表現位置

- 概念モデルで宣言した `ManifestPathShapePolicy` は **Go 側に独立した型を作らない**。
- 表現位置: `internal/source/schemas/manifest.schema.json` の中に閉じる（JSON Schema の `oneOf` / `allOf` + `if`/`then` 構造）。
- Go の Validator は当該 JSON Schema を読み込んで適用するだけ。Go 側の責務は schema を選ぶことに留まる。
- 設計書中で型の言及がない場合は「schema に閉じる」を意味する。

## 7. Boundary Decisions

| Boundary | Hidden detail | Reason |
|---|---|---|
| `SkillAsset` のコンストラクタ | パス正規化ロジック / SKILL.md 弾き | 値オブジェクト内に閉じる |
| `SkillMeta` のコンストラクタ | Path 重複検査ロジック | 値オブジェクト内に閉じる |
| `SkillAssetCollector`（unexported） | fs.WalkDir の使い方 / ファイル種別判定 / sort 順 | Loader 内部の戦略。外部 API に出さない |
| `source.Loader` | manifest 読込 → schema 検証 → entry 読込 → skill 解決 の順序と内部関数 | Loader 内部に閉じる |
| `internal/source` の sentinel 集合 | sentinel 値の作り方 (`errors.New` / `fmt.Errorf`) | 値そのものは公開、生成手段は隠す |

## 8. Interface Risks

### 8.1 Oversized interfaces

- なし。新規追加は値オブジェクト 2 種 + unexported コラボレータ 1 種 + sentinel 群のみ。

### 8.2 Primitive obsession

- SkillMeta.Root を string で持つことの primitive obsession 懸念があるが、既存 Entry.Path も string であり統一されている。refactoring-conceptual-model にある「ArtifactPath を値オブジェクト化」の流れに乗るなら将来的に PackRelativeDirPath や SkillAssetRelPath のような値オブジェクト導入可。本変更スコープ外（次の PR で導入する想定）。

### 8.3 Infrastructure leakage

- `SkillAssetCollector` は `fs.FS` を受け取るが、これは既存 Loader の API 規約（neutral 層が fs.FS を扱う）に従う。実装が `os` パッケージや git を直接触ることはない。

### 8.4 Boolean flag risks

- 新規 API に boolean フラグなし。SkillMeta コンストラクタも `IncludeHiddenFiles` 等の bool パラメータを持たない（要件で隠しファイル区別なしと定義済み）。

### 8.5 LSP risk

- KindRenderer の全実装が `[]Artifact` を返すよう揃えるため、呼び出し側の `len(arts) == 1` を前提とするテストがあれば壊れる。`append(arts...)` で集合扱いするテストに揃える必要あり。
- `Entry.Path` を skill では SkillMeta.Root の冗長コピーとして残す決定は、`Entry.Path` を「ファイル参照」と暗黙仮定する既存テスト・コードがあれば壊れる可能性。Loader 内部のみ参照と確認済み。
- 規約遵守の担保: 「skill 文脈で `Entry.Path` を直接読まない」というルールは linter では強制しない。後続 PR で Entry.Path 自体を `func (e Entry) LocationPath() string` などのメソッド化で隠蔽することは可能だが、本変更スコープ外。代わりに Test Specification 側で「skill renderer は `Entry.Skill.Root()` だけを参照する」ふるまいを明示テストする。

## 9. 順序と前提

### 8.1 manifest 全体の走査順序

- Loader は `manifest.entries` の **宣言順**（YAML 配列順）で 1 エントリずつ処理する。
- いずれかのエントリで失敗が起きた瞬間、それを最終 error として返し、後続エントリの処理は行わない（既存 Loader と同じ「fail-fast」セマンティクス）。
- 結果として、複数 skill で異常がある場合は **最初の異常エントリのエラー** が返る。テストは「N 番目で失敗する」ことに依拠してよい。

### 8.2 単一 entry 内のエラー優先順位（先勝ち）

manifest 全体 → 単一 entry の処理に入った後の優先順位:

1. manifest schema 違反（ManifestPathShapePolicy 含む。**entry 走査前に全体検証で確定**）
2. entry frontmatter schema 違反 / `ErrIDMismatch`
3. skill 用エントリで `ErrSkillPathNotFound`（path 不在）
4. skill 用エントリで `ErrSkillPathNotDirectory`（path がファイル）
5. skill 用エントリで `ErrSkillBodyNotFound`（SKILL.md 欠落）
6. SkillAsset/SkillMeta コンストラクタ系 (`ErrInvalidSkillAssetPath` 等)

非 skill エントリでは 3〜6 は発生せず、既存の `ErrEntryNotFound` 等が代わりに返る。

### 8.3 context cancel

各 entry 処理前に `ctx.Err()` をチェック（既存どおり）。skill 集合走査中の各ファイル読込前にもチェックする。

### 8.4 SkillAssetCollector のエラーラップ規約

- Collector 自身は fs.FS のエラーを **ラップしない**（透過で返す）。
- Loader 側で 1 回だけ `fmt.Errorf("source: collect skill assets %q: %w", root, err)` 形式でラップする。
- 結果として呼び出し側は `errors.Is` を多段で書く必要が無い（既存の loader エラー文字列パターンと一致）。
