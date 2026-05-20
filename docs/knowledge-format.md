# Knowledge Format 仕様

`knowledge/` 配下に置く中立フォーマットの正式仕様。`knit` はこの形式を入力として読み、各 AI ツール固有のフォーマット・ディレクトリ構造に変換する。

ルート [README.md](../README.md) の「ナレッジの抽象モデル」「ナレッジパック」セクションが概念モデルにあたる。本ドキュメントはそれを満たす実ファイルのスキーマを定義する。

---

## ファイル構成

```text
knowledge/
└── <pack-name>/
    ├── manifest.yaml          # パック定義（必須）
    ├── skills/<name>.md       # kind: skill
    ├── agents/<name>.md       # kind: agent
    ├── rules/<name>.md        # kind: rule
    └── prompts/<name>.md      # kind: prompt
```

- `<pack-name>` / `<name>` は kebab-case
- `manifest.yaml` 以外のサブディレクトリは、その kind を含むエントリが 1 つ以上ある場合に作る（不要なディレクトリは置かない）

---

## id 命名規約

各ナレッジファイルは中立な ID を持つ。形式:

```text
<pack-name>.<kind>.<entry-name>
```

- ピリオド `.` 区切り
- 各セグメントは kebab-case
- `<kind>` は `skill` / `agent` / `rule` / `prompt`
- パック内で一意

例:

```text
structure-behavior-design.skill.orchestrator
structure-behavior-design.agent.solid-reviewer
```

---

## 共通 frontmatter スキーマ

すべてのナレッジファイルは YAML frontmatter + Markdown 本文の形式を取る。

| field | type | 必須 | 説明 |
|---|---|:---:|---|
| `id` | string | ✓ | 上記命名規約に従う中立 ID |
| `kind` | enum (`skill` / `agent` / `rule` / `prompt`) | ✓ | ナレッジの種類 |
| `name` | string | ✓ | ターゲットツールに渡される識別子。慣例として `<pack-name>-<entry-name>` |
| `description` | string | ✓ | 概要。複数行は `\|` で折り返し可 |
| `tags` | string[] | - | 分類用タグ |
| `tools` | map<target, ToolConfig> | - | ターゲット別設定（後述） |
| `uses_skills` | string[] | - | 依存スキルの中立 ID 配列。`kind: agent` のみで意味を持つ |

### `tools.<target>` (ToolConfig)

ターゲットごとのビルド指示。

| field | type | 必須 | 説明 |
|---|---|:---:|---|
| `enabled` | bool | - | `true` のときビルドの対象にする。省略時は `manifest.yaml` の `default_tools` に従う |
| `frontmatter` | map<string, any> | - | ターゲット固有 frontmatter として、生成物の frontmatter にそのまま展開する追加メタデータ |

---

## kind 別の追加ルール

### `kind: skill`

- 追加フィールドなし
- 本文は手順・観点・出力形式の定義

### `kind: agent`

- `uses_skills` で依存スキルを中立 ID で宣言できる
- 本文は専門担当の役割・レビュー観点・出力形式

ビルダーは `uses_skills` の中立 ID をターゲットの参照形式に変換する（例: Claude Code 向けでは `<name>` を抽出して `skills:` 配列に展開する）。

### `kind: rule`

- 追加フィールドは未定義（将来拡張）
- 本文は常時適用される指示・前提

複数の rule をどう合成して 1 ファイルにまとめるか（順序・見出しの付与など）の規約は別途定義する。

### `kind: prompt`

- 追加フィールドは未定義（将来拡張）
- 本文は再利用可能なプロンプト・スラッシュコマンドの内容

---

## manifest.yaml スキーマ

パック単位の定義ファイル。`knowledge/<pack-name>/manifest.yaml` に置く。

| field | type | 必須 | 説明 |
|---|---|:---:|---|
| `pack` | string | ✓ | パック名（kebab-case）。ディレクトリ名と一致 |
| `version` | string | ✓ | semver |
| `description` | string | ✓ | パックの概要 |
| `default_tools` | string[] | - | エントリ側で `tools.<target>.enabled` が省略された場合に有効化するターゲットの一覧 |
| `entries` | Entry[] | ✓ | パックに含まれるエントリ一覧 |

### Entry

| field | type | 必須 | 説明 |
|---|---|:---:|---|
| `id` | string | ✓ | 該当ファイルの frontmatter `id` と一致 |
| `path` | string | ✓ | パックルートからの相対パス（例: `skills/orchestrator.md`） |

manifest と各ファイルの frontmatter は冗長だが、パック全体の一覧性とファイル単体の自己記述性の両方を担保するために両方を維持する。整合性チェックは `knit` のビルド時に行う。

---

## tools.<target> の伝搬ルール

`knit` のターゲット別ビルダーは以下のルールで生成物を作る。

1. `tools.<target>.enabled` が `true`（または `default_tools` に含まれる）のエントリのみを対象にする
2. 中立フィールド（`name`, `description`, `uses_skills` など）はビルダーがターゲットの慣習に従って変換して生成物の frontmatter に書く
3. `tools.<target>.frontmatter` のキー / 値は、生成物の frontmatter にそのままマージする
4. 同名フィールドが中立変換結果と `tools.<target>.frontmatter` の両方にある場合は、`tools.<target>.frontmatter` を優先する（個別オーバーライド）

---

## 本文（Markdown）の扱い

- Markdown 本文はそのまま生成物の本文として転記する
- 現スコープではテンプレート展開・変数置換・他エントリへのリンク解決は行わない
- 将来、`[[id]]` 形式のクロスリファレンスや変数展開を導入する余地は残すが、本仕様には含めない

---

## バリデーション

中立フォーマットは [JSON Schema](https://json-schema.org/) でバリデーションする。スキーマファイルは `schema/` 直下に置く。

| 対象 | スキーマファイル |
|---|---|
| パック定義（`manifest.yaml`） | [`schema/manifest.schema.json`](../schema/manifest.schema.json) |
| エントリ frontmatter（各 `.md` 冒頭） | [`schema/entry.schema.json`](../schema/entry.schema.json) |

`knit` のビルド時バリデーションと、エディタ（YAML Language Server 等）でのリアルタイムチェックの両方で同じスキーマを参照する。

### スキーマで強制している主な制約

- `id` / `name` / `pack` などは kebab-case
- `version` は semver
- `id` は `<pack>.<kind>.<entry>` 形式、`uses_skills` の各要素は `<pack>.skill.<entry>` 形式
- `kind: agent` 以外で `uses_skills` を書くとエラー
- 未知のフィールド（例: スペルミス）は `additionalProperties: false` でエラーになる

---

## 例

サンプルは `knowledge/structure-behavior-design/` を参照。

- skill 例: [`knowledge/structure-behavior-design/skills/orchestrator.md`](../knowledge/structure-behavior-design/skills/orchestrator.md)
- agent 例: [`knowledge/structure-behavior-design/agents/solid-reviewer.md`](../knowledge/structure-behavior-design/agents/solid-reviewer.md)
- manifest 例: [`knowledge/structure-behavior-design/manifest.yaml`](../knowledge/structure-behavior-design/manifest.yaml)
