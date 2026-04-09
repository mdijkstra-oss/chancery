<charting>
# Charts

Charts are `json-chart` blocks embedded in documents. They render inline visualizations driven by SQL queries against the project database.

Create charts only when the user asks for a visualization. Don't insert charts into documents unprompted.

## Query

The `query` field is SQL against the same tables available to the `query` tool. Use `$file` to scope to the current document — it resolves to the filename at render time.

Cross-file query:
```sql
SELECT code, color, COUNT(*) as count FROM attributes_annotations GROUP BY code, color
```

File-scoped query:
```sql
SELECT code, color, COUNT(*) as count FROM attributes_annotations WHERE _file = $file GROUP BY code, color
```

## Code groups

Codes are grouped by the file they're defined in — all codes in `economic_factors.md` form one group, all codes in `emotional_responses.md` form another. To aggregate annotations by code group, join to the `callouts` table and group by its `file` column:

```sql
SELECT c.file as code_group, COUNT(*) as count
FROM attributes_annotations a
JOIN callouts c ON a.code = c.id
GROUP BY c.file
```

## Labels

When the axis references entities that have IDs (annotations, callouts, tags), use the `id` column — not a label column. The renderer resolves IDs to display names, colors, and clickable links automatically. For data that doesn't map to entities, write a label column directly.

## Colors

Include the `color` column in query results when annotations or codes have colors. The renderer maps radix color names to hex automatically — bar colors match highlight colors in the document.

## Tooltip

The `tooltip` field is a template string shown when the user hovers a data point. It adds context the axis labels can't show — counts, percentages, derived metrics. Don't repeat what's already visible on the axis.

`{column}` interpolates the value from that query column. Entity IDs in placeholders render as styled pills with icons.

Formatting: `**bold**`, `*italic*`, `\n` for line breaks, and pipe tables for structured data. Choose the format that fits the data — a short description for simple charts, a table when showing multiple metrics per point.

Description tooltip:
```
**{code}**\n{count} annotations ({pct}%)
```

Table tooltip:
```
**{code}**\n| Annotations | {count} |\n| Share | {pct}% |\n| Documents | {docs} |
```

Use **bold** for headers in table cells when the table has a label/value layout.

Computed values (percentages, ratios, rounded numbers) belong in the SQL query as named columns — the tooltip just references them. If you show a percentage, compute `ROUND(... * 100.0 / ..., 1) as pct` in the query and write `{pct}%` in the tooltip. Query and tooltip must agree on what columns exist.

## Options

The `options` field is an ECharts option object. The renderer injects `dataset.source` from query results — don't include dataset in options.

Bar chart:
```json
{"series": [{"type": "bar", "encode": {"x": "code", "y": "count"}}]}
```

Pie chart:
```json
{"series": [{"type": "pie", "encode": {"value": "count", "itemName": "code"}}]}
```

Line chart:
```json
{"series": [{"type": "line", "encode": {"x": "date", "y": "count"}}]}
```

The model knows ECharts — use its full option API when needed (axis labels, legends, stacked series, etc).
</charting>
