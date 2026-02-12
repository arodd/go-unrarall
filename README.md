# go-unrarall

`go-unrarall` is a self-contained Go CLI for bulk RAR extraction and cleanup.

It is designed to match the behavior of the original `unrarall` shell workflow while avoiding runtime dependencies on external tools like `unrar`, `rar`, `7z`, `cksfv`, `find`, or `sed`.

## What It Does

- Scans a directory tree for first-volume archives:
  - `*.rar`
  - `*.part01.rar` style sets
  - `*.001` style sets
- Extracts archives in-process (including multi-volume sets).
- Optionally verifies SFV CRC32 manifests before extraction.
- Supports password retries from a password file.
- Supports recursive nested extraction up to `--depth`.
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
- `-f, --force`: continue when checks fail and allow cleanup even on extraction failure.
- `--allow-failures`: return exit code 0 when there is at least one success.
- `-s, --disable-cksfv`: disable SFV verification.
- `--clean=SPEC`: `none|all|hook1,hook2`.
- `--full-path`: preserve archive paths while extracting.
- `-o, --output DIR`: output directory (must already exist).
- `--depth N`: recursive scan depth (default `4`).
- `--skip-if-exists`: skip extraction if all archive entries already exist by name.
- `--password-file FILE`: password source file (default `~/.unrar_passwords`).
- `--max-dict BYTES`: max RAR dictionary size (default `1073741824`, 1 GiB).
- `--allow-symlinks`: allow symlink extraction with in-tree target validation.

## Cleanup Hooks

Available hook names for `--clean=`:

- `nfo`
- `rar`
- `osx_junk`
- `windows_junk`
- `covers_folders`
- `proof_folders`
- `sample_folders`
- `sample_videos`
- `empty_folders`

Special values:

- `all`: run all hooks in default order.
- `none`: disable cleanup hooks.

## Security Defaults

- Path traversal protection is enforced for archive entry paths.
- Symlink extraction is disabled by default.
- Symlink extraction is only enabled when `--allow-symlinks` is set, and link targets are validated to stay within the extraction root.
- Decoder dictionary size is capped by default via `--max-dict` (1 GiB default).

## Runtime Dependencies

No external archive or checksum binaries are required at runtime.

At runtime, `go-unrarall` uses only:

- the compiled binary itself
- local filesystem access

