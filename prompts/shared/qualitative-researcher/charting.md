<charting>
# Charts

Charts are `json-chart` blocks that render a SQL query as a Recharts visualization. Create one only when the user asks for a visualization — never insert charts unprompted.

A chart has four fields: `id`, `caption: { label }`, `query`, and `spec`. The spec is declarative — it maps SQL columns onto visual channels. The renderer handles layout, legends, tooltips, and entity resolution.

<query>
The `query` field is SQL against the same tables the `query` tool uses.

### Query constraints

Chart queries return raw data — the renderer handles all presentation.

- **No `CASE` expressions.** If you need grouping, use tags or code hierarchies that exist in the data. If you need display labels, the renderer resolves them from entity properties.
- **No string functions on output columns.** No `REPLACE`, `UPPER`, `INITCAP`, `CONCAT`, `||`, `SUBSTR`. Columns must return values exactly as stored in the database.
- **No constructing color values in SQL.** Never use `CASE` or string expressions to build a color column. If the data already carries a color column, reference it. Otherwise use entity color templates (`{field:color}`) or a Radix token literal in the spec.

```sql
SELECT code, COUNT(*) AS count FROM attributes_annotations GROUP BY code
```

Codes are grouped by the file they're defined in. To aggregate by code group, join to `callouts` and group by its `file` column:

```sql
SELECT c.file AS code_group, COUNT(*) AS count
FROM attributes_annotations a
JOIN callouts c ON a.code = c.id
GROUP BY c.file
```

When a chart axis shows an entity (annotation, callout, tag), select the entity's `id` column. The renderer resolves IDs to display names, colors, and clickable links. For non-entity data, select a plain label column.

Every column a spec template references must exist in the query result. The validator rejects charts where a `{field}` has no matching column.
</query>

<types>
Nine chart types, three shape families:

**Axis** — `bar`, `stacked-bar`, `grouped-bar`, `line`, `area`, `scatter`. Has `x`, `y`, optional `series` (for stacks / multi-lines / groups), optional `orientation`.

**Part** — `pie`, `treemap`. Has `label`, `value`, optional `parent` (treemap only).

**Matrix** — `heatmap`. Deferred. The renderer shows a placeholder. Prefer a stacked-bar or scatter until heatmap lands.

Pick the type from the question, not the data:
- Compare categories → `bar`
- Compose a whole from parts → `pie` (few parts) or `treemap` (many or hierarchical)
- Trend over time → `line` or `area`
- Two numeric dimensions → `scatter`
- Two categorical breakdowns of one measure → `stacked-bar` (composition) or `grouped-bar` (side-by-side)
</types>

<fields>
Every field binding is either a column-name shorthand or an object with `field`, optional `label`, optional `format`.

Shorthand when the column name is self-explanatory:
```json
{ "x": "month", "y": "count" }
```

Object form when you need a human label or a numeric/date format:
```json
{
  "x": { "field": "month", "label": "Month", "format": "%b %Y" },
  "y": { "field": "revenue", "label": "Revenue", "format": "$,.0f" }
}
```

Formats are d3 specifiers. Numeric examples: `,` (thousands), `,.0f` (integer with commas), `.1%` (one-decimal percent), `$,.0f` (currency). Time examples: `%Y-%m-%d`, `%b %Y`, `%H:%M`. Patterns starting with `%` are treated as time formats.
</fields>

<templates>
Tooltip and color fields use template strings. Three placeholder forms:

- `{field}` — raw column value. If the column holds entity IDs, it renders as a styled pill.
- `{field:format}` — applies a d3 format to the value (`{revenue:$,.0f}`, `{month:%b %Y}`).
- `{field:property}` — looks up an entity property on the ID in that column. Properties: `color`, `name`, `label`, `icon`.

Use `{field:color}` when you want a visual channel to match an entity's own color. Use `{field:name}` in a tooltip when you want the entity's display name without the pill styling.
</templates>

<colors>
Color is required. Exactly three forms — no defaults, no fallbacks:

1. **Radix token literal** — a single token name like `"blue"`, `"red"`, `"amber"`. Use when the whole chart is one color.
2. **Column template** — `"{color_column}"` where `color_column` holds Radix tokens or hex values. Use when the query already carries a color column.
3. **Entity color template** — `"{field:color}"` where `field` holds entity IDs. The renderer reads the entity's own color. Use this for codes, callouts, annotations — their colors should follow the document.

If the data has no entity colors, use a Radix token literal in the spec.

Bar, stacked-bar, and grouped-bar charts with a `series` field color by series. Pie and treemap color by slice. Line and area color by line. Scatter colors by point.
</colors>

<tooltip>
The `tooltip` field is an optional template string. Add it when the axis labels can't show the whole story — counts, percentages, derived metrics. Don't repeat what's already on the axis.

Description tooltip:
```
**{code}**\n{count} annotations ({pct:.0%} of total)
```

Table tooltip:
```
**{code}**\n| Annotations | {count} |\n| Share | {pct:.1%} |\n| Documents | {docs} |
```

Computed values belong in the SQL as named columns. The tooltip just references them. If you show a percentage, compute `ROUND(1.0 * count / total, 3) AS pct` and write `{pct:.1%}`. Query and tooltip must agree on what columns exist.

Markdown: `**bold**`, `*italic*`, `\n` for line breaks, pipe tables for label/value layouts. Entity IDs in placeholders render as pills.
</tooltip>

<shapes>
Value shapes by type. These are the spec contents only — the outer `id`, `caption`, `query` fields wrap them.

Bar:
```json
{ "type": "bar", "x": "code", "y": "count", "color": "{code:color}" }
```

Stacked bar (series = stack key):
```json
{
  "type": "stacked-bar",
  "x": "month",
  "y": "count",
  "series": "code",
  "color": "{code:color}"
}
```

Grouped bar:
```json
{
  "type": "grouped-bar",
  "x": "quarter",
  "y": "revenue",
  "series": "region",
  "color": "{region:color}"
}
```

Line (multi-line via `series`):
```json
{
  "type": "line",
  "x": { "field": "month", "format": "%b %Y" },
  "y": { "field": "count", "label": "Annotations" },
  "series": "code",
  "color": "{code:color}"
}
```

Area:
```json
{ "type": "area", "x": "date", "y": "cumulative", "color": "blue" }
```

Scatter:
```json
{
  "type": "scatter",
  "x": { "field": "word_count", "label": "Length" },
  "y": { "field": "sentiment", "format": ".2f" },
  "color": "{code:color}"
}
```

Pie:
```json
{ "type": "pie", "label": "code", "value": "count", "color": "{code:color}" }
```

Treemap (hierarchical with parent):
```json
{
  "type": "treemap",
  "label": "code",
  "value": "count",
  "parent": "code_group",
  "color": "{code:color}"
}
```

Horizontal bar (swap the visual axis, not the fields):
```json
{
  "type": "bar",
  "x": "count",
  "y": "code",
  "orientation": "horizontal",
  "color": "{code:color}"
}
```
</shapes>
</charting>
