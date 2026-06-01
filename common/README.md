# common

Lightly-modernized copies of `Hash` and supporting helpers from
[go-ethereum](https://github.com/ethereum/go-ethereum)'s
`common` and `common/hexutil` packages. Modifications listed
under [Local edits on top of upstream](#local-edits-on-top-of-upstream).

## Why a copy and not a build dep

Pulling `github.com/ethereum/go-ethereum` as a module dependency
brings the entire geth dependency tree (cgo, kzg, RLP, secp256k1
bindings, holiman/uint256, p2p stack, EVM, …) — hundreds of MB and
hundreds of transitive packages, almost none of which we use. We
only need a 32-byte hash type with the Ethereum-canonical
0x-prefixed hex serialization, plus the `hexutil` helpers it
delegates to. Copying ~600 lines is dramatically smaller blast
radius than vendoring the full module.

## Provenance

Imported from go-ethereum **v1.17.2** (latest stable at the time
of import):

| Our file                | Upstream path                       |
|-------------------------|-------------------------------------|
| `bytes.go`              | `common/bytes.go`                   |
| `bytes_test.go`         | `common/bytes_test.go`              |
| `hash.go`               | `common/types.go` (Hash bits only)  |
| `hash_test.go`          | `common/types_test.go` (Hash tests) |
| `hexutil/hexutil.go`    | `common/hexutil/hexutil.go`         |
| `hexutil/hexutil_test.go` | `common/hexutil/hexutil_test.go`  |
| `hexutil/json.go`       | `common/hexutil/json.go`            |
| `hexutil/json_test.go`  | `common/hexutil/json_test.go`       |

## Local edits on top of upstream

These files are **not** byte-identical to upstream. The diff is
small and intentional — keep this section accurate when
re-syncing so the modifications can be replayed cleanly.

### Structural (content removal)

- `hash.go` is `types.go` with `Address`, `MixedcaseAddress`,
  `AddressEIP55`, `UnprefixedAddress`, `Decimal`, `PrettyBytes`
  removed. Only `Hash`, `UnprefixedHash`, `IsHexHash` (new in
  v1.17.2), and supporting consts/vars remain. Imports trimmed
  accordingly.
- `hash_test.go` keeps only the five Hash-related test functions:
  `TestBytesConversion`, `TestHashJsonValidation`, `TestHash_Scan`,
  `TestHash_Value`, `TestHash_Format`.
- `hexutil/json.go` drops the `U256` type and the
  `github.com/holiman/uint256` import.
- `hexutil/json_test.go` correspondingly drops `TestUnmarshalU256`
  and the `unmarshalU256Tests` fixture.

### Import-path rewrites

- `bytes.go` and `hash.go`:
  `github.com/ethereum/go-ethereum/common/hexutil` →
  `github.com/polymerdao/utils/common/hexutil`.

### Lint-driven modernizations

The project lints under the relayer-style golangci config; the
following one-liners were applied so the imported files lint
clean without per-path exclusions:

- `interface{}` → `any` everywhere it appeared:
  `hash.go` (Scan, UnmarshalGraphQL),
  `hash_test.go` (Scan args struct),
  `hexutil/json.go` (Bytes/Big.UnmarshalGraphQL),
  `hexutil/hexutil_test.go` (marshalTest, unmarshalTest fields).
- `hexutil/hexutil.go` `DecodeBig`: `if start < 0 { start = 0 }`
  collapsed to `start := max(end-bigWordNibbles, 0)` (Go 1.21+
  builtin).
- `hash.go` `Hash.Generate`: unused `size int` parameter renamed
  to `_ int`.
- `hash.go` `Hash.Format`: function-level
  `//nolint:errcheck` directive added (writes to `fmt.State` are
  best-effort by API contract — upstream geth doesn't check the
  return either).

## Re-syncing from upstream

When bumping the source version:

1. Diff upstream against the previously-imported version (e.g.
   `git diff v1.17.2..vNEW -- common/types.go common/bytes.go
   common/hexutil/`).
2. Apply the diff to our copies.
3. Re-apply the **Local edits on top of upstream** above. Most are
   trivial sed-able transformations; the structural removals are
   stable as long as upstream doesn't add new methods to
   `Address`/`Decimal`/etc.
4. Update the version in **Provenance** and adjust this section
   if the set of local edits changes.
5. `go test ./common/...` — the imported tests are the contract.

The general principle: when a lint rule fires on the imported
code, prefer fixing it inline (and recording the edit here) over
adding a path exclusion — exclusions silently hide real bugs that
re-syncs might introduce.

## License

go-ethereum is **LGPL-3.0**. The vendored files retain their
original copyright headers and license, which is the
LGPL's only requirement for redistributing the source.

LGPL-3.0 lets us copy, modify, and link these files into a project
under any license, provided that:

1. The file-level copyright and license notices are preserved
   (they are — see the headers).
2. Anyone we distribute a binary to can obtain the (possibly
   modified) source for the LGPL'd parts and the means to relink.
   For a Go binary that statically links these files, this is
   satisfied by making the source in this directory available.
3. Modifications to the LGPL'd files themselves remain
   LGPL-3.0. The rest of the module (everything outside this
   `common/` tree) is unaffected by the LGPL terms and stays under
   whatever license the project chooses.

This is the same arrangement many Go projects use to consume
small pieces of geth (`erigon`, `op-geth` forks, libraries that
re-export hex/RLP types) — file-level LGPL preservation, project
license elsewhere.

> **Not legal advice.** If the project's distribution model
> changes (closed-source binary, embedded firmware, etc.), have
> counsel re-confirm. The summary above is the consensus
> interpretation of LGPL-3.0 for source-level vendoring of small
> components, not a legal opinion.
