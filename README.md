# valsenv

`valsenv` resolves `ref+<backend>://` references inside dotenv (`.env`) files using [helmfile/vals](https://github.com/helmfile/vals). vals's own `eval`/`exec`/`env` commands only accept YAML/JSON, so `valsenv` fills the dotenv gap.

## What it does

You point `valsenv` at a dotenv stream. It reads each line, resolves any value that is a full `ref+` or `secretref+` reference into its real secret, and writes the rendered dotenv back out. Everything else (comments, blank lines, key order, plain values) passes through unchanged.

## Install

```
go install github.com/zoispag/valsenv@latest
```

Prebuilt binaries are also published on the [GitHub Releases](https://github.com/zoispag/valsenv/releases) page. An optional Homebrew tap may be available.

## Usage

```
valsenv render [-f file] [-o file]
```

- `-f file`: input dotenv file. When omitted, `valsenv` reads from stdin.
- `-o file`: output file. When omitted, `valsenv` writes to stdout.

Example:

```
# in.env
API_KEY=ref+doppler://PROJECT/CONFIG/API_KEY
DB_PASSWORD=ref+vault://secret/data/db#/password
TIMEZONE=UTC

valsenv render -f in.env -o out.env
```

Every `ref+` value is replaced with its resolved secret. `TIMEZONE` stays as-is.

### Try it without a backend

The `echo` backend just returns its own path, so you can see the flow with no credentials:

```
printf 'A=ref+echo://hello\nT=UTC\n' | valsenv render
```

Output:

```
A=hello
T=UTC
```

## Exit codes

- `0`: success.
- `1`: resolution or render failure. `valsenv` is fail-closed, so nothing is written on error (stdout stays empty and any `-o` target is left untouched).
- `2`: usage error.

## Backends

Resolution is handled by [vals](https://github.com/helmfile/vals), which supports a wide range of backends: Doppler, Vault, AWS Secrets Manager, GCP Secret Manager, 1Password, SOPS, and more. See the vals documentation for the full list and reference syntax.

The Doppler backend reads `DOPPLER_TOKEN` from the environment.

## Behavior & limitations

- Comments, blank lines, and key order are preserved.
- Only values that are a complete `ref+` or `secretref+` reference are resolved. Plain values and embedded refs are left alone.
- Multiline resolved values are rejected in v1. Simple dotenv consumers split on newlines, so a secret containing a newline would corrupt the file.
- A trailing inline comment after a ref value is not supported.

## License

MIT. See [LICENSE](LICENSE).
