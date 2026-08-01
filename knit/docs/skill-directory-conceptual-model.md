# Skill Directory Unit - Conceptual Model

> Historical design record for the original directory migration. The current
> canonical model is defined by
> [`../../docs/knowledge-format.md`](../../docs/knowledge-format.md).

要件仕様: [skill-directory-requirements.md](./skill-directory-requirements.md)

本文書は既存の概念モデル（[refactoring-conceptual-model.md](./refactoring-conceptual-model.md)）への追補。差分のみを扱う。

## 1. 既存モデルからの変更点（概要）

| 種別 | 対象 | 変更内容 |
|---|---|---|
| 新規 | SkillAsset | skill ルート相対パス + 中身を持つ値オブジェクト（要件文書の「sibling」と同義） |
| 新規 | SkillMeta | Entry の Kind=skill 時のサブ構造（Agent/AgentMeta の対称）。skill ルート pack 相対 path と SkillAsset 集合を保持 |
| 新規 | SkillRootResolution | loader が skill ルートを解決した結果の状態列挙（正常 / path 不在 / path がファイル / SKILL.md 欠落） |
| 新規 | SkillAssetCollector | skill ルート配下の通常ファイルから本体を除いた集合を返す境界。実装は Loader 内に閉じる |
| 新規 | ManifestPathShapePolicy | manifest スキーマ上の「`id.kind` と `path` 形状の整合」規約（既存の target 配置 path 用 `PathPolicy` と名前空間を分離） |
| 更新 | Entry | `Skill *SkillMeta` を追加。Kind=KindSkill のとき非 nil・それ以外は nil。`Path` の意味は変更せず「pack 相対の場所参照（ファイル or ディレクトリ）」のまま。skill ルートの正本は SkillMeta.Root が持ち、Entry.Path は値の冗長コピーとして互換のため残す（外部消費者は SkillMeta.Root を参照する） |
| 更新 | KindRenderer | 単一 Artifact から **複数 Artifact** を返す契約に拡張。skill 以外は 1 要素スライスを返す（代替案との比較は §6.4） |
| 更新 | Artifact | 既存の「`SourceEntryIDs` は集合的」コメントを契約化（複数 Artifact が同一 entry id を共有可） |
| 更新 | Manifest スキーマ | ManifestPathShapePolicy を schema として表現（`id.kind` の値で `path` 形状が決まる構造） |
| 削除 | 旧 path pattern「skill 用に SKILL.md を含む path」 | ManifestPathShapePolicy 違反として拒否 |

agent / rule / prompt の Entry・Renderer の概念的な責務は **変わらない**。

## 2. Conceptual Model（追補）

### 新規概念

| Concept | Meaning | State | Behavior | Constraint / Invariant |
|---|---|---|---|---|
| SkillAsset | skill ディレクトリ配下、本体 SKILL.md を除く 1 ファイルを表す値オブジェクト。要件文書の「sibling」と同義 | skill ルートからの相対パス文字列 / 不透明な byte 列の中身 | （getter のみの値オブジェクト。状態に対する操作は持たない） | 構築時: 相対パスが空でない・絶対でない・`..` を含まない・forward slash 区切り・`SKILL.md`（完全一致）でない |
| SkillMeta | Entry の Kind=skill のときのみ意味を持つサブ構造。skill ルートと SkillAsset 集合の一体性を表す | skill ルート pack 相対 path (Root) / SkillAsset の集合 (Assets) | Root と Assets を不可分の単位として保持 | Root は空でない pack 相対ディレクトリ path・末尾スラッシュ無し・`..` を含まない。Assets 内に Path 重複は無い。Assets の各 Path はいずれも Root の外に出ない |
| SkillRootResolution | loader が skill ルートを解決した結果を表す状態列挙 | { Resolved / PathMissing / PathIsFile / BodyMissing } の 4 値 | 状態に応じて識別可能なエラー identity を持つ | EC1〜EC3 の 3 種異常は互いに `errors.Is` で識別できる（Resolved 以外は専用 sentinel に対応） |
| SkillAssetCollector | skill ルート配下から本体 SKILL.md を除いた通常ファイル集合を構築する境界（Loader 内部の責務） | （戦略のみ、永続状態を持たない） | skill ルートを入力に SkillAsset 集合を返す | 入力 fs.FS の `ReadDir` セマンティクスのみに依拠。通常ファイル以外（ディレクトリ・シンボリックリンク・特殊ファイル）は黙って除外 |
| ManifestPathShapePolicy | manifest スキーマ層の「`id.kind` と `path` 形状の整合」を司る規約 | （schema 宣言のみ、状態無し） | manifest の各 entry を受け入れ可否で判定 | target 配置 path 用 `PathPolicy` とは別概念（名前空間が紛らわしいため命名で区別） |

