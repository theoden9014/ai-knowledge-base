# Refactoring Interface Design: 配布層共通化と値オブジェクト

> Historical design record. It predates the two-kind model and is not
> normative. See [`concept.md`](concept.md) and the current code interfaces.

このドキュメントは概念モデル ([refactoring-conceptual-model.md](./refactoring-conceptual-model.md)) で定義した新概念のインタフェース設計を、責務と契約レベルで記述する。具体的なシグネチャや実装コードは含めない (TDD ステージで決定する)。

## Proposed Interfaces / Responsibilities

### 値オブジェクト

| 型 | Consumer | 責務 | Inputs / Outputs (概念) | Error Contract |
|---|---|---|---|---|
| ArtifactPath | distribution Builder / inventory Transactional 群 / cli ArtifactWriter | Inventory root に対する相対パスを保持し、自身の不変条件 (空・絶対・`..`・NUL・バックスラッシュ不可) を所有する | 構築入力: 文字列。提供操作: トップセグメント取得、等価判定、Zero 判定、文字列化 | 構築不能入力に対し ErrInvalidArtifactPath |
| AbsoluteArtifactPath | inventory Transactional 群 / ArtifactStore | InventoryRoot と ArtifactPath を結合した絶対パスを保持する | 構築入力: InventoryRoot + ArtifactPath。提供操作: RelativePath / Root / 文字列化 | root 脱出を検出した場合 ErrArtifactPathEscape |
| InventoryRoot | InventoryRoots / PathResolver / ArtifactWriter (boundary 引数) | Inventory の絶対ルートパスを保持する | 構築入力: 絶対パス文字列。提供操作: Join(ArtifactPath) → AbsoluteArtifactPath | 相対パス・空文字列を拒否 ErrInvalidInventoryRoot |
| InventoryRoots | PathResolver | (user, project) 対を保持し、scope から InventoryRoot を返す | 構築入力: userRoot (必須) / projectRoot (任意)。提供操作: For(Scope) → InventoryRoot | scope=project かつ projectRoot 空のとき ErrProjectRootNotConfigured |
| EntryID | source / inventory (Provenance) / cli / distribution Builder | `<pack>.<kind>.<name>` を保持し、Pack / Kind / Name の分解を提供する | 構築入力: 文字列。提供操作: Pack / Kind / Name / Equal / Zero / 文字列化 | 構築不能入力に対し ErrInvalidEntryID |
| InstallationID | inventory LabelStore / Transactional 群 | ArtifactPath から派生する不透明識別子。Sidecar 用のエンコード/デコード規約を所有する | 構築は ArtifactPath 派生関数 (`FromArtifactPath(ArtifactPath)`) のみ public。文字列直接構築は内部限定 | エンコード往復で同一性が壊れる入力を拒否 ErrInvalidInstallationID |

### 戦略型 (distribution パッケージ提供)

| 型 | Consumer | 責務 | Inputs / Outputs (概念) | Error Contract |
|---|---|---|---|---|
| PathPolicy | PathResolver / Builder / Transactional 群 | ArtifactPath の意味的妥当性 (許容トップディレクトリ・特例ファイル名) を判定する。データのみ。 | 操作: Validate(ArtifactPath) → ok or error | 規約違反時 ErrInvalidArtifactPath |
| PathResolver | TransactionalInstaller / Uninstaller / Lister | PathPolicy + InventoryRoots を組み合わせ、Scope と ArtifactPath を AbsoluteArtifactPath に解決する | 操作: Resolve(Scope, ArtifactPath) → AbsoluteArtifactPath / ResolveRoot(Scope) → InventoryRoot | ErrInvalidArtifactPath / ErrProjectRootNotConfigured / ErrInvalidScope |
| KindRenderer | distribution Builder / RendererRegistry | 単一 Kind の Entry を target 固有 Artifact に変換する | 操作: Render(Entry, Pack) → Artifact | フロントマター衝突は ErrFrontmatterMergeConflict、unsupported value は ErrUnsupportedFrontmatterValue |
| RuleAggregator | distribution Builder | 複数の rule kind Entry を 1 つの Artifact に集約する | 操作: Aggregate(\[]Entry, Pack) → Artifact | rule に frontmatter を許さない target で `tools[target].frontmatter` 非空のとき ErrFrontmatterMergeConflict |
| RendererRegistry | distribution Builder | Kind → KindRenderer / RuleAggregator のディスパッチを所有する | 操作: RendererFor(Kind) → KindRenderer か RuleAggregator か unsupported | unsupported Kind に対し ErrUnsupportedKind |

