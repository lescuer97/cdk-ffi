use std::collections::HashMap;
use std::sync::Arc;

use cdk::amount::SplitTarget;
use cdk::nuts::nut00::ProofsMethods;
use cdk::nuts::{CurrencyUnit, MintQuoteState};
use cdk::wallet::{PreparedSend, SendMemo, SendOptions, Wallet as CdkWallet};
use cdk::Amount;
use cdk_common::common::Melted;
use cdk_common::database::WalletDatabase;
use cdk_common::wallet::{MeltQuote, MintQuote, SendKind};

use bip39::Mnemonic;
use tokio::runtime::Runtime;

// Export the uniffi bindings
uniffi::setup_scaffolding!();

/// Generate a 12-word mnemonic phrase
#[uniffi::export]
pub fn generate_mnemonic() -> Result<String> {
    let mnemonic = Mnemonic::generate(12).map_err(|e| FFIError::InternalError {
        msg: format!("Failed to generate mnemonic: {}", e),
    })?;
    Ok(mnemonic.to_string())
}

/// Convert a mnemonic phrase to a 64-byte seed for wallet creation
fn mnemonic_to_seed(mnemonic_words: String) -> Result<[u8; 64]> {
    let mnemonic = Mnemonic::parse(&mnemonic_words).map_err(|e| FFIError::InvalidInput {
        msg: format!("Invalid mnemonic: {}", e),
    })?;

    Ok(mnemonic.to_seed_normalized(""))
}

// Error handling
#[derive(Debug, thiserror::Error, uniffi::Error)]
pub enum FFIError {
    #[error("Wallet error: {msg}")]
    WalletError { msg: String },

    #[error("Invalid input: {msg}")]
    InvalidInput { msg: String },

    #[error("Network error: {msg}")]
    NetworkError { msg: String },

    #[error("Internal error: {msg}")]
    InternalError { msg: String },
}

impl From<cdk::error::Error> for FFIError {
    fn from(err: cdk::error::Error) -> Self {
        FFIError::WalletError {
            msg: err.to_string(),
        }
    }
}

impl From<cdk_common::database::Error> for FFIError {
    fn from(err: cdk_common::database::Error) -> Self {
        FFIError::WalletError {
            msg: err.to_string(),
        }
    }
}

impl From<cdk::nuts::nut00::Error> for FFIError {
    fn from(err: cdk::nuts::nut00::Error) -> Self {
        FFIError::WalletError {
            msg: err.to_string(),
        }
    }
}

impl From<cdk_sqlite::wallet::error::Error> for FFIError {
    fn from(err: cdk_sqlite::wallet::error::Error) -> Self {
        FFIError::WalletError {
            msg: err.to_string(),
        }
    }
}

type Result<T> = std::result::Result<T, FFIError>;

// Helper to create a tokio runtime
fn runtime() -> Runtime {
    Runtime::new().expect("Failed to create tokio runtime")
}

// Records (pass by value) - simple data structures

#[derive(uniffi::Record)]
pub struct FFIAmount {
    pub value: u64,
}

impl From<Amount> for FFIAmount {
    fn from(amount: Amount) -> Self {
        Self {
            value: amount.into(),
        }
    }
}

impl From<FFIAmount> for Amount {
    fn from(ffi_amount: FFIAmount) -> Self {
        Amount::from(ffi_amount.value)
    }
}

#[derive(uniffi::Record)]
pub struct FFIMintQuote {
    pub id: String,
    pub mint_url: String,
    pub amount: FFIAmount,
    pub unit: String,
    pub request: String,
    pub state: FFIMintQuoteState,
    pub expiry: u64,
}

impl From<MintQuote> for FFIMintQuote {
    fn from(quote: MintQuote) -> Self {
        Self {
            id: quote.id,
            mint_url: quote.mint_url.to_string(),
            amount: quote.amount.into(),
            unit: quote.unit.to_string(),
            request: quote.request,
            state: quote.state.into(),
            expiry: quote.expiry,
        }
    }
}

#[derive(uniffi::Record)]
pub struct FFIMintQuoteBolt11Response {
    pub quote: String,
    pub request: String,
    pub state: FFIMintQuoteState,
    pub expiry: Option<u64>,
}

