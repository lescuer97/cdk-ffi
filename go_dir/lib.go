package main

import "go_dir/cdk_ffi"

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

type Unit cdk_ffi.FfiCurrencyUnit

const Sat Unit = cdk_ffi.FfiCurrencyUnitSat

func RestoreFroMMnemonic(minturl string, unit Unit, storage Storage, mnemonic string) (*Wallet, error) {
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
