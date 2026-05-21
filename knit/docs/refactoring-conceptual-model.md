# Refactoring Conceptual Model: 配布層共通化と値オブジェクト導入

このドキュメントは、knit の `internal/distribution/{claude,codex,gemini}` における Builder / Installer / Uninstaller / Lister の手続き重複と、`Artifact.Path` / `InventoryRoots` / `EntryID` / `InstallationID` の primitive obsession を解消するためのリファクタリングで導入する概念を整理する。

実装の手順・段階・コード詳細はこの文書には含めない。責務・関係・不変条件のみを記述する。

## Conceptual Model

### 既存概念 (継続)

| Concept | Meaning | State | Behavior | Constraint / Invariant |
|---|---|---|---|---|
| Pack | 1 単位のナレッジパック | Name / Version / Description / DefaultTools / Entries | EntriesFor(target) など | Name は kebab-case |
| Entry | パック内の個別ナレッジ | ID / Kind / Name / Description / Tags / Tools / Agent / Body | 自身の EntryID を提供 | ID は `<pack>.<kind>.<name>` |
| Artifact | target 固有の中間表現 | Target / Path / Content / Mode / SourceEntryIDs | (ステージ 1 で値オブジェクト経由化) | Path は Inventory root 相対の ArtifactPath |
| Installation | 配置済み Artifact + Label | InstallationID / Label / Provenance / Artifact | (ステージ 1 で生成を集約) | Label は non-zero。InstallationID は ArtifactPath から派生 |
| Provenance | Installation の出自 (どの Pack / どの Entry 群から来たか) | SourceEntryIDs / Pack 情報 | (ステージ 1 で Installation 経由のアクセスを集約) | Label と Installation 双方で複製しない (Label が正、Installation はビューを返す) |
| Label | knit 由来を示すメタデータ | Target / Version / Source URL / Provenance | IsZero | LabelStore で永続化。Provenance を内包する |
| Scope | 配布スコープ | user / project の列挙 | Valid / Validate / String | 2 値のみ |
| Target | 配布先 AI ツール識別 | claude / codex / gemini の文字列 | String | distribution パッケージごとに 1 つ |
| Kind | エントリ種別 | skill / agent / rule / prompt | IsValid / String | 4 値のみ |
| Inventory | (Scope, Target) 単位の Installation 集合 | Scope / Target / InventoryRoots / ArtifactWriter+Reader / LabelStore | (Transactional* 操作経由で参照・改変) | 「ある InstallationID に対しファイル有 ⇔ Label 有」を整合性条件とする (アトミック性はベストエフォート) |
| LabelStore | Label の永続化抽象 | (実装依存) | Set / Get / Delete / List | sidecar 実装が現状唯一 |

### 新規概念 (導入)

#### 値オブジェクト

| Concept | Meaning | State | Behavior | Constraint / Invariant |
|---|---|---|---|---|
| ArtifactPath | Inventory root に対する Artifact の相対パス | 単一の path 文字列 | TopSegment / Equal / Zero 判定 / 文字列化 | 構築時: 空 / 絶対 / `..` / NUL / バックスラッシュ含有を拒否 |
| AbsoluteArtifactPath | Inventory root と ArtifactPath を結合した絶対パス | 絶対パス文字列 (内部に root と相対を保持) | RelativePath() / Root() / String() | 構築時: root が絶対であること、結合結果が root を脱出しないこと |
| InventoryRoot | Inventory の絶対ルートパス | 絶対パス文字列 | Join(ArtifactPath) → AbsoluteArtifactPath | 構築時: 絶対パスかつ非空 |
| InventoryRoots | (user, project) の InventoryRoot 対 | userRoot (必須) / projectRoot (任意) | For(Scope) → InventoryRoot | userRoot は必須・絶対。projectRoot は省略可、要求された scope=project で未設定なら ErrProjectRootNotConfigured |
| EntryID | Entry の不変識別子 | `<pack>.<kind>.<name>` 文字列 | Pack / Kind / Name 取り出し / Equal / Zero 判定 / 文字列化 | 構築時にパターン適合 (kebab-case Pack + 有効 Kind + kebab-case Name) を検証 |
| InstallationID | ArtifactPath から派生する Installation の不透明識別子 | ArtifactPath 由来の文字列 | EncodedBaseName / DecodedBaseName 規約をメソッドとして所有 | ArtifactPath からの派生関数のみで構築。エンコード往復で同一性が保たれる |

