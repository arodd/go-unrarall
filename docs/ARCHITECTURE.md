# Architecture

This document explains how `go-unrarall` is structured and how one archive flows through the runtime pipeline.

## Package map

- `cmd/unrarall/main.go`
  Entry point. Parses CLI args, prints help/version, runs app orchestration, and exits with script-parity exit codes.
- `internal/cli`
  CLI options parsing, validation, and usage text rendering.
- `internal/log`
  Lightweight logger with quiet/info/verbose modes.
- `internal/finder`
  Directory walk and candidate detection for first-volume archives.
- `internal/rar`
  Archive signature checks, multi-volume open settings, listing for skip checks, and stream extraction.
- `internal/sfv`
  SFV parser plus CRC32 verification.
- `internal/app`
  Top-level orchestration logic: per-candidate processing, retries, recursion, cleanup execution, and summary stats.
- `internal/hooks`
  Cleanup hook registry and implementations for `--clean` behavior.
- `internal/fsutil`
  Shared filesystem safety primitives: path sanitization, temp dir creation, and safe move/copy fallback.

## Candidate discovery

Candidate discovery starts in `internal/finder/scan.go`.

- Walks the target directory with `filepath.WalkDir`.
- Enforces `--depth` during the walk (entries deeper than max depth are skipped).
- Accepts only first-volume candidates:
  - any `*.rar` that is not a `partNN` continuation;
  - `*.part01.rar`/`*.part1.rar` style first-part files;
  - `*.001` (and not `.002+`).
- Produces a deterministic, case-insensitive sorted candidate list.

## Archive processing pipeline

For each candidate archive, `internal/app/run.go` executes:

1. Signature validation
- Uses `internal/rar/validate.go` to scan the first SFX window for RAR4/RAR5 signatures.
- Files that fail signature checks are counted as failures and skipped.

2. SFV verification (optional)
- If `<stem>.sfv` exists and SFV is enabled, parse and verify all entries.
- SFV failure blocks extraction unless `--force` is set.

3. Skip-if-exists gate (optional)
- If `--skip-if-exists` is set, and `--force` is not set, and SFV passed:
  - list archive entries;
  - check whether every non-directory entry already exists at destination by name.
- In `--full-path` mode, relative paths are preserved for existence checks.
- In flatten mode, only basenames are checked.

4. Extraction
- Normal run:
  - create a temp extraction directory under the archive directory;
  - extract archive entries into temp using stream extraction.
- Dry run (`--dry`):
  - skip extraction and filesystem writes;
  - log what would be extracted.

5. Password retries (if needed)
- First extraction attempt uses no password.
- Password errors trigger line-by-line retries from `--password-file`.
- Non-password extraction errors fail immediately.

6. Nested recursion
- After successful extraction, nested candidate scanning runs on the temp directory with `depth-1`.
- Nested failures propagate using the same exit-code logic.

7. Move to destination
- Artifacts are moved from temp into destination root (`--output` or archive directory).
- Move logic uses rename first, with cross-device copy/remove fallback.
- Destination collisions are avoided with `.1`, `.2`, ... suffixes.

8. Cleanup hooks
- If `--clean` selects hooks, hooks run:
  - after success; or
  - after extraction failure only when `--force` is set.
- In dry-run mode hooks execute in dry-run behavior (no deletes).

9. Stats and summary
- Tracks found/extracted/skipped/failure counters.
- Process exit code is derived from failure count and `--allow-failures`.

## Recursion model

Recursion lives in `internal/app/recursive.go`.

- Top-level run scans the initial directory using `--depth`.
- Each successful extraction can trigger a nested run on temp output with depth decremented by one.
- When the decremented depth is below zero, recursion stops.
- Nested run failures are treated as candidate failures in the parent run.

## Safety boundaries

`go-unrarall` enforces safety checks at multiple layers.

- Path traversal protection
  - Archive entry paths are normalized and sanitized.
  - Absolute paths, `..` escapes, and drive-prefixed paths are rejected.
- Symlink policy
  - Symlink extraction is disabled by default.
  - `--allow-symlinks` opt-in is required.
  - Symlink targets are decoded and validated to stay in-tree.
- Dictionary size cap
  - Decoder max dictionary size defaults to 1 GiB (`--max-dict`).
- Supported artifact types
  - Extracted artifacts are expected to be regular files/directories (plus symlinks only when explicitly allowed).

## Cleanup hook engine

Hooks are implemented in `internal/hooks` and selected by `--clean=none|all|hook1,hook2`.

Available hooks:

- `nfo`
- `rar`
- `osx_junk`
- `windows_junk`
- `covers_folders`
- `proof_folders`
- `sample_folders`
- `sample_videos`
- `empty_folders`

`all` runs hooks in registry order, and `none` disables cleanup.