impl From<cdk::nuts::MintQuoteBolt11Response<String>> for FFIMintQuoteBolt11Response {
    fn from(response: cdk::nuts::MintQuoteBolt11Response<String>) -> Self {
        Self {
            quote: response.quote,
            request: response.request,
            state: response.state.into(),
            expiry: response.expiry,
        }
    }
}

#[derive(uniffi::Record)]
pub struct FFIMeltQuote {
    pub id: String,
    pub unit: String,
    pub amount: FFIAmount,
    pub request: String,
    pub fee_reserve: FFIAmount,
    pub expiry: u64,
    pub payment_preimage: Option<String>,
}

impl From<MeltQuote> for FFIMeltQuote {
    fn from(quote: MeltQuote) -> Self {
        Self {
            id: quote.id,
            unit: quote.unit.to_string(),
            amount: quote.amount.into(),
            request: quote.request,
            fee_reserve: quote.fee_reserve.into(),
            expiry: quote.expiry,
            payment_preimage: quote.payment_preimage,
        }
    }
}

#[derive(uniffi::Record)]
pub struct FFIMelted {
    pub state: String,
    pub preimage: Option<String>,
    pub amount: FFIAmount,
    pub fee_paid: FFIAmount,
}

impl From<Melted> for FFIMelted {
    fn from(melted: Melted) -> Self {
        Self {
            state: melted.state.to_string(),
            preimage: melted.preimage,
            amount: melted.amount.into(),
            fee_paid: melted.fee_paid.into(),
        }
    }
}

#[derive(uniffi::Record)]
pub struct FFIToken {
    pub token_string: String,
    pub mint: String,
    pub memo: Option<String>,
    pub unit: String,
}

impl TryFrom<cdk::nuts::Token> for FFIToken {
    type Error = FFIError;

    fn try_from(token: cdk::nuts::Token) -> Result<Self> {
        let mint_url = token
            .mint_url()
            .map_err(|e| FFIError::WalletError { msg: e.to_string() })?
            .to_string();

        let token_str = token.to_string();

        Ok(Self {
            token_string: token_str,
            mint: mint_url,
            memo: token.memo().clone(),
            unit: token.unit().map(|u| u.to_string()).unwrap_or_default(),
        })
    }
}

#[derive(uniffi::Record)]
pub struct FFISendOptions {
    pub memo: Option<FFISendMemo>,
    pub amount_split_target: FFISplitTarget,
    pub send_kind: FFISendKind,
    pub include_fee: bool,
    pub metadata: HashMap<String, String>,
    pub max_proofs: Option<u64>,
}

impl From<FFISendOptions> for SendOptions {
    fn from(options: FFISendOptions) -> Self {
        Self {
            memo: options.memo.map(|m| m.into()),
            conditions: None, // TODO: Add support for spending conditions
            amount_split_target: options.amount_split_target.into(),
            send_kind: options.send_kind.into(),
            include_fee: options.include_fee,
            metadata: options.metadata,
            max_proofs: options.max_proofs.map(|p| p as usize),
        }
    }
}

#[derive(uniffi::Record)]
pub struct FFISendMemo {
    pub memo: String,
    pub include_memo: bool,
}

impl From<FFISendMemo> for SendMemo {
    fn from(memo: FFISendMemo) -> Self {
        SendMemo {
            memo: memo.memo,
            include_memo: memo.include_memo,
        }
    }
}

#[derive(uniffi::Record)]
pub struct FFIPreparedSend {
    pub amount: FFIAmount,
    pub swap_fee: FFIAmount,
    pub send_fee: FFIAmount,
    pub total_fee: FFIAmount,
}

impl From<PreparedSend> for FFIPreparedSend {
    fn from(send: PreparedSend) -> Self {
        Self {
            amount: send.amount().into(),
            swap_fee: send.swap_fee().into(),
            send_fee: send.send_fee().into(),
            total_fee: send.fee().into(),
        }
    }
}

// Enums

#[derive(uniffi::Enum)]
pub enum FFIMintQuoteState {
    Unpaid,
    Paid,
    Issued,
}

impl From<MintQuoteState> for FFIMintQuoteState {
    fn from(state: MintQuoteState) -> Self {
        match state {
            MintQuoteState::Unpaid => Self::Unpaid,
            MintQuoteState::Paid => Self::Paid,
            MintQuoteState::Issued => Self::Issued,
            _ => Self::Unpaid, // Handle any other states
        }
    }
}

#[derive(uniffi::Enum)]
pub enum FFISplitTarget {
    None,
    Default,
}