### サービス型 (inventory パッケージ提供)

| 型 | Consumer | 責務 | Inputs / Outputs (概念) | Error Contract |
|---|---|---|---|---|
| ArtifactReader | TransactionalInstaller (preflight 用) / TransactionalLister | ファイルの存在判定を提供する | 操作: Exists(AbsoluteArtifactPath) → bool | os エラーをそのまま伝搬 |
| ArtifactWriter | TransactionalInstaller / TransactionalUninstaller | ファイル書き込み・削除・空親ディレクトリの整理を提供する | 操作: Write(AbsoluteArtifactPath, content, mode) / Remove(AbsoluteArtifactPath) / PruneAncestorsWithin(child AbsoluteArtifactPath, boundary InventoryRoot) | os エラーをそのまま伝搬。PruneAncestorsWithin は boundary 超過時 ErrPruneBoundaryViolation |
| ArtifactStore | (便宜エイリアス) | ArtifactReader + ArtifactWriter の合成 | 上記の両方 | 各メソッドに同じ |
| TransactionalInstaller | distribution/<target> の `NewInstaller` | Artifact + Label の二段書き込みとロールバックを所有する | 構築時束縛: ArtifactStore + LabelStore + PathResolver + Target。操作: Install(ctx, Scope, Artifact) → Installation | Inventory 整合性条件を維持する |
| TransactionalUninstaller | distribution/<target> の `NewUninstaller` | Installation の安全な除去 | 構築時束縛: 同上。操作: Uninstall(ctx, Scope, Installation) | 同上 |
| TransactionalLister | distribution/<target> の `NewLister` | LabelStore から Installation 集合を構築し orphan を除外する | 構築時束縛: LabelStore + ArtifactReader + PathResolver + Target。操作: List(ctx, Scope) → \[]Installation | scope=project 要求時に projectRoot 不在なら ErrProjectRootNotConfigured |

## TransactionalInstaller の責務分解

レビュー指摘 (Installer の責務肥大) を踏まえ、Install 自身は核トランザクションのみを所有する。target/scope の検証は構築時束縛 + PathResolver が担う。

| ステージ | 責務の所有者 | 内容 |
|---|---|---|
| Build-time | TransactionalInstaller の構築 | Target を構築時に束縛 (Install 引数では受け取らない) |
| Build-time | PathResolver | InventoryRoots + PathPolicy を構築時に束縛 |
| Call-time | PathResolver.Resolve | Scope + ArtifactPath → AbsoluteArtifactPath を返す (scope/root/path 検証はここで完結) |
| Call-time | TransactionalInstaller.Install | preflight (LabelStore.Get + ArtifactReader.Exists) → ArtifactWriter.Write → LabelStore.Set → 失敗時 ArtifactWriter.Remove (ロールバック) |
| Call-time | TransactionalUninstaller.Uninstall | 同様の核トランザクション (preflight → file delete → label delete → PruneAncestorsWithin) |
| Call-time | TransactionalLister.List | scope/root 解決 → LabelStore.List → ArtifactReader.Exists で orphan 除外 → Installation 構築 |

Lister は構築時に Target を束縛しており、操作引数として target を受け取らない。よって `ErrTargetMismatch` は Lister の責務範囲外 (Installer / Uninstaller のみ)。

## preflight 状態判定 (Installer)

`(Label, File)` の状態に対する Installer の判定:

| Label | File | 判定 | sentinel | 続行 |
|---|---|---|---|---|
| 無 | 無 | 正常 (未インストール) | — | Write + Label.Set へ進む |
| 無 | 有 | unmanaged file が存在 | ErrUnmanagedArtifactExists | 中止 |
| 有 | 無 | orphan label が残存 | ErrAlreadyInstalled (or 専用 sentinel; 実装側で判断) | 中止 (ユーザーが意図的に削除した可能性があるため) |
| 有 | 有 | すでに installed | ErrAlreadyInstalled | 中止 |

Uninstaller の判定:

| Label | File | 判定 | sentinel | 動作 |
|---|---|---|---|---|
| 無 | * | 該当 Installation 不在 | ErrInstallationNotFound | 中止 |
| 有 | 有 | 正常 Installation | — | file delete → label delete → prune |
| 有 | 無 | orphan label | — | label delete のみ実行し、ファイル操作はスキップ (PruneAncestorsWithin も呼ばない) |

Lister の判定:

| Label | File | 動作 |
|---|---|---|
| 無 | * | (列挙経路に来ない) |
| 有 | 有 | Installation を返す |
| 有 | 無 | orphan として除外 (warn は呼び出し側の責務、最小実装では暗黙除外) |