#### 戦略・ポリシー

| Concept | Meaning | State | Behavior | Constraint / Invariant |
|---|---|---|---|---|
| PathPolicy | target が許容する Artifact path 規約 (データ宣言) | 許容トップディレクトリ集合 / 特例ファイル名集合 / user・project ルートのサブディレクトリ規約 | Validate(ArtifactPath) | target ごとに 1 インスタンス。データのみで状態遷移を持たない |
| PathResolver | PathPolicy + InventoryRoots + Scope から AbsoluteArtifactPath を組み立てるリゾルバ | PathPolicy / InventoryRoots | Resolve(Scope, ArtifactPath) → AbsoluteArtifactPath | InventoryRoot を root として ArtifactPath を Join する。root 脱出を拒否 |
| KindRenderer | 単一 Kind の Entry を target 固有 Artifact に変換 | 自身が受け持つ Kind / target 識別子 | Render(Entry, Pack) → Artifact | 出力 ArtifactPath は PathPolicy を満たす |
| RuleAggregator | 複数の rule kind Entry を 1 つの Artifact に集約 | target 識別子 / 集約ヘッダ規約 | Aggregate(\[]Entry, Pack) → Artifact | 集約結果は単一 Artifact (target ごとの単一ファイル名)。frontmatter 衝突は契約違反 |
| RendererRegistry | Kind → Renderer (or Aggregator) のディスパッチ表 | Kind ごとの登録エントリ | RendererFor(Kind) → KindRenderer または RuleAggregator | Builder の `switch e.Kind` を不要にする |

#### サービス (inventory 配下)

| Concept | Meaning | State | Behavior | Constraint / Invariant |
|---|---|---|---|---|
| ArtifactReader | ファイル存在判定の抽象 | (実装依存) | Exists(AbsoluteArtifactPath) | 副作用を持たない |
| ArtifactWriter | ファイル書き込み・削除・空親ディレクトリ整理の抽象 | (実装依存) | Write(AbsoluteArtifactPath, content, mode) / Remove(AbsoluteArtifactPath) / PruneAncestorsWithin(child AbsoluteArtifactPath, boundary InventoryRoot) | 冪等性を保証。`PruneAncestorsWithin` は boundary を越えて削除しない (型で boundary を表現) |
| ArtifactStore | ArtifactReader と ArtifactWriter を埋め込んだ便宜エイリアス | (実装依存) | Reader と Writer の合成 | テスト時に Reader だけ・Writer だけを差し替え可能 |
| TransactionalInstaller | Artifact + Label の二段書き込みとロールバックを所有する操作型 | ArtifactStore / LabelStore / PathResolver / Target | Install(ctx, Scope, Artifact, InstallationID) | Inventory 整合性条件を維持する (中間状態を残さない) |
| TransactionalUninstaller | Installation の安全な除去を所有する操作型 | 同上 | Uninstall(ctx, Scope, Installation) | Inventory 整合性条件を維持する |
| TransactionalLister | Inventory から Installation 集合を構築する操作型 | LabelStore / ArtifactReader / PathResolver / Target | List(ctx, Scope) | orphan label (ファイルが消失した label) は除外する |

## Relationships