impl From<FFISplitTarget> for SplitTarget {
    fn from(target: FFISplitTarget) -> Self {
        match target {
            FFISplitTarget::None => SplitTarget::None,
            FFISplitTarget::Default => SplitTarget::default(),
        }
    }
}

#[derive(uniffi::Enum)]
pub enum FFISendKind {
    OnlineExact,
    OnlineTolerance { tolerance: FFIAmount },
    OfflineExact,
    OfflineTolerance { tolerance: FFIAmount },
}

impl From<FFISendKind> for SendKind {
    fn from(kind: FFISendKind) -> Self {
        match kind {
            FFISendKind::OnlineExact => SendKind::OnlineExact,
            FFISendKind::OnlineTolerance { tolerance } => {
                SendKind::OnlineTolerance(tolerance.into())
            }
            FFISendKind::OfflineExact => SendKind::OfflineExact,
            FFISendKind::OfflineTolerance { tolerance } => {
                SendKind::OfflineTolerance(tolerance.into())
            }
        }
    }
}

#[derive(uniffi::Enum)]
pub enum FFICurrencyUnit {
    Sat,
    Msat,
    Usd,
    Eur,
}

impl TryFrom<String> for FFICurrencyUnit {
    type Error = FFIError;

    fn try_from(unit: String) -> Result<Self> {
        match unit.to_lowercase().as_str() {
            "sat" => Ok(Self::Sat),
            "msat" => Ok(Self::Msat),
            "usd" => Ok(Self::Usd),
            "eur" => Ok(Self::Eur),
            _ => Err(FFIError::InvalidInput {
                msg: format!("Unknown currency unit: {}", unit),
            }),
        }
    }
}

impl From<FFICurrencyUnit> for CurrencyUnit {
    fn from(unit: FFICurrencyUnit) -> Self {
        match unit {
            FFICurrencyUnit::Sat => CurrencyUnit::Sat,
            FFICurrencyUnit::Msat => CurrencyUnit::Msat,
            FFICurrencyUnit::Usd => CurrencyUnit::Usd,
            FFICurrencyUnit::Eur => CurrencyUnit::Eur,
        }
    }
}

// Objects (pass by reference) - stateful objects

#[derive(uniffi::Object)]
pub struct FFILocalStore {
    inner: Arc<dyn WalletDatabase<Err = cdk_common::database::Error> + Send + Sync>,
}

#[uniffi::export]
impl FFILocalStore {
    #[uniffi::constructor]
    pub fn new() -> Result<Arc<Self>> {
        Self::new_with_path(None)
    }

    #[uniffi::constructor]
    pub fn new_with_path(db_path: Option<String>) -> Result<Arc<Self>> {
        let rt = runtime();
        let store = rt.block_on(async {
            let final_db_path = match db_path {
                Some(custom_path) => {
                    // Use the provided path directly
                    custom_path
                }
                None => {
                    // Fallback to temp directory (original behavior)
                    let temp_path = std::env::temp_dir()
                        .join(format!("cdk_wallet_{}.db", uuid::Uuid::new_v4()));
                    temp_path.to_string_lossy().to_string()
                }
            };
            cdk_sqlite::WalletSqliteDatabase::new(&final_db_path).await
        })?;
        Ok(Arc::new(Self {
            inner: Arc::new(store),
        }))
    }
}

#[derive(uniffi::Object)]
pub struct FFIWallet {
    inner: CdkWallet,
    runtime: Runtime,
}

#[uniffi::export]
impl FFIWallet {
    #[uniffi::constructor]
    pub fn from_mnemonic(
        mint_url: String,
        unit: FFICurrencyUnit,
        localstore: Arc<FFILocalStore>,
        mnemonic_words: String,
    ) -> Result<Arc<Self>> {
        let seed = mnemonic_to_seed(mnemonic_words.clone())?;

        let wallet = CdkWallet::new(
            &mint_url,
            unit.into(),
            localstore.inner.clone(),
            &seed,
            None,
        )?;

        Ok(Arc::new(Self {
            inner: wallet,
            runtime: runtime(),
        }))
    }

