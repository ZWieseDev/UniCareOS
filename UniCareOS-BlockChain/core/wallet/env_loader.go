package wallet

import (
    "os"
    "errors"
)

type EnvWalletLoader struct{}

func (l *EnvWalletLoader) LoadWallet() (*Wallet, error) {
    privKey := os.Getenv("UNICAREOS_SIGNER_PRIVKEY")
    if privKey == "" {
        return nil, errors.New("UNICAREOS_SIGNER_PRIVKEY not set in environment")
    }
    return &Wallet{PrivateKey: privKey}, nil
}
