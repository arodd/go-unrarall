# go-unrarall

`go-unrarall` is a self-contained Go CLI for bulk RAR extraction and cleanup.

It targets practical parity with the original `unrarall` shell workflow while avoiding runtime dependencies on external tools like `unrar`, `rar`, `7z`, `cksfv`, `find`, or `sed`.

## What It Does

- Scans a directory tree for first-volume archives:
  - `*.rar`
  - `*.part01.rar` style sets
  - `*.001` style sets
- Extracts archives in-process (including multi-volume sets).
- Optionally verifies SFV CRC32 manifests before extraction.
- Supports password retries from a password file.
- Supports recursive nested extraction up to `--depth` while keeping top-level candidate scanning unbounded.
- Supports cleanup hooks (`--clean`) for post-extraction cleanup.

## Build

Recommended build flow with vendored modules:

```bash
go mod vendor
CGO_ENABLED=0 go build -mod=vendor -o unrarall ./cmd/unrarall
```

Notes:

- `CGO_ENABLED=0` is recommended for a fully static, portable binary where possible.
- `-mod=vendor` ensures builds use vendored dependencies.

## Run

```bash
./unrarall [options] <DIRECTORY>
```

Help and version:

```bash
./unrarall --help
./unrarall --version
```

## Common Examples

Extract everything under a directory:

```bash
./unrarall /data/downloads
```

Dry run (no writes):

```bash
./unrarall --dry /data/downloads
```

Extract with full archive paths preserved:

```bash
./unrarall --full-path /data/downloads
```

Extract into a dedicated output directory:

```bash
./unrarall --output /data/extracted /data/downloads
```

Continue even when SFV checks fail:

```bash
./unrarall --force /data/downloads
```

Allow partial overall success exit behavior (`--allow-failures`):

```bash
./unrarall --allow-failures /data/downloads
```

Try passwords from a file:

```bash
./unrarall --password-file ~/.unrar_passwords /data/downloads
```

Append command output to a log file while still writing to the console:

```bash
./unrarall --log-file /var/log/unrarall.log /data/downloads
```

Run cleanup hooks after extraction:

```bash
./unrarall --clean=all /data/downloads
./unrarall --clean=rar,empty_folders --force /data/downloads
```

## Options

- `-h, --help`: show usage and exit.
- `--version`: show version and exit.
- `-v, --verbose`: verbose logging.
- `-q, --quiet`: suppress command output.
- `-d, --dry`: dry-run mode.
- `-f, --force`: continue candidate processing when SFV/extraction checks fail and allow cleanup hooks after extraction errors.
- `--allow-failures`: return exit code `0` when there is at least one successful candidate (extracted or skipped), even if some candidates failed.
- `-s, --disable-cksfv`: disable SFV verification for `<stem>.sfv` manifests.
- `--clean=SPEC`: `none|all|hook1,hook2`.
- `--full-path`: preserve archive paths while extracting.
- `-o, --output DIR`: output directory (must already exist).
- `--log-file FILE`: append command output to `FILE` without changing normal stdout/stderr behavior.
- `--depth N`: nested recursion depth budget (default `4`); top-level candidate scanning remains unbounded.
- `--skip-if-exists`: skip extraction if all archive entries already exist by name.
- `--password-file FILE`: password source file (default `~/.unrar_passwords`).
- `--max-dict BYTES`: max RAR dictionary size (default `1073741824`, 1 GiB).
- `--allow-symlinks`: allow symlink extraction with in-tree target validation.

## Cleanup Hooks

Available hook names for `--clean=`:

- `covers_folders`
- `nfo`
- `osx_junk`
- `proof_folders`
- `rar`
- `sample_folders`
- `sample_videos`
- `windows_junk`
- `empty_folders`

Special values:

- `all`: run all hooks in default order.
- `none`: disable cleanup hooks.

## Behavior Contract

This section documents current runtime behavior from `internal/app` and related packages.

### Candidate detection

- Candidate files are discovered by scanning the full target directory tree (unbounded depth).
- Accepted first-volume candidates are:
  - `*.rar` (including single-volume archives)
  - `*.part01.rar` / `*.part1.rar`
  - `*.001`
- Continuation volumes (`.r00`, `.part02.rar`, `.002`, etc.) are not treated as starting candidates.

