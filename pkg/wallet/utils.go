package wallet

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/blocto/solana-go-sdk/types"
)

func GenerateWallets(numWallets int) error {
	for i := 0; i < numWallets; i++ {
		wallet := types.NewAccount() // Generate new wallet
		privateKey := wallet.PrivateKey
		publicKey := wallet.PublicKey.ToBase58()

		// Create a JSON file for each private key using the public key in the filename
		// Truncate the public key for the filename
		truncatedPublicKey := publicKey[:10] // Use first 10 characters
		filename := fmt.Sprintf("assets/wallet_%s.json", truncatedPublicKey)
		file, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("unable to create file: %v", err)
		}
		defer file.Close()

		// Write private key to JSON file
		if err := json.NewEncoder(file).Encode(privateKey); err != nil {
			return fmt.Errorf("unable to write to file: %v", err)
		}

		fmt.Printf("Wallet with Public Key: %s, Private Key saved to %s\n", publicKey, filename)
	}

	return nil
}
