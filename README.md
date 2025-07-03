# cdk-ffi

FFI bindings for the [Cashu Development Kit (CDK)](https://github.com/cashubtc/cdk) wallet, powered by [UniFFI](https://mozilla.github.io/uniffi-rs/).

This crate exposes the Rust wallet API to multiple languages (Swift, Kotlin/Java, Python, JavaScript, …) with a single source of truth.

* High-level helpers for common Cashu wallet operations

---

## Features

| Capability | Function(s) |
|------------|-------------|
| Generate 12-word mnemonic | `generate_mnemonic()` |
| Create / restore wallet from mnemonic | `FFIWallet::from_mnemonic`, `FFIWallet::restore_from_mnemonic` |
| Mint tokens (request / pay / receive) | `mint_quote`, `mint_quote_state`, `mint` |
| Send tokens | `prepare_send`, `send` |
| Melt (pay LN invoice) | `melt_quote`, `melt` |
| Query balance and metadata | `balance`, `mint_url`, `unit`, `get_mint_info` |

All amounts are handled with the CDK `Amount` new-type and exposed as the simple record `FFIAmount` for foreign languages.