    #[uniffi::constructor]
    pub fn restore_from_mnemonic(
        mint_url: String,
        unit: FFICurrencyUnit,
        localstore: Arc<FFILocalStore>,
        mnemonic_words: String,
    ) -> Result<Arc<Self>> {
        let seed = mnemonic_to_seed(mnemonic_words.clone())?;

        let wallet = CdkWallet::new(
            &mint_url,
            unit.into(),
            localstore.inner.clone(),
            &seed,
            None,
        )?;

        let runtime = runtime();

        // Call restore on the wallet
        runtime.block_on(async { wallet.restore().await })?;

        Ok(Arc::new(Self {
            inner: wallet,
            runtime,
        }))
    }

    pub fn mint_quote(
        &self,
        amount: FFIAmount,
        description: Option<String>,
    ) -> Result<FFIMintQuote> {
        self.runtime.block_on(async {
            let quote = self.inner.mint_quote(amount.into(), description).await?;
            Ok(quote.into())
        })
    }

    pub fn mint_quote_state(&self, quote_id: String) -> Result<FFIMintQuoteBolt11Response> {
        self.runtime.block_on(async {
            let state = self.inner.mint_quote_state(&quote_id).await?;
            Ok(state.into())
        })
    }

    pub fn mint(&self, quote_id: String, split_target: FFISplitTarget) -> Result<FFIAmount> {
        self.runtime.block_on(async {
            let proofs = self
                .inner
                .mint(&quote_id, split_target.into(), None)
                .await?;
            let amount = proofs.total_amount()?;
            Ok(amount.into())
        })
    }

    pub fn prepare_send(
        &self,
        amount: FFIAmount,
        options: FFISendOptions,
    ) -> Result<FFIPreparedSend> {
        self.runtime.block_on(async {
            let prepared = self
                .inner
                .prepare_send(amount.into(), options.into())
                .await?;
            Ok(prepared.into())
        })
    }

    pub fn send(
        &self,
        amount: FFIAmount,
        options: FFISendOptions,
        memo: Option<FFISendMemo>,
    ) -> Result<FFIToken> {
        self.runtime.block_on(async {
            // First prepare the send
            let prepared = self
                .inner
                .prepare_send(amount.into(), options.into())
                .await?;

            // Then send it
            let token = self.inner.send(prepared, memo.map(|m| m.into())).await?;
            Ok(token.try_into()?)
        })
    }

    pub fn balance(&self) -> Result<FFIAmount> {
        self.runtime.block_on(async {
            let balance = self.inner.total_balance().await?;
            Ok(balance.into())
        })
    }

    pub fn mint_url(&self) -> String {
        self.inner.mint_url.to_string()
    }

    pub fn unit(&self) -> String {
        self.inner.unit.to_string()
    }

    /// Fetch and initialize mint information
    /// This should be called after wallet creation to set up the mint in the database
    pub fn get_mint_info(&self) -> Result<String> {
        self.runtime.block_on(async {
            // First try to get existing mint info from database
            match self.inner.get_mint_info().await? {
                Some(mint_info) => {
                    let name = mint_info.name.unwrap_or_else(|| "Unknown Mint".to_string());
                    Ok(format!("Mint info already initialized: {}", name))
                },
                None => {
                    // Mint info not in database, try to fetch and initialize
                    // First get the mint keysets which should also fetch mint info
                    match self.inner.get_mint_keysets().await {
                        Ok(keysets) => {
                            // Check if mint info is now available
                            match self.inner.get_mint_info().await? {
                                Some(mint_info) => {
                                    let name = mint_info.name.unwrap_or_else(|| "Unknown Mint".to_string());
                                    Ok(format!("Mint info fetched and initialized: {} (keysets: {})", 
                                              name, keysets.len()))
                                },
                                None => {
                                    Ok(format!("Mint keysets loaded ({} keysets) but mint info still not available", 
                                              keysets.len()))
                                }
                            }
                        },
                        Err(e) => {
                            Ok(format!("Failed to get mint keysets: {}", e))
                        }
                    }
                }
            }
        })
    }

    /// Create a melt quote for paying a Lightning invoice
    pub fn melt_quote(&self, request: String) -> Result<FFIMeltQuote> {
        self.runtime.block_on(async {
            let quote = self.inner.melt_quote(request, None).await?;
            Ok(quote.into())
        })
    }

    /// Execute a melt operation (pay Lightning invoice)
    pub fn melt(&self, quote_id: String) -> Result<FFIMelted> {
        self.runtime.block_on(async {
            let result = self.inner.melt(&quote_id).await?;
            Ok(result.into())
        })
    }
}
