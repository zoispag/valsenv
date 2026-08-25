# Releasing

Releases are automated with [GoReleaser](https://goreleaser.com/) via the
`Release` GitHub Actions workflow. Pushing a `vX.Y.Z` tag builds cross-platform
binaries, publishes a GitHub Release, and updates the Homebrew formula in the
`zoispag/homebrew-tap` repository.

## Cutting a release

```
git tag vX.Y.Z
git push origin vX.Y.Z
```

The workflow then produces:

- A GitHub Release for the tag.
- Binaries for linux/darwin/windows across amd64/arm64 (`tar.gz`, `zip` on Windows).
- A `checksums.txt` file.
- An updated `Casks/valsenv.rb` in `zoispag/homebrew-tap`.

Users install with:

```
brew install --cask zoispag/tap/valsenv
```

## One-time Homebrew tap setup

The built-in `GITHUB_TOKEN` cannot push to a different repository, so a GitHub
App installation token scoped to the tap repo is used to bump the formula.

1. Create the tap repository `zoispag/homebrew-tap` if it does not exist (public).
2. Create a GitHub App under the `zoispag` account
   (Settings → Developer settings → GitHub Apps → New GitHub App).
3. Grant it exactly one permission: Repository → **Contents: Read and write**.
   No other permissions are required.
4. Install the App on the `zoispag/homebrew-tap` repository.
5. Generate a private key for the App and note its App ID.
6. In the `valsenv` repo, go to Settings → Secrets and variables → Actions and
   add two secrets:
   - `TAP_APP_ID` — the App ID.
   - `TAP_APP_PRIVATE_KEY` — the App private key (full PEM contents).

The release workflow mints a short-lived installation token from these secrets
and passes it to GoReleaser as `HOMEBREW_TAP_GITHUB_TOKEN`.