### 更新される概念

| Concept | 既存の State | 追加・変更 |
|---|---|---|
| Entry | ID / Kind / Name / Description / Tags / Tools / Agent (*AgentMeta) / Path / Body | **Skill (*SkillMeta) を追加**。Kind=KindSkill のとき非 nil、それ以外で nil。Body は引き続き SKILL.md 本文を保持。Entry.Path の意味は **変えない**（pack 相対の場所参照）。skill ルートの正本は SkillMeta.Root が持ち、Entry.Path は loader 内部での参照と既存テストの互換のため同じ値を保持する冗長コピーに留まる。外部消費者（Renderer 等）は skill 文脈では SkillMeta を経由する |
| KindRenderer | `Render(Entry, Pack) → Artifact` | **`Render(Entry, Pack) → []Artifact` に拡張**。skill 用 renderer は SKILL.md 本体 1 + sibling N の計 1+N 個を返す。agent/prompt 用 renderer は 1 要素スライスを返す |
| Artifact | Target / Path / Content / Mode / SourceEntryIDs / SourceRef | 概念は変えない。skill 由来の複数 Artifact が単一の `SourceEntryIDs = [<pack>.skill.<name>]` を共有することを **契約として明示**（既存コメントの規範化） |
| Manifest（schema 上の Entry） | `id` / `path` の単純パターン | `id.kind` の値で `path` 形状が決まる構造（kind=skill はディレクトリ、agent/rule/prompt はそれぞれの kebab-case `.md` ファイル）。スキーマ違反として拒否されるケースが要件 EC4 に列挙されている |

既存の RuleAggregator / RendererRegistry / Builder の責務は変えない（Builder は KindRenderer 戻り値が複数になっても、それをそのまま結果スライスに append するだけ）。

## 3. Relationships（追補）

| Concept A | Relationship | Concept B | Notes |
|---|---|---|---|
| Entry (Kind=skill) | owns | 1 SkillMeta | SkillMeta は Entry の所有物。Entry が SkillMeta を介して skill ルートと Assets を一体的に保持する |
| SkillMeta | composes | Root + Assets | Root と Assets は不可分。SkillMeta のライフサイクルは Entry に従う |
| SkillMeta | aggregates | 0..n SkillAsset | 集合内で Path 重複は無い |
| Entry (Kind=skill) | renders to | 1..n Artifact | 本体 1 + sibling N。すべて同じ SourceEntryIDs を共有（§4 不変条件 + §5.5 EntryID 値オブジェクト整合） |
| Entry (Kind ≠ skill) | renders to | 1 Artifact | 既存どおり（KindRenderer 戻り型が `[]Artifact` でも要素 1 件） |
| Loader | uses | SkillAssetCollector | skill ルート走査の戦略を Loader 内で組み込み利用 |
| Loader | returns | SkillRootResolution | 異常時は 3 種に識別された sentinel エラーで通知 |
| Artifact (skill 由来) | has same | SourceEntryIDs | 主・sibling すべて `[<pack>.skill.<name>]` |
| Provenance (skill 由来 Installation) | shares | EntryID | 既存 `BelongsToPack` が SkillMeta 経由で生まれた集合を一括判定するための前提 |
| Manifest entry (kind=skill) | refers to | skill ルートディレクトリ | ManifestPathShapePolicy で path 形状が拘束される |
| ManifestPathShapePolicy | distinct from | target 配置 PathPolicy | 名前空間を分離（前者は manifest 上の path 形状、後者は target 配置上の path 規約） |
| skill ルートディレクトリ | requires | `SKILL.md` (固定名) | EC1 で欠落を専用 sentinel として識別 |

## 4. 不変条件（型レベルで守るべきもの）

| 不変条件 | 守る型 |
|---|---|
| Entry.Skill は Kind == KindSkill のとき非 nil、それ以外で nil | Entry（構築時バリデーション） |
| Entry.Agent は Kind == KindAgent のとき非 nil、それ以外で nil（既存） | Entry |
| SkillMeta.Root は空でない pack 相対ディレクトリ path・末尾スラッシュ無し・`..` を含まない | SkillMeta（コンストラクタ） |
| SkillMeta.Assets 内に Path 重複は無い | SkillMeta（構築時） |
| SkillAsset.Path は空でない・絶対でない・`..` を含まない・forward slash 区切り | SkillAsset（コンストラクタ） |
| SkillAsset.Path は `SKILL.md` 完全一致でない | SkillAsset |
| SkillRootResolution は 4 値のうち 1 つに必ず収束する。異常 3 種は互いに `errors.Is` で識別可能 | Loader（呼び出し契約） |
| KindRenderer は `[]Artifact` を返し、要素数 ≥ 1 | KindRenderer 実装契約 |
| skill 由来の Artifact 群はすべて同一 `SourceEntryIDs` を持つ | skill 用 renderer 実装契約 |
| sibling Artifact の Mode はゼロ値 | skill 用 renderer 実装契約 |
| Manifest スキーマで `id.kind` と `path` 形状が整合する | ManifestPathShapePolicy |

