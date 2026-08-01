# Skill Directory Unit - Responsibility Assignment

> Historical design record for the original directory migration. Current
> responsibility boundaries are documented in
> [`../../docs/authoring-guidelines.md`](../../docs/authoring-guidelines.md).

要件: [skill-directory-requirements.md](./skill-directory-requirements.md)
概念モデル: [skill-directory-conceptual-model.md](./skill-directory-conceptual-model.md)
既存責務分担: [refactoring-conceptual-model.md](./refactoring-conceptual-model.md), [concept.md](./concept.md), [modules.md](./modules.md)

## 1. パッケージ境界（変更なしの確認）

既存 modules.md の依存方向は **本変更で変えない**。

```
cli ──> (source, inventory, distribution)
distribution/<target> ──> (source, inventory)
inventory ──> source
source ──> (nothing)
```

新規概念の置き場は次の通り。いずれも既存境界を越えない。

| 新規概念 | パッケージ | 理由 |
|---|---|---|
| SkillAsset | `internal/source` | Entry に紐づく値オブジェクト。Pack/Entry と同じ neutral 層 |
| SkillMeta | `internal/source` | Entry のサブ構造（AgentMeta と同居） |
| SkillRootResolution（エラー sentinel 群） | `internal/source` | Loader の戻り側に出るため source に置く |
| SkillAssetCollector | `internal/source` 内部（unexported） | Loader 内部の協力者。外部公開しない |
| ManifestPathShapePolicy | `internal/source/schemas/manifest.schema.json`（schema 表現） | schema 層に閉じる。Go 側に追加型は作らない（validator が JSON Schema 経由で強制） |

## 2. Responsibility Assignment

