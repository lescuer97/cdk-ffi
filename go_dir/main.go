package main

import (
	"go_dir/cdk_ffi"
	"log"
)

// example on how to run:
// CGO_ENABLED="1" CGO_LDFLAGS="-L/home/leo/Programar/trabajo/open_source/cdk-ffi/target/release -lcdk_ffi" LD_LIBRARY_PATH="/home/leo/Programar/trabajo/open_source/cdk-ffi/target/release" go run ./...

func main() {
	log.Println("TEST")
	nmonic, err := cdk_ffi.GenerateMnemonic()
	if err != nil {
		log.Panicf("could not create seed phrase. %+v", err)

	}
	storage, err := NewStorage()
	if err != nil {
		log.Panicf("could not create storage. %+v", err)
	}

	wallet, err := NewWalletFromMnemonic("http://localhost:8081", Sat, storage, nmonic)
	if err != nil {
		log.Panicf("could not generate wallet. %+v", err)
	}

	balance, err := wallet.Balance()
	if err != nil {
		log.Panicf("could not get balance. %+v", err)
	}
	log.Printf("\n  balance before minting %+v", balance)

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

	log.Printf("\n Balance after minting. %+v", balance)
	sendAmount := Amount {
		Value: 10,
	}
	memo := SendMemo{
		Memo: "test memo",
		IncludeMemo: true,
	}
	sendOptions := SendOptions{
		AmountSplitTarget: SplitTargetDefault,
		Memo: &memo,
		Kind: SendKindOnlineExact{},
		IncludeFee: true,
		Metadata: nil,
		MaxProofs: nil,
	}
	token, err := wallet.Send(sendAmount,sendOptions )
	if err != nil {
		log.Panicf("could not get balance. %+v", err)
	}
	log.Printf("\n token made for sending %+v", token.String())

	balance, err = wallet.Balance()
	if err != nil {
		log.Panicf("could not get balance. %+v", err)
	}

	log.Printf("\n Balance after sending token. %+v", balance)


	meltQuote, err := wallet.MeltQuote("lnbc100n1p5tmvnlpp5luw5fra3zgpnugrh0vuss9hzy9m6xr5uf3mnw6n2xlcv06srqmhqdqqcqzzsxqyz5vqrzjqvueefmrckfdwyyu39m0lf24sqzcr9vcrmxrvgfn6empxz7phrjxvrttncqq0lcqqyqqqqlgqqqqqqgq2qsp5rsr6jf4ukg8h7u96hfjxspukxswyam90q5pqc0pssnlw403hq8us9qxpqysgq4jnaqd35ly4jtw243533wcae6kk9dsue9sxz0uu042exg4u7m4hn2vkq94m4u8j9ph93fplv7v7q22h994qw6pruy3ywcg9jltcfzhgprwe8d2")
	if err != nil {
		log.Panicf("could not get melt quote. %+v", err)
	}
	meltResult, err := wallet.Melt(meltQuote.Id)
	if err != nil {
		log.Panicf("could not melt . %+v", err)
	}
	log.Printf("\n result from melt: %+v", meltResult)

	balance, err = wallet.Balance()
	if err != nil {
		log.Panicf("could not get balance. %+v", err)
	}

	log.Printf("\n balance after melting: %+v", balance)
	// wallet.wallet.
	// balance.Destroy
}