### Validation and SFV flow

- Each candidate is checked for a RAR signature before extraction.
- If signature validation fails, that candidate is counted as a failure and skipped.
- If SFV is enabled and `<stem>.sfv` exists, all SFV entries are verified.
- If SFV verification fails:
  - without `--force`, extraction is skipped and failure count increases;
  - with `--force`, extraction continues and failure is logged.

### Skip-if-exists behavior

- `--skip-if-exists` is only applied when:
  - `--force` is not set; and
  - `--dry` is not set; and
  - SFV verification did not fail.
- The check compares archive entry names against files in the archive directory (script parity), even when `--output` is set:
  - in `--full-path` mode, entry relative paths are respected;
  - otherwise basenames are used (flatten-style matching).
- If listing/checking fails, extraction continues (best-effort skip gate).

### Extraction destination and collisions

- Archives are extracted into a temp directory under the archive directory, then moved into final destination.
- Final destination root is:
  - `--output` if provided;
  - otherwise the archive directory.
- Moves use rename first, with cross-device copy fallback.
- Existing destination names are never clobbered; `.1`, `.2`, ... suffixes are used when needed.

### Password retry flow

- First extraction attempt is always without a password.
- On password-related errors, passwords from `--password-file` are tried line-by-line.
- First successful password wins.
- If the archive is encrypted and no usable password is available, extraction fails with a password-required error.

### Recursion and depth

- After successful extraction, nested candidate scanning runs inside the temp output.
- `--depth` only controls whether nested recursive passes continue (`depth-1` per recursion level).
- Recursion stops when depth drops below zero.
- Nested failures propagate into parent run failure accounting.

### Cleanup execution rules

- Hooks run only when `--clean` is not `none`.
- In normal mode, hooks run after successful extraction.
- If extraction fails, hooks only run when `--force` is set.
- In `--dry` mode, extraction is not performed, but selected hooks run in dry-run mode (no deletes).

### Security boundaries

- Archive entry paths are sanitized to prevent traversal and absolute-path writes.
- Symlink extraction is disabled by default.
- `--allow-symlinks` enables symlink extraction only when targets remain inside extraction root.
- Decoder dictionary size is capped by default with `--max-dict` (1 GiB).

## Exit Codes

`go-unrarall` exits with:

- `0` when there are no failures.
- `0` when there are failures, `--allow-failures` is set, and at least one candidate succeeded (extracted or skipped).
- `1` for argument/validation errors or when failures remain under all other conditions.

## Troubleshooting

### "does not appear to be a valid rar file"

- Cause: candidate did not pass signature detection.
- Check:
  - file is actually a RAR archive and not mislabeled;
  - you are invoking from the intended root directory.

### SFV verification failures

- Cause: `<stem>.sfv` references missing files or CRC mismatches.
- Check:
  - all release files listed in SFV are present;
  - files are not partially downloaded/corrupted.
- Override: use `--force` to continue extraction despite SFV failure.

### Password failures or encrypted archive errors

- Cause: encrypted archive and no valid password found.
- Check:
  - `--password-file` exists and is readable;
  - one password per line;
  - expected password is included exactly.

### Path/symlink safety errors

- Cause: archive includes unsafe entry paths or symlink targets.
- Check:
  - entry paths do not escape destination (`..`, absolute paths, drive prefixes);
  - symlink extraction is only enabled intentionally via `--allow-symlinks`.

### "skip-if-exists" did not skip

- Cause: one or more expected destination files were missing, or archive listing failed.
- Check:
  - files in the archive directory (skip checks are rooted there for script parity);
  - `--full-path` mode differences;
  - whether `--dry` is set (dry-run bypasses skip checks);
  - whether `--force` was set (which bypasses skip-if-exists).

## Compatibility Notes

- Candidate discovery is name-pattern based and intentionally limited to first-volume starters.
- Signature validation uses a bounded prefix scan rather than full archive parsing.
- Flatten extraction mode can produce suffixed filenames when basename collisions occur.
- Cleanup behavior is deterministic and scoped to implemented hooks only.

## Architecture Reference

For code-level package layout and pipeline internals, see:

- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md)

## Runtime Dependencies

No external archive or checksum binaries are required at runtime.

At runtime, `go-unrarall` uses only:

- the compiled binary itself
- local filesystem access