| Responsibility | Owner | Reason to change | SOLID concern | Not owner | Reason |
|---|---|---|---|---|---|
| manifest 上の `id.kind` と `path` 形状の整合を検証 | `manifest.schema.json` + `source.Validator` | schema が更新された時、または新 kind が追加された時 | SRP: schema 規約は schema 層に閉じる | Go 側の独立型を作らない | 過剰な抽象化を避ける。JSON Schema で完結する規約 |
| skill ルートのディレクトリ実在検証 | `source.Loader` | skill 解決の異常 3 種（path 不在 / path がファイル / SKILL.md 欠落）の取り扱いが変わった時 | SRP: 入力解釈は Loader が担う | Validator ではない | Validator は manifest 文書の静的検査担当。FS 走査は Loader |
| skill 配下の通常ファイル走査と本体除外 | `SkillAssetCollector`（Loader 内部） | 走査ポリシ（シンボリックリンク・隠しファイル等）の方針が変わった時 | SRP: 走査ロジックは Loader 本体から分離して 1 箇所に固める | Loader 本体ではない | Loader 本体は manifest entry → Entry 値オブジェクト変換のオーケストレータに留める |
| Entry.Skill (SkillMeta) の構築 | `source.Loader` | skill 専用フィールドの追加・除去時 | SRP: Entry の identity 構築は Loader | Renderer ではない | Renderer は読み取り専用 |
| `Entry.Skill` の Kind 連動 nil 不変条件（Kind=KindSkill のとき非 nil、それ以外で nil）の保証 | `source.Entry` の構築側（コンストラクタまたは Loader 内 Entry 組立点） | Entry のサブ構造種別が増減した時 | LSP, ISP: agent/prompt/rule renderer は Entry.Skill を読まない契約を型レベルで担保 | Renderer ではない | Renderer に分岐ロジックを書かない |
| `Entry.Path` の意味（pack 相対の場所参照）と doc コメント更新 | `source.Entry` の型宣言と doc | skill 用途でディレクトリ指し示しになる方針が確定した時 | LSP（暗黙契約の変更を doc で吸収） | Loader ではない | Loader は値を代入するだけ |
| skill 時の `Entry.Path == SkillMeta.Root` 同期不変条件 | `source.Loader` での Entry 組立点 | skill ルートの正本が SkillMeta.Root であると確定した時 | LSP: 2 つの場所に同じ値があるなら同期は構築点に閉じる | Renderer ではない | Renderer は SkillMeta.Root のみ参照する規約 |
| SkillAsset の不変条件保持（path 正規化・SKILL.md でない等） | `SkillAsset`（コンストラクタ） | 不変条件の追加・緩和時 | SRP, LSP: 値オブジェクト自身が不変条件を保証 | Loader ではない | Loader は構築呼び出しに留め、検証ロジックを内部に持たない |
| SkillMeta の不変条件保持（Root の形・Assets 重複なし） | `SkillMeta`（コンストラクタ） | 同上 | SRP, LSP | Loader ではない | 同上 |
| SkillRootResolution の **内部状態列挙**（Resolved / PathMissing / PathIsFile / BodyMissing の 4 値）の判定 | `source.Loader` の内部関数 | 異常分類が増減した時 | SRP: skill 解決の状態判定ロジックを 1 箇所に集める | （外部公開しない） | 列挙自体は loader 内部の中間表現で十分 |
| SkillRootResolution の **外部 sentinel**（PathMissing/PathIsFile/BodyMissing に対応する 3 種の error 値） | `source` パッケージのエラー定義 | 異常分類が増減した時 | ISP: 呼び出し側は `errors.Is` で必要分のみ識別 | Loader 本体ではない | sentinel 自体は静的データ。Loader は内部状態列挙からマップして返すだけ |
| Entry.Kind ごとの Artifact 生成方針の切替 | `source.RendererRegistry` | 新 kind 追加・既存 kind の renderer 差し替え時 | OCP: registry に renderer を登録するだけで拡張可 | Builder ではない | Builder は registry に委譲 |
| KindRenderer の Render 戻り型を `[]Artifact` に拡張 | `source.KindRenderer`（interface） + 各 target の renderer | skill が複数ファイルを生む方針に変わった時（今回） | OCP, LSP: 全 renderer が一律のシグネチャを実装。1 件返す renderer も `[]Artifact{art}` に統一 | RuleAggregator は対象外 | RuleAggregator は別契約（複数 Entry → 1 Artifact）で意味が異なる |
| skill 用 renderer が SkillMeta.Assets から sibling Artifact 群を生成 | `distribution/<target>/skill_renderer.go`（既存ファイル） | sibling artifact 生成規約が変わった時 | SRP: target 固有の path 規約と一緒に存在 | source 層ではない | target 固有の path 形式は distribution 担当 |
| skill 由来 Artifact の `SourceEntryIDs` 整合（全 Artifact 同一値） | 同 skill_renderer | 同上 | LSP: renderer が契約として保証 | Builder ではない | Builder は append するだけ |
| sibling Artifact の `Mode` をゼロ値に統一 | 同 skill_renderer | レギュラーファイル既定の方針が変わった時 | SRP | inventory ではない | Installer は受け取った Mode を尊重する |
| target 配置 path の組み立て（`skills/<name>/<subpath>`） | 同 skill_renderer | target 固有の skills 規約が変わった時 | SRP, DIP: distribution 層が target 固有規約を所有 | source ではない | 既存責務と同じ |
| codex の path policy を `skills/<name>/<subpath>` 任意階層対応に拡張 | `distribution/codex/path_policy.go`（`validateSkillPath` の改修） | target の skills 配置規約が変わる時 | SRP | source ではない | path policy は target 固有 |
| claude / gemini の path policy は現状で `skills/<...>` のサブパス全般を許容しており追加変更なし（確認済み） | （変更不要） | — | — | — | — |
| Inventory への配置・整合性維持（label + file の 2 段書き込み） | `inventory.TransactionalInstaller`（既存、変更不要） | 整合性条件 (Label, File) の 2×2 規約が変わった時 | DIP, SRP | distribution ではない | inventory 層が neutral に担う |
| pack 指定 uninstall で sibling まで一括削除 | `cli.cmd_uninstall` 経由 `inventory.Provenance.BelongsToPack` | uninstall の集合判定規約が変わった時 | SRP, OCP: Provenance の動的ビューで実現済み | 新規概念 SkillCohort を作らない | 集合は永続化されたファーストクラスではなく動的ビューで足りる（概念モデル §5.2） |
| Reinstaller が sibling 増減に破綻しないこと（最低保証） | `inventory.reinstaller`（既存挙動の範囲） | reinstaller 仕様が変わった時 | LSP | 本変更で規範化しない | sibling 増減仕様は R7 で本変更スコープ外 |
| 既存 manifest.yaml の新形式書き換え | `knowledge/structure-behavior-design/manifest.yaml`（リポジトリ内データ） | 新スキーマ移行時 | （データ更新） | — | — |

## 3. SOLID Risk Assessment