## Boundary Decisions

| Boundary | 隠す詳細 | 理由 |
|---|---|---|
| ArtifactReader / ArtifactWriter | `os.MkdirAll` / `os.WriteFile` / `os.Remove` / 親ディレクトリ削除のロジック | DIP。Transactional 群が `os` を直接触らない |
| ArtifactWriter.PruneAncestorsWithin の境界 | InventoryRoot を boundary 引数として型表現し、boundary 超過削除を構造的に禁止 | レビュー Critical 指摘 (root を超えて削除する事故を構造で防ぐ) |
| PathPolicy / PathResolver の分離 | データ宣言と scope ディスパッチを別概念に分ける | 変更理由の分離 (target 規約 vs scope) |
| KindRenderer / RuleAggregator / RendererRegistry | Builder の `switch e.Kind` を消し、Kind ごとの責務を独立した型に集約 | OCP / SRP |
| ArtifactPath / AbsoluteArtifactPath | path 文字列の不変条件と root 脱出禁止を型レベルで担保 | primitive obsession の解消 |
| InventoryRoot / InventoryRoots | (userRoot, projectRoot) 引数取り違え事故を構造で禁止 | 同上 |
| EntryID | `<pack>.<kind>.<name>` パース規約 | cli/helpers.go の `neutralIDPack` と claude builder の `neutralIDToShortName` の二重実装統合 |
| InstallationID | エンコード規約は ID の不変条件であり、ID 以外から参照されてはならない | 責務凝集 |

## 既存 interface の扱い

| 既存 interface / 型 | 扱い | 理由 |
|---|---|---|
| `source.Loader` / `source.Validator` | 変更しない | 範囲外 |
| `source.Builder` (target 横断) | シグネチャ維持。実装が RendererRegistry を保持し Pack を走査するだけになる | consumer 影響を抑制 |
| `source.Pack` | 変更なし | 範囲外 |
| `source.Entry` | ID は文字列フィールドのまま維持し、必要箇所で `source.NewEntryID` を使う | 未使用 API を増やさない |
| `source.Artifact` | Path は互換性のため string のまま維持し、inventory 境界で ArtifactPath に変換 | consumer 影響を抑制 |
| `inventory.Installer/Uninstaller/Lister` (interface) | **interface は維持し、TransactionalInstaller/Uninstaller/Lister を各 distribution の薄いラッパーから利用する** | target 固有 sentinel 変換を distribution 境界に残す |
| `inventory.LabelStore` | 変更しない | 範囲外 |
| `inventory.Installation` | コンストラクタ経由を public な構築規約に変更し、ゼロ値構築を抑止。`Provenance` は Label 由来のビューを返すメソッドに |  |
| `inventory.InstallationID` | 文字列ベースは維持。`FromArtifactPath` コンストラクタ + EncodedBaseName/DecodedBaseName メソッドを追加 |  |
| `inventory.Scope` | 変更しない |  |
| 各 distribution の `NewBuilder` / `NewInstaller` / `NewUninstaller` / `NewLister` | シグネチャ互換を可能な限り維持しつつ、内部実装を Transactional* と PathResolver + Renderer の組み立てに差し替え | factory.go の修正最小化 |
| 各 distribution の `Err*` sentinel | 当面維持 | 共通化は別 PR |

## 値オブジェクトの構築規約

### 構築時の不変条件 (コンストラクタが拒否する入力)

| 型 | 拒否する入力 | 検証順序 |
|---|---|---|
| ArtifactPath | 空 / 絶対 / `..` / NUL / バックスラッシュ含有 | 空 → 絶対 → 危険文字 → `..` の順で早期 return |
| InventoryRoot | 空 / 相対 | 空 → 絶対判定 |
| InventoryRoots | userRoot が空・相対 / projectRoot が空文字列を渡された場合 (省略=未設定とは異なる) | userRoot 先行検証 |
| AbsoluteArtifactPath | root が絶対でない / 結合結果が root を脱出 | root 検証 → 結合 → 正規化 → 脱出検出 |
| EntryID | パターン不一致 | パッケージ名・Kind・エントリ名の構成要素を順に検査 (正規表現は実装詳細として実装段階で決定) |
| InstallationID | ArtifactPath 派生でない / エンコード往復で同一性が崩れる | `FromArtifactPath` のみ public |

### 操作時の前提条件

| 型 | 前提 | 違反時 |
|---|---|---|
| InventoryRoots.For(scope) | scope が user か project | ErrInvalidScope |
| InventoryRoots.For(project) | projectRoot が設定されている | ErrProjectRootNotConfigured |
| PathResolver.Resolve(scope, p) | scope が valid、p が PathPolicy.Validate に合格 | 上記 + ErrInvalidArtifactPath |

