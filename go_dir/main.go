package main

import (
	"go_dir/cdk_ffi"
	"log"
)

func main() {
	log.Println("TEST")
	nmonic, err := cdk_ffi.GenerateMnemonic()
	if err != nil {
		log.Panicf("could not create seed phrase. %+v", err)

	}
	log.Println("nmonic: ", nmonic)
	// log.Println("nmonic: ", nmonic)
	// storage, err := NewStorageFromPath(".")
	storage, err := NewStorage()
	if err != nil {
		log.Panicf("could not create storage. %+v", err)
	}
	log.Println("storage: ", storage)


	wallet, err := NewWalletFromMnemonic("http://localhost:8081", Sat, storage, nmonic)
	if err != nil {
		log.Panicf("could not generate wallet. %+v", err)
	}

	balance, err := wallet.Balance()
	if err != nil {
		log.Panicf("could not get balance. %+v", err)
	}
	log.Printf("\n current balance. %+v", balance)

	log.Println("trying to get mint quote")
	mintquote, err := wallet.MintQuote(Amount{Value: 100}, nil)
	if err != nil {
		log.Panicf("wallet.MintQuote(Amount{Value: 100}, nil). %+v", err)
	}

	log.Println("Minting...")
	amount, err := wallet.Mint(mintquote.Id, SplitTargetDefault)
	if err != nil {
		log.Panicf("wallet.Mint(mintquote.Id, SplitTargetDefault). %+v", err)
	}
	log.Printf("minted amount: %+v", amount)

	balance, err = wallet.Balance()
	if err != nil {
		log.Panicf("could not get balance. %+v", err)
	}

	log.Printf("\n current balance. %+v", balance)

	// wallet.wallet.
	// balance.Destroy
}