## 5. 既存モデルとの整合確認

### 5.1 Provenance 所有

既存ルール:
- Label が Provenance を所有する正本。Installation.Provenance は Label.Provenance のビュー。

本変更による影響:
- skill 由来の複数 Installation がそれぞれ独立した Label を持ち、各 Label の Provenance に同じ `SourceEntryIDs` が記録される。
- Provenance の **多重格納はしない**（Label が単一の真正本という原則は維持）。
- skill 全体を 1 つの Provenance に集約する新たな概念は導入しない。

### 5.2 Inventory 整合性条件

既存ルール:
- ある InstallationID に対し「Label 有 ⇔ File 有」。

本変更による影響:
- skill 由来の複数 InstallationID それぞれが独立して 2×2 状態表に従う。
- 「skill 集合の整合（全 sibling が揃っているか）」は **本要件の整合性条件には含めない**。集合の一部だけが残った状態（例: 部分失敗時）は、各 Installation 単位では正常状態のまま許容される。
- 設計判断: `SkillCohort`（skill 集合を永続化されたファーストクラスの単位として表す概念）は **導入しない**。集合は `Provenance.Pack` および `SourceEntryIDs` に紐付く動的ビューとしてのみ存在する。理由:
  - F6 の pack 指定 uninstall は `BelongsToPack` の動的判定で集合を一括処理できる。永続化集合を持たなくても要件は満たせる。
  - 集合トランザクション（EC6 で扱う部分失敗時の全体ロールバック）は要件で明示的に非ゴール。
  - reinstaller の sibling 増減仕様（R7）は本要件のスコープ外。
- 結果として、knit list / uninstall は従来どおり Installation 単位で動き、集合としての健全性は CLI レイヤの責務外。

### 5.3 ArtifactPath 値オブジェクト

既存ルール:
- ArtifactPath は Inventory root 相対の安全なパス。`..`、絶対、NUL/バックスラッシュ含有を拒否。

本変更による影響:
- sibling Artifact の `Path` は `skills/<name>/<subpath>` 形式の ArtifactPath。複数階層を含むため、ArtifactPath の不変条件はそのまま流用できる。
- skill ルート相対 path（= SkillAsset.Path）は ArtifactPath とは別の値（pack 相対ではなく skill ルート相対）。混同しないよう **別の型** とする。

### 5.4 ソース fs.FS の境界

既存:
- Loader は `fs.FS` を介してパックを読む。実装は os/embed/git fetch 等いずれでも良い。

本変更:
- 「skill ルートディレクトリの配下を集合として読む」のは Loader の責務。`fs.FS` の `ReadDir` セマンティクスのみに依拠し、シンボリックリンクや特殊ファイルは黙って除外する。
- この責務を担う概念として `SkillAssetCollector` を導入する。Loader はこれを内部協力者として使う。
- 「どう走査するか」（深さ優先か幅優先か、何回の syscall か）は実装詳細であり概念モデルには含めない。

### 5.5 EntryID 値オブジェクト導入順序との整合

既存 refactoring-conceptual-model 側で `EntryID` 値オブジェクト化が予告されている。本変更は先行・後置・同時のいずれでも動作する設計とする。

- 本変更が先行する場合: skill 由来 Artifact の `SourceEntryIDs` は文字列スライスとして同一値を共有する。
- EntryID 値オブジェクト化が後で実施される場合: 「同一文字列を共有」は「同一 EntryID 値を共有」と読み替えるだけで契約は維持される。

両者の依存は無いため、PR の前後関係は機械的に問題ない。

## 6. Structural Risks

### Missing concepts — 本変更で解消

