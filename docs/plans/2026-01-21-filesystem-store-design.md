# Filesystem Backends for Package Store

## Overview

Add three filesystem-based implementations of `store.Interface` to enable local storage options alongside the existing S3API backend.

## Implementations

### 1. DirectFile (`directfile.go`)

Simple key→file mapping. A key `foo/bar/baz` creates directories `foo/bar/` and a file `baz` in the base directory.

**Constructor:**

```go
func NewDirectFile(baseDir string) (Interface, error)
```

**Key details:**

- Creates base directory if missing
- Uses `os.MkdirAll` for nested paths automatically
- `List` uses `filepath.WalkDir` to find files under prefix
- No concurrency protection beyond OS file locking
- Empty prefix means list everything in base dir
- Files stored as raw bytes

**Path validation:**

- Keys containing `..` segments are rejected (return `ErrBadConfig`)
- Empty keys are rejected
- Leading `/` is stripped

**Error handling:**

- `Exists` returns `ErrNotFound` if file doesn't exist
- `Get` returns `ErrNotFound` wrapped around `os.ErrNotExist`
- `Delete` succeeds silently if file already gone

**Prometheus metrics:** `directfile` driver

---

### 2. JSONMutexDB (`jsonmutex.go`)

In-memory JSON index with mutex-protected concurrent access.

**Constructor:**

```go
func NewJSONMutexDB(baseDir string) (Interface, error)
```

**Directory structure:**

```
baseDir/
├── index.json    # key→filename index
└── data/         # actual data files
    └── <files>
```

**Data structure:**

```go
type jsonMutexDB struct {
    mu    sync.RWMutex
    index map[string]string  // key → filename
    base  string
}
```

**Key details:**

- Creates both directories if missing
- On `Set`: write lock, update in-memory map, write index JSON, write data to `data/<filename>`
- On `Get/Delete/Exists`: read lock, lookup key in index, operate on `data/<filename>`
- `List` filters keys by prefix using `strings.HasPrefix`
- Saves entire JSON file on each write

**Error handling:**

- Returns `ErrBadConfig` if `index.json` is corrupted on load
- Creates empty index if file doesn't exist
- `Get` returns `ErrNotFound` if key not in index

**Prometheus metrics:** `jsonmutex` driver

---

### 3. CAS (`cas.go`)

Content-addressable storage with deduplication. Identical data shares storage; index maps keys to hash-based filenames.

**Constructor:**

```go
func NewCAS(baseDir string) (Interface, error)
```

**Directory structure:**

```
baseDir/
├── index.bolt    # BoltDB key→hash index
└── objects/      # content-addressed data files
    └── <hash files>
```

**BoltDB schema:**

- Bucket: `keys`
- Key: the store key (e.g., "foo/bar/baz")
- Value: SHA-256 hash (hex encoded, 64 chars)

**Storage format:**

- Objects stored as `objects/aa/bb/filename` using first 4 chars for sharding
- Full hash as filename

**Key details:**

- On `Set`: compute SHA-256, write object if missing, update BoltDB
- On `Get`: lookup hash in BoltDB, read from `objects/`
- On `Delete`: remove BoltDB entry (orphaned objects OK, GC not required initially)
- `List`: iterates BoltDB keys, filters by prefix

**Error handling:**

- Returns `ErrBadConfig` if BoltDB open fails
- Returns `ErrNotFound` if key missing or object file gone

**Prometheus metrics:** `cas` driver

---

## Testing

Each implementation has its own test file using `t.TempDir()`:

- **`directfile_test.go`**: CRUD, nested directories, path edge cases, prefix listing
- **`jsonmutex_test.go`**: CRUD, concurrent safety, index persistence, corrupted index
- **`cas_test.go`**: CRUD, deduplication, BoltDB operations, orphaned objects

---

## Error Consistency

All implementations follow the S3API pattern:

- Missing keys → `ErrNotFound` (wrapped)
- Invalid configuration → `ErrBadConfig`
- IO failures → underlying error wrapped with context

---

## Dependencies

New dependency required for CAS:

- `go.etcd.io/bbolt` - BoltDB for the CAS index