## Error Contract

### sentinel の所属

| sentinel | 発生位置 |
|---|---|
| ErrInvalidArtifactPath | ArtifactPath コンストラクタ / PathPolicy.Validate |
| ErrArtifactPathEscape | AbsoluteArtifactPath 構築 |
| ErrInvalidInventoryRoot | InventoryRoot コンストラクタ |
| ErrInvalidEntryID | EntryID コンストラクタ |
| ErrInvalidInstallationID | InstallationID 構築 |
| ErrProjectRootNotConfigured | InventoryRoots.For(project) |
| ErrInvalidScope | InventoryRoots.For / PathResolver |
| ErrTargetMismatch | TransactionalInstaller / Uninstaller (Lister は対象外) |
| ErrAlreadyInstalled | TransactionalInstaller preflight |
| ErrUnmanagedArtifactExists | TransactionalInstaller preflight |
| ErrInstallationNotFound | TransactionalUninstaller |
| ErrFrontmatterMergeConflict | KindRenderer / RuleAggregator |
| ErrUnsupportedFrontmatterValue | KindRenderer (gemini prompt 等) |
| ErrUnsupportedKind | RendererRegistry |
| ErrPruneBoundaryViolation | ArtifactWriter.PruneAncestorsWithin |

### Installer / Uninstaller / Lister の検証順序

| ステージ | Installer | Uninstaller | Lister |
|---|---|---|---|
| 1 | target 検証 (Artifact.Target が束縛 target と一致) | target 検証 (Installation.Label.Target が束縛 target と一致) | (target は構築時束縛のみ。引数なし) |
| 2 | scope 検証 + InventoryRoot 解決 | scope 検証 + InventoryRoot 解決 | scope 検証 + InventoryRoot 解決 |
| 3 | ArtifactPath validate + AbsoluteArtifactPath 構築 | ArtifactPath validate + AbsoluteArtifactPath 構築 | LabelStore.List → 各 Label の ArtifactPath を validate + 構築 |
| 4 | preflight 2x2 状態判定 | preflight 2x2 状態判定 | 各 Installation の File 存在を確認し orphan 除外 |
| 5 | Write → Label.Set → 失敗時 Rollback | Remove → Label.Delete → PruneAncestorsWithin | (該当なし) |

## 依存方向

```
source (純粋ドメイン)            ←── distribution/<target>
   ↑                                       ↑
   │                                       │
   │                          (PathPolicy / Renderer 群)
   │                                       │
inventory (永続化 + Transactional)          │
   │                                       │
   │  Reader/Writer/PathResolver ─────────┐│
   ▼                                      ▼▼
ArtifactReader/Writer 実装 ──→ os / io/fs
```

- inventory が ArtifactReader / ArtifactWriter / PathResolver / Transactional 群を提供。
- distribution/<target> は PathPolicy + Renderer 群 + 公開コンストラクタの薄いパッケージになる。
- cli は LabelStore を組み立て、distribution の各コンストラクタを呼ぶだけ。

## ISP / SRP / DIP / OCP に対する設計上の意図

- ArtifactReader と ArtifactWriter を最初から分離 (ISP)。Lister は Reader にのみ依存することで contract test の意図 ("読み専用") が明確になる。
- TransactionalInstaller / Uninstaller / Lister は分離 (SRP)。共有依存集合を持つが、トランザクション境界が異なる。
- PathPolicy (データ) と PathResolver (組立) を分離 (SRP)。変更理由が異なる。
- ArtifactStore (の実装) のみが `os` / `io/fs` に依存 (DIP)。core は infra 非依存。
- RendererRegistry の導入で Builder は OCP に近づく (新 Kind の追加は Renderer 登録のみ)。

## テスト可能性への影響

- ArtifactReader / ArtifactWriter のメモリ実装で Transactional 群の単体テストが `os.TempDir` 不要に。
- KindRenderer は単体テストでケース表を埋めれば target 差分を網羅できる。
- 既存の `inventorytest` contract test 群は無変更で再利用 (Transactional 群が contract を満たす実装になる)。
- PathPolicy はデータ宣言なのでテーブル駆動テストで網羅。

## 非ゴール (再掲)

- LabelStore 実装の差し替え可能性は範囲外。
- Loader / Validator の YAML 結合解消は範囲外。
- CLI 層のユースケース層切り出しは範囲外。
- 新規 target 追加は範囲外。
