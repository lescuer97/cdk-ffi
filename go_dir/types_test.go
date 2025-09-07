package main

import (
	"testing"
	"go_dir/cdk_ffi"
)

func TestSendMemoConversion(t *testing.T) {
	m := &SendMemo{Memo: "hello", IncludeMemo: true}
	ffi := m.ToFFI()
	got := SendMemoFromFFI(ffi)
	if got == nil || got.Memo != "hello" || got.IncludeMemo != true {
		t.Fatalf("unexpected conversion: %#v", got)
	}
}

func TestTokenConversion(t *testing.T) {
	f := cdk_ffi.FfiToken{TokenString: "tok", Mint: "mint1", Memo: nil, Unit: "sat"}
	got := TokenFromFFI(f)
	if got.TokenString != "tok" || got.Mint != "mint1" || got.Unit != "sat" {
		t.Fatalf("unexpected token conversion: %#v", got)
	}
}

func TestSendOptionsRoundTrip(t *testing.T) {
	max := uint64(10)
	o := SendOptions{
		Memo:              &SendMemo{Memo: "memo", IncludeMemo: true},
		AmountSplitTarget: 1,
		Kind:              SendKindOnlineTolerance{Tolerance: 123},
		IncludeFee:        true,
		Metadata:          map[string]string{"k": "v"},
		MaxProofs:         &max,
	}
	ffi := o.ToFFI()
	back := SendOptionsFromFFI(ffi)
	if back.AmountSplitTarget != o.AmountSplitTarget || back.IncludeFee != o.IncludeFee || back.Metadata["k"] != "v" {
		t.Fatalf("roundtrip mismatch: %#v", back)
	}
	if back.Kind == nil {
		t.Fatalf("kind lost in roundtrip")
	}
}
