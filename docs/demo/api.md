# API Reference

## Endpoints

### GET /__api/files

Returns the file index as JSON. Supports fuzzy search with `?q=` parameter.

```json
[
  {"relPath": "README.md", "name": "README.md", "isDir": false},
  {"relPath": "docs/guide.md", "name": "guide.md", "isDir": false}
]
```

### GET /__api/search

Fulltext search across all indexed documents.

```bash
curl http://localhost:3000/__api/search?q=architecture
```

### GET /__api/document

Document metadata including headings, forward links, and backlinks.

```bash
curl http://localhost:3000/__api/document?file=README.md
```

### GET /__api/graph

Link graph as JSON nodes and edges for visualization.

### GET /__api/tasks

- [ ] Extract TODO items from all markdown files
- [x] Support priority markers (`!high`, `!low`)
- [ ] Add due date parsing (`@due(2026-05-01)`)

See the [architecture overview](architecture.md) for system design.