| Concept A | Relationship | Concept B | Notes |
|---|---|---|---|
| Pack | 1..n | Entry | 既存 |
| Entry | derives | EntryID | EntryID は Entry の identity |
| Entry | renders to | Artifact | KindRenderer 経由 |
| Pack | 1..n | rule Entry の集約 → Artifact | RuleAggregator 経由 |
| Artifact | has | ArtifactPath | Artifact.Path を値オブジェクト化 |
| Artifact | has | Target | 配布先を示す |
| InventoryRoot | 1 | (user か project の) Inventory ルート絶対パス |  |
| InventoryRoots | aggregates | 2 つの InventoryRoot |  |
| InventoryRoot + ArtifactPath | yields | AbsoluteArtifactPath | Join 操作 |
| ArtifactPath | derives | InstallationID | 一方向。InstallationID から元の ArtifactPath を取り出せる |
| Inventory | identified by | (Scope, Target) | (Scope, Target) は Inventory の identity |
| Inventory | contains | 0..n Installation |  |
| Installation | identified by | InstallationID |  |
| Installation | composed of | Label / Provenance (Label 由来のビュー) / Artifact |  |
| Target | 1—1 | PathPolicy | target ごとに 1 つ |
| Target | 1—n | KindRenderer | Kind × Target の組み合わせごとに 1 つ |
| Target | 0..1 | RuleAggregator | rule kind を持つ target のみ |
| PathPolicy + InventoryRoots | composes | PathResolver |  |
| Builder (target) | owns | RendererRegistry | Kind → Renderer のマッピングを保持 |
| Builder (target) | walks | Pack (Entries) → \[]Artifact | Builder は走査と RendererRegistry の参照のみ |
| TransactionalInstaller / Uninstaller / Lister | depends on | ArtifactStore (or Reader/Writer のサブセット) + LabelStore + PathResolver + Target | DIP |
| TransactionalLister | depends on | ArtifactReader のみ (書き込み不要) |  |
| distribution/<target> | provides | PathPolicy + KindRenderer 群 + RuleAggregator | target 固有のデータ宣言のみ |
| distribution/<target> | exposes | Builder/Installer/Uninstaller/Lister を構築するコンストラクタ群 | 各コンストラクタは Transactional* を返し、`inventory.Installer/Uninstaller/Lister` interface をそのまま満たす |

## Builder の責務縮小

新モデルにおける Builder (target) の責務:

| 項目 | 内容 |
|---|---|
| State | RendererRegistry / RuleAggregator (rule を持つ target のみ) / Target |
| Behavior | Build(Pack) → \[]Artifact: Pack の Entries を走査し、Kind に対応する Renderer か RuleAggregator にディスパッチ |
| Invariant | Builder は `switch e.Kind` を持たない (RendererRegistry を通じてのみ Kind を解釈) |

これにより Builder は target ごとの薄いオーケストレータになる。Kind に対する手続きは KindRenderer / RuleAggregator に集約される。

## Provenance の所有

| 観点 | 内容 |
|---|---|
| 主体 | Label が Provenance を所有する (LabelStore で永続化される) |
| Installation 側 | `Installation.Provenance` は Label.Provenance のビュー (派生プロパティ)。重複格納しない |
| SourceEntryIDs | Provenance に含まれる。EntryID 値オブジェクトとして扱う (cli/helpers.go の `neutralIDPack` を Provenance のメソッド `Packs() []string` 等で置き換える) |

## Inventory 整合性条件

| 状態 (Label, File) | 解釈 | 操作の責務 |
|---|---|---|
| (有, 有) | 正常な Installation。List で返る | Installer 完了状態 / Uninstaller の前提 |
| (有, 無) | orphan label。File が外部削除された後の状態 | Lister は除外。Uninstaller の Label.Delete のみ実行 |
| (無, 有) | unmanaged file。knit が触れない外部ファイル | Installer の preflight で ErrUnmanagedArtifactExists |
| (無, 無) | 未インストール | Installer の正常開始状態 |

この 2×2 状態表が Installer/Uninstaller/Lister の preflight 判定の正本。詳細な sentinel 対応と操作順序は interface design に記述する。

## Structural Risks

### Missing concepts (現状で欠落している概念) — 本リファクタで解消