| Principle | Risk | Mitigation |
|---|---|---|
| **SRP** | Loader が「manifest 読み」「entry ファイル読み」「skill ルート走査」「異常 3 種の分類」と責務が膨らみがち | skill ルート走査を `SkillAssetCollector` に切り出し、Loader 本体は dispatch に留める。skill 解決の状態判別は SkillRootResolution の sentinel 群に閉じる |
| **SRP** | skill_renderer が「本体生成」「sibling 群生成」「path 組み立て」「SourceEntryIDs 整合」と複合化 | 単一 target × 単一 kind の責務範囲（≒ 既存 skill_renderer の役割）に閉じる限り SRP 違反にはならない。target 横断の共通処理が見つかった場合のみ後続で抽出する。先回りの抽象化はしない |
| **SRP（重複コード方針）** | 3 target の `skill_renderer.go` は本体 + sibling の生成ロジックがほぼ同形になり、本変更でその同形が増幅される | **本変更では「3 重複を維持する」とする**。理由: (a) 各 target の `path` 文字列・frontmatter 規約は独立して変わり得るため、共通化したヘルパに細かい可変点を流し込むと結果的に分岐が増える / (b) 重複は実装後に重複検査エージェントで再確認し、target 間で完全同形だった部分のみ後続 PR で抽出する。本 PR で `BuildSkillArtifacts(meta, pathBuilder)` のような共通ヘルパは導入しない |
| **OCP** | KindRenderer.Render の戻り型を `[]Artifact` に変えると既存 renderer 全てを修正する必要が出る | (a) RendererRegistry の Build ループは `append(artifacts, arts...)` への置換で要素数非依存になり、新 kind が複数 Artifact を返してもループ側は不変 (b) 既存 renderer は `[]Artifact{art}` を返すだけのワンライナーで済む。本変更は閉じている契約の機械的更新であり、設計上は「拡張ではなく契約の正規化」 |
| **LSP** | skill_renderer 以外の renderer も新シグネチャ `[]Artifact` を返すが、呼び出し側が「常に 1 件」を暗黙に期待しているとサイズ依存のバグを呼ぶ | RendererRegistry の Build は `append` で扱うため要素数に依存しない。テストでは「集合等価」で書く |
| **LSP** | Entry.Path の意味を skill では SkillMeta.Root の冗長コピーとして残す決定 | 外部消費者は SkillMeta.Root を参照する規約とする。LSP 違反は出ないが、規約をドキュメントとテストで担保する |
| **ISP** | SkillMeta を agent renderer 等が誤って参照する可能性 | Entry.Skill は Kind=KindSkill のとき非 nil・それ以外 nil の不変条件。agent/prompt/rule renderer は Entry.Skill を読まないことをコードレビューで担保 |
| **ISP** | SkillAssetCollector を interface 化すると skill 以外のユーザが現れず ISP 違反の手前まで行く | interface 化しない。Loader 内部の具体型として持つ（DIP の節も参照） |
| **DIP** | Loader が fs.FS を経由する既存規約を skill 走査でも維持できるか | SkillAssetCollector も入力に fs.FS を受け取る形にすれば、Loader の DI 構造は不変 |
| **DIP** | distribution/<target>/skill_renderer が source 内部の SkillMeta を参照する | source は distribution に対して上位（依存方向は distribution → source）なので、distribution が source の型を参照するのは方向として正しい。DIP 違反ではない |

## 4. Procedural Risk

### 4.1 Rules at risk of being placed in handlers/use cases

- **cli.cmd_install が「skill ディレクトリの異常検知」を組み込もうとする誘惑**: 失敗時のエラー文言に skill ルートのパスを足したい等。これは Loader が返した sentinel をそのまま CLI が printf するに留め、判定ロジックは Loader 側で完結する。
- **cli.cmd_uninstall が「sibling を集める専用ループ」を持とうとする誘惑**: 既存の `BelongsToPack` 一括判定で集合は得られる。CLI 側に skill 専用分岐を入れない。

### 4.2 Behavior that should move closer to state

- **SkillAsset の不変条件**: 「`..` を含まない」「SKILL.md 完全一致でない」等は Loader の手続き内に書くと再発する。SkillAsset コンストラクタに閉じ込めることで、Renderer・Builder・Installer のどの層に渡っても再検証不要にする。
- **SkillMeta.Root の正規化**: 末尾スラッシュ無し、`..` 不含等。これも SkillMeta コンストラクタに閉じる。
- **SkillRootResolution の状態識別**: Loader 内で「path が無い／ファイル／SKILL.md 欠落」を if 連鎖で判定して 3 種の error を `fmt.Errorf` するのではなく、内部関数で 1 度判定して状態列挙にマップしてから sentinel に変換する形を取る。識別ロジックを 1 箇所に集める。

### 4.3 Abstractions that may be premature

- **SkillCohort（skill 集合のファーストクラス化）**: 概念モデル §5.2 で導入しないと決定済み。pack 指定の動的ビューで足りる。本変更で持ち込まない。
- **SkillAssetCollector の interface 化**: 単一実装で十分。Loader 内部の具体型で持つ。テスト時に差し替えたい欲求が出てから検討。
- **MultiArtifactRenderer など別 interface**: 概念モデル §6.4 で代替案として却下済み。KindRenderer 戻り型の正規化（→ `[]Artifact`）で十分。
- **target ごとの SkillSiblingRenderer 抽出**: 現状 target 1 つにつき 1 つの skill_renderer.go で十分。target 間に共通処理が見えてきたら後で抽出する。先回りでの interface 化はしない。
- **ManifestPathShapePolicy の Go 型化**: JSON Schema 側だけで完結する。Go 側に独立型を作ると schema との二重メンテになる。

## 5. 境界・依存方向の確認

| 観点 | 確認 |
|---|---|
| `source` → 何にも依存しない | 維持。SkillAsset/SkillMeta/SkillRootResolution/SkillAssetCollector はいずれも source 内 |
| `inventory` → source のみ | 維持。本変更で inventory が新規概念を直接参照することは無い |
| `distribution/<target>` → source + inventory | 維持。skill_renderer が source.SkillMeta を読む方向 |
| `cli` → 全層 | 維持。CLI 側に skill 特有の分岐は追加しない |

依存方向は本変更で **変わらない**。