- **SkillAsset / SkillMeta**: 「skill が複数ファイルから成る」事実が現状どこにも型として現れていない。Entry.Body だけでは sibling を表現できず、Pack の構造ですらない。
- **SkillRootResolution**: skill ルート解決の正常 / 3 種異常状態が型・状態列挙としてどこにも現れない。
- **SkillAssetCollector**: skill ルート走査戦略の責務名が無いと、F2 の「通常ファイル限定」「`..` 不許可」「シンボリックリンク除外」が手続きの行間に戻る。
- **ManifestPathShapePolicy**: manifest schema 層の「`id.kind` と `path` 形状の整合」を司る規約に名前が無いと、target 配置 path 用 `PathPolicy` と混同される。
- **Kind 別 Path 形状**: 現状の manifest schema は kind と path 形状が独立しており、誤組み合わせを schema レベルで弾けない。本変更で kind に依存した path 形状を schema 上で表現する。

### Hidden state — 本変更で解消

- skill ディレクトリ全体に対する Loader の責務が「単一ファイルを読む」と書かれているのに、schema 説明は「ディレクトリと sibling」と宣言。読み手と実装が乖離。
- KindRenderer の戻り型が単一 Artifact なのに、Artifact.SourceEntryIDs の doc コメントは「複数 entry 共有」「複数 artifact 共有」両方が起こり得ると述べている。型と doc が乖離。

### Change-prone areas — 本変更で縮小

- 新しい kind を「複数ファイルからなる」概念で追加したいケース（将来、agent set / prompt pack 等）に対し、KindRenderer の `[]Artifact` 戻り値は再利用可能になる。

### Boundary candidates

- **Loader ↔ Entry (source 内部)**: skill ルート走査を Loader 内に閉じ、Entry には集合化された SkillMeta だけが渡る。Loader の内部に「skill 走査戦略」(SkillAssetCollector) が出現するが、これは source パッケージ内部の関心であり外部 API には漏らさない。
- **Entry ↔ Renderer (source / distribution)**: Entry に Skill フィールドが乗ることで、distribution の skill renderer は Entry 経由で sibling 集合を取得する。Renderer は fs アクセスをしない。

### 6.4 KindRenderer 戻り型変更の代替案比較

| 案 | 説明 | 利点 | 欠点 |
|---|---|---|---|
| **採用案: `Render → []Artifact`** | KindRenderer.Render の戻り型をスライス化 | (a) `Artifact.SourceEntryIDs` の既存 doc コメント「複数 Artifact が同一 entry を共有可」と型が一致 / (b) RuleAggregator の対称（複数 Entry → 1 Artifact の逆方向） / (c) 将来「複数ファイルからなる kind」を追加するとき再利用可能 / (d) Builder の append ロジックがそのまま流用できる | 全 target × 全 kind の renderer 実装（9 個前後）でシグネチャ変更が発生 |
| 案 B: 別 interface `MultiArtifactRenderer` 併設 | skill 用だけ別 interface を実装、registry が両方をディスパッチ | 既存 KindRenderer は触らない | registry に 2 種の dispatch ロジックが増える / skill が「特殊扱い」の概念として残り、将来の同類 kind が出るたびに interface が増える |
| 案 C: `RenderResult{Primary Artifact, Auxiliaries []Artifact}` | 戻り型を構造体化 | Primary / Auxiliary の区別が型に出る | Primary という概念は skill では SKILL.md という固有事情に過ぎず、他 kind では空概念。汎化として弱い |
| 案 D: 戻り型は単一のまま、`Siblings(Entry, Pack) ([]Artifact, error)` メソッドを追加 | KindRenderer に sibling 取得を委ねる | 戻り型は不変 | 2 メソッドの呼び出し順序・整合が暗黙化し、registry が複雑化 |

採用案は (a)〜(d) の利点で他案を上回り、欠点はシグネチャ機械的変更にとどまる。Builder/registry 側の概念単純性を保つため採用案を選ぶ。

### 残るリスク

- **R-A**: KindRenderer の戻り型変更は全 target × 全 kind の renderer に波及（既存 9 個前後）。シグネチャ変更だが処理内容は単純（既存 renderer は `[]Artifact{art}` を返すだけ）。
- **R-B**: skill ルートの走査結果（SkillAsset 集合）の **順序保証** は概念モデル上はしない。Builder 結果の順序を期待するテストがある場合、本変更でランダム化される可能性。これはテスト設計時に「集合等価」で書くことで吸収する。
- **R-C**: target 側 path_policy（特に gemini）が `skills/<name>/<subpath>` のサブディレクトリ階層を許容しているかは未確認。許容していなければ policy の拡張も発生する（要件 R5）。
- **R-D**: Entry.Path の冗長コピー残置は将来の負債。Entry.Path を完全削除する判断は本変更のスコープ外（影響範囲が広い）。Renderer 等の外部消費者を SkillMeta.Root 経由に統一しておけば、後続の削除リファクタは容易。
