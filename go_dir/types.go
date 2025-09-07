package main

import (
	"go_dir/cdk_ffi"
)

// MintQuoteState is a Go-native enum matching cdk_ffi.FfiMintQuoteState
type MintQuoteState uint

const (
	MintQuoteStateUnpaid MintQuoteState = 1
	MintQuoteStatePaid   MintQuoteState = 2
	MintQuoteStateIssued MintQuoteState = 3
)

func (s MintQuoteState) ToFFI() cdk_ffi.FfiMintQuoteState {
	return cdk_ffi.FfiMintQuoteState(s)
}

// MintQuote is a Go-native representation of cdk_ffi.FfiMintQuote
type MintQuote struct {
	Id      string
	MintUrl string
	Amount  Amount
	Unit    string
	Request string
	State   MintQuoteState
	Expiry  uint64
}

func MintQuoteFromFFI(f cdk_ffi.FfiMintQuote) MintQuote {
	return MintQuote{
		Id:      f.Id,
		MintUrl: f.MintUrl,
		Amount:  Amount{Value: f.Amount.Value},
		Unit:    f.Unit,
		Request: f.Request,
		State:   MintQuoteStateFromFFI(f.State),
		Expiry:  f.Expiry,
	}
}

func (m MintQuote) ToFFI() cdk_ffi.FfiMintQuote {
	return cdk_ffi.FfiMintQuote{
		Id:      m.Id,
		MintUrl: m.MintUrl,
		Amount:  cdk_ffi.FfiAmount{Value: m.Amount.Value},
		Unit:    m.Unit,
		Request: m.Request,
		State:   m.State.ToFFI(),
		Expiry:  m.Expiry,
	}
}

// MintQuoteBolt11 is a Go-native representation of cdk_ffi.FfiMintQuoteBolt11Response
type MintQuoteBolt11 struct {
	Quote   string
	Request string
	State   MintQuoteState
	Expiry  *uint64
}

func MintQuoteBolt11FromFFI(f cdk_ffi.FfiMintQuoteBolt11Response) MintQuoteBolt11 {
	return MintQuoteBolt11{
		Quote:   f.Quote,
		Request: f.Request,
		State:   MintQuoteStateFromFFI(f.State),
		Expiry:  f.Expiry,
	}
}

func (m MintQuoteBolt11) ToFFI() cdk_ffi.FfiMintQuoteBolt11Response {
	return cdk_ffi.FfiMintQuoteBolt11Response{
		Quote:   m.Quote,
		Request: m.Request,
		State:   m.State.ToFFI(),
		Expiry:  m.Expiry,
	}
}

func MintQuoteStateFromFFI(v cdk_ffi.FfiMintQuoteState) MintQuoteState {
	return MintQuoteState(v)
}

// SendMemo represents a memo for sending tokens
type SendMemo struct {
	Memo        string
	IncludeMemo bool
}

func (m *SendMemo) ToFFI() *cdk_ffi.FfiSendMemo {
	if m == nil {
		return nil
	}
	return &cdk_ffi.FfiSendMemo{
		Memo:        m.Memo,
		IncludeMemo: m.IncludeMemo,
	}
}

func SendMemoFromFFI(f *cdk_ffi.FfiSendMemo) *SendMemo {
	if f == nil {
		return nil
	}
	return &SendMemo{
		Memo:        f.Memo,
		IncludeMemo: f.IncludeMemo,
	}
}

// Token is a Go-native representation of cdk_ffi.FfiToken
type Token struct {
	TokenString string
	Mint        string
	Memo        *string
	Unit        string
}

func TokenFromFFI(f cdk_ffi.FfiToken) Token {
	return Token{
		TokenString: f.TokenString,
		Mint:        f.Mint,
		Memo:        f.Memo,
		Unit:        f.Unit,
	}
}

func (t Token) ToFFI() cdk_ffi.FfiToken {
	return cdk_ffi.FfiToken{
		TokenString: t.TokenString,
		Mint:        t.Mint,
		Memo:        t.Memo,
		Unit:        t.Unit,
	}
}

// SendKind wrapper types
type SendKind interface{}

type SendKindOnlineExact struct{}

type SendKindOnlineTolerance struct{ Tolerance uint64 }

type SendKindOfflineExact struct{}

type SendKindOfflineTolerance struct{ Tolerance uint64 }

func SendKindToFFI(k SendKind) cdk_ffi.FfiSendKind {
	switch v := k.(type) {
	case SendKindOnlineExact:
		return cdk_ffi.FfiSendKindOnlineExact{}
	case SendKindOnlineTolerance:
		return cdk_ffi.FfiSendKindOnlineTolerance{Tolerance: cdk_ffi.FfiAmount{Value: v.Tolerance}}
	case SendKindOfflineExact:
		return cdk_ffi.FfiSendKindOfflineExact{}
	case SendKindOfflineTolerance:
		return cdk_ffi.FfiSendKindOfflineTolerance{Tolerance: cdk_ffi.FfiAmount{Value: v.Tolerance}}
	default:
		// default to online exact
		return cdk_ffi.FfiSendKindOnlineExact{}
	}
}

func SendKindFromFFI(f cdk_ffi.FfiSendKind) SendKind {
	switch v := f.(type) {
	case cdk_ffi.FfiSendKindOnlineExact:
		return SendKindOnlineExact{}
	case cdk_ffi.FfiSendKindOnlineTolerance:
		return SendKindOnlineTolerance{Tolerance: v.Tolerance.Value}
	case cdk_ffi.FfiSendKindOfflineExact:
		return SendKindOfflineExact{}
	case cdk_ffi.FfiSendKindOfflineTolerance:
		return SendKindOfflineTolerance{Tolerance: v.Tolerance.Value}
	default:
		return nil
	}
}

// SendOptions is a Go-native representation
type SendOptions struct {
	Memo              *SendMemo
	AmountSplitTarget uint
	Kind              SendKind
	IncludeFee        bool
	Metadata          map[string]string
	MaxProofs         *uint64
}

func (o SendOptions) ToFFI() cdk_ffi.FfiSendOptions {
	var ffiMemo *cdk_ffi.FfiSendMemo
	if o.Memo != nil {
		ffiMemo = o.Memo.ToFFI()
	}
	ffiKind := SendKindToFFI(o.Kind)

	return cdk_ffi.FfiSendOptions{
		Memo:              ffiMemo,
		AmountSplitTarget: cdk_ffi.FfiSplitTarget(o.AmountSplitTarget),
		SendKind:          ffiKind,
		IncludeFee:        o.IncludeFee,
		Metadata:          o.Metadata,
		MaxProofs:         o.MaxProofs,
	}
}

func SendOptionsFromFFI(f cdk_ffi.FfiSendOptions) SendOptions {
	return SendOptions{
		Memo:              SendMemoFromFFI(f.Memo),
		AmountSplitTarget: uint(f.AmountSplitTarget),
		Kind:              SendKindFromFFI(f.SendKind),
		IncludeFee:        f.IncludeFee,
		Metadata:          f.Metadata,
		MaxProofs:         f.MaxProofs,
	}
}
