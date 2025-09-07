package main

import "go_dir/cdk_ffi"

// Amount represents a monetary amount with a uint64 value
type Amount struct {
	Value uint64
}

// SplitTarget represents the target for splitting proofs
type SplitTarget uint

const (
	SplitTargetNone    SplitTarget = SplitTarget(cdk_ffi.FfiSplitTargetNone)
	SplitTargetDefault SplitTarget = SplitTarget(cdk_ffi.FfiSplitTargetDefault)
)

type Wallet struct {
	wallet cdk_ffi.FfiWalletInterface
}

type Storage struct {
	storage *cdk_ffi.FfiLocalStore
}

func NewStorage() (Storage, error) {
	storage, err := cdk_ffi.NewFfiLocalStore()
	if err != nil {
		return Storage{storage: storage}, err
	}

	return Storage{storage: storage}, nil
}

func NewStorageFromPath(path string) (Storage, error) {
	storage, err := cdk_ffi.FfiLocalStoreNewWithPath(&path)
	if err != nil {
		return Storage{storage: storage}, err
	}

	return Storage{storage: storage}, nil
}

type Unit = cdk_ffi.FfiCurrencyUnit

const Sat Unit = Unit(cdk_ffi.FfiCurrencyUnitSat)

func RestoreFromMnemonic(minturl string, unit Unit, storage Storage, mnemonic string) (*Wallet, error) {
	wallet, err := cdk_ffi.FfiWalletRestoreFromMnemonic(minturl, cdk_ffi.FfiCurrencyUnit(unit), storage.storage, mnemonic)
	if err != nil {
		return nil, err
	}
	return &Wallet{
		wallet: wallet,
	}, nil
}

func NewWalletFromMnemonic(minturl string, unit Unit, storage Storage, mnemonic string) (*Wallet, error) {
	wallet, err := cdk_ffi.FfiWalletFromMnemonic(minturl, cdk_ffi.FfiCurrencyUnit(unit), storage.storage, mnemonic)
	if err != nil {
		return nil, err
	}
	return &Wallet{
		wallet: wallet,
	}, nil
}

// Balance returns the wallet's balance
func (w *Wallet) Balance() (Amount, error) {
	amount, err := w.wallet.Balance()
	if err != nil {
		return Amount{}, err
	}
	return Amount{Value: amount.Value}, nil
}

// GetMintInfo fetches and initializes mint information
// This should be called after wallet creation to set up the mint in the database
func (w *Wallet) GetMintInfo() (string, error) {
	return w.wallet.GetMintInfo()
}

// MintUrl returns the mint URL
func (w *Wallet) MintUrl() string {
	return w.wallet.MintUrl()
}

// PreparedSend is a Go-native representation of cdk_ffi.FfiPreparedSend
type PreparedSend struct {
	Amount   Amount
	SwapFee  Amount
	SendFee  Amount
	TotalFee Amount
}

// PrepareSend prepares a send operation using Go-native SendOptions
func (w *Wallet) PrepareSend(amount Amount, options SendOptions) (PreparedSend, error) {
	ffiOptions := options.ToFFI()
	ffiPrepared, err := w.wallet.PrepareSend(cdk_ffi.FfiAmount{Value: amount.Value}, ffiOptions)
	if err != nil {
		return PreparedSend{}, err
	}
	return PreparedSend{
		Amount:   Amount{Value: ffiPrepared.Amount.Value},
		SwapFee:  Amount{Value: ffiPrepared.SwapFee.Value},
		SendFee:  Amount{Value: ffiPrepared.SendFee.Value},
		TotalFee: Amount{Value: ffiPrepared.TotalFee.Value},
	}, nil
}

// Send sends tokens using Go-native SendOptions and SendMemo
func (w *Wallet) Send(amount Amount, options SendOptions, memo *SendMemo) (Token, error) {
	ffiOptions := options.ToFFI()
	var ffiMemo *cdk_ffi.FfiSendMemo
	if memo != nil {
		ffiMemo = memo.ToFFI()
	}
	ffiToken, err := w.wallet.Send(cdk_ffi.FfiAmount{Value: amount.Value}, ffiOptions, ffiMemo)
	if err != nil {
		return Token{}, err
	}
	return TokenFromFFI(ffiToken), nil
}

// MeltQuote is a Go-native representation of cdk_ffi.FfiMeltQuote
type MeltQuote struct {
	Id              string
	Unit            string
	Amount          Amount
	Request         string
	FeeReserve      Amount
	Expiry          uint64
	PaymentPreimage *string
}

// MeltQuote creates a melt quote for paying a Lightning invoice
func (w *Wallet) MeltQuote(request string) (MeltQuote, error) {
	f, err := w.wallet.MeltQuote(request)
	if err != nil {
		return MeltQuote{}, err
	}
	return MeltQuote{
		Id:              f.Id,
		Unit:            f.Unit,
		Amount:          Amount{Value: f.Amount.Value},
		Request:         f.Request,
		FeeReserve:      Amount{Value: f.FeeReserve.Value},
		Expiry:          f.Expiry,
		PaymentPreimage: f.PaymentPreimage,
	}, nil
}

// Melted is a Go-native representation of cdk_ffi.FfiMelted
type Melted struct {
	State    string
	Preimage *string
	Amount   Amount
	FeePaid  Amount
}

// Melt executes a melt operation (pay Lightning invoice)
func (w *Wallet) Melt(quoteId string) (Melted, error) {
	m, err := w.wallet.Melt(quoteId)
	if err != nil {
		return Melted{}, err
	}
	return Melted{
		State:    m.State,
		Preimage: m.Preimage,
		Amount:   Amount{Value: m.Amount.Value},
		FeePaid:  Amount{Value: m.FeePaid.Value},
	}, nil
}

// Mint mints tokens from a quote
func (w *Wallet) Mint(quoteId string, splitTarget SplitTarget) (Amount, error) {
	amount, err := w.wallet.Mint(quoteId, cdk_ffi.FfiSplitTarget(splitTarget))
	if err != nil {
		return Amount{}, err
	}
	return Amount{Value: amount.Value}, nil
}

// MintQuote creates a mint quote for a specific amount
func (w *Wallet) MintQuote(amount Amount, description *string) (cdk_ffi.FfiMintQuote, error) {
	return w.wallet.MintQuote(cdk_ffi.FfiAmount{Value: amount.Value}, description)
}

// MintQuoteState gets the state of a mint quote
func (w *Wallet) MintQuoteState(quoteId string) (cdk_ffi.FfiMintQuoteBolt11Response, error) {
	return w.wallet.MintQuoteState(quoteId)
}

// Unit returns the wallet's currency unit
func (w *Wallet) Unit() string {
	return w.wallet.Unit()
}
