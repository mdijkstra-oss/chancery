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

## Colors

Include the `color` column in query results when annotations or codes have colors. The renderer maps radix color names to hex automatically — bar colors match highlight colors in the document.

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