- **Inventory**: 「(Scope, Target) の Installation 集合」が型として存在せず、Installer/Uninstaller/Lister の依存集合と整合性条件が手続きの行間に埋もれている。
- **ArtifactReader / ArtifactWriter**: ファイル I/O が target ごとに 3 回ハードコードされ、Lister も Installer も同じ `os.*` を直接呼ぶ。
- **PathPolicy / PathResolver**: 各 target の `pathresolver.go` に if 連鎖で埋もれる規約と、scope ディスパッチが混在。
- **RendererRegistry / KindRenderer / RuleAggregator**: Builder 内の `switch e.Kind` が 3 target で複製。
- **AbsoluteArtifactPath**: ArtifactPath を導入してもファイル境界で string に戻ると primitive obsession が再発する。
- **Provenance (独立化)**: 現状 Label と Installation の両方にフィールドが分散。

### Hidden state (隠れた状態) — 本リファクタで解消

- Installer のトランザクション順序が手続き分散 → 「Inventory 整合性条件」と 2×2 状態表で宣言。
- ArtifactPath の不変条件が doc コメントのみ → 値オブジェクトのコンストラクタで型レベル担保。

### Change-prone areas (本リファクタで縮小)

- 新 target 追加: 現状 6〜12 箇所 → distribution/<target> パッケージ 1 つに集約 (PathPolicy + Renderer 群 + 公開コンストラクタ)。
- 新 Kind 追加: 現状 3 target の Builder 全部 → RendererRegistry に Renderer を登録するだけ。

### Boundary candidates (本リファクタで確定)

- **inventory ↔ filesystem**: ArtifactReader / ArtifactWriter を inventory 側 interface として宣言、os 実装を `internal/inventory/fsstore`(または同等の場所) に置く。
- **distribution ↔ inventory**: distribution は PathPolicy + Renderer のデータ宣言と、Transactional* を組み立てるコンストラクタのみを公開する。Installer/Uninstaller/Lister の手続きは inventory 側に集約される。
- **distribution 内部の Builder ↔ Renderer**: Builder は RendererRegistry を参照するだけになる。

## 不変条件一覧 (型レベルで守るべきもの)

| 不変条件 | 守る型 |
|---|---|
| ArtifactPath は空でない・絶対でない・`..` を含まない・NUL/バックスラッシュを含まない | ArtifactPath |
| AbsoluteArtifactPath は root の配下に収まる | AbsoluteArtifactPath |
| InventoryRoot は絶対パスかつ非空 | InventoryRoot |
| InventoryRoots.projectRoot は scope=project 要求時のみ要求 | InventoryRoots |
| EntryID は (kebab-case Pack, Kind 4 値, kebab-case Name) の 3 段ドット区切り | EntryID |
| InstallationID は ArtifactPath からの派生のみで構築可。エンコード往復で同一性が保たれる | InstallationID |
| Installation.Label は non-zero | Installation (コンストラクタで担保) |
| Provenance は Label が正本、Installation はビューを返す | Label, Installation |
| Inventory 整合性条件: ある InstallationID に対し Label 有 ⇔ File 有 (操作後) | TransactionalInstaller / Uninstaller |
| PruneAncestorsWithin は boundary を越えて削除しない | ArtifactWriter |
| Builder は switch e.Kind を持たない | Builder (各 target 実装) |

## 非ゴール

- LabelStore 実装の差し替え (xattr 等) は範囲外。本リファクタは「配布層の重複解消に直結する ArtifactStore 抽象化」のみを範囲とする。
- Loader / Validator / フロントマター分割の YAML 結合解消は範囲外。
- CLI 層 (cmd_*.go) のユースケース層切り出しは範囲外 (配布層 API が安定した後に別 PR)。
- 新規 target 追加は範囲外。

範囲選別の基準: 「本 PR は配布層の手続き重複と値オブジェクト欠落を解消する」。それ以外の改善 (LabelStore 抽象, Loader 分離, CLI usecase 層) は同じ理由で改善余地があるが、本 PR では扱わない。
