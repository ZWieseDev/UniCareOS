package wallet

type Wallet struct {
    PrivateKey string
    // Add other fields as needed (e.g., PublicKey, Metadata)
}

type WalletLoader interface {
    LoadWallet() (*Wallet, error)
}
