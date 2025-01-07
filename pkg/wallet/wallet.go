package wallet

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/blocto/solana-go-sdk/client"
	"github.com/blocto/solana-go-sdk/common"
	"github.com/blocto/solana-go-sdk/program/associated_token_account"
	"github.com/blocto/solana-go-sdk/program/system"
	"github.com/blocto/solana-go-sdk/program/token"
	"github.com/blocto/solana-go-sdk/rpc"
	"github.com/blocto/solana-go-sdk/types"
	"github.com/ALen-404/go-solana/pkg/config"
)

const SOL_MINT_ADDR = "So11111111111111111111111111111111111111112"
const GOAT_MINT_ADDR = "CzLSujWBLFsSjncfkh59rUFqvafWcY5tzedWJSuypump"

// WalletManager 管理 Solana 钱包的结构体
type WalletManager struct {
	Client  *client.Client
	Network string // "mainnet", "testnet", "devnet", "localhost"
	Account types.Account
	// TokenCache map[string]common.PublicKey // 缓存代币地址
}

// 添加 Jupiter API 相关常量
const (
	JupiterQuoteAPI    = "https://quote-api.jup.ag/v6/quote"
	JupiterSwapAPI     = "https://quote-api.jup.ag/v6/swap"
	defaultSlippageBps = 100 // 1% slippage
)

// QuoteResponse Jupiter 报价响应
type QuoteResponse struct {
	InputMint            string      `json:"inputMint"`
	InAmount             string      `json:"inAmount"`
	OutputMint           string      `json:"outputMint"`
	OutAmount            string      `json:"outAmount"`
	OtherAmountThreshold string      `json:"otherAmountThreshold"`
	SwapMode             string      `json:"swapMode"`
	SlippageBps          int         `json:"slippageBps"`
	PlatformFee          interface{} `json:"platformFee"`
	PriceImpactPct       string      `json:"priceImpactPct"`
	RoutePlan            []RoutePlan `json:"routePlan"`
	ScoreReport          interface{} `json:"scoreReport"`
	ContextSlot          int         `json:"contextSlot"`
	TimeTaken            float64     `json:"timeTaken"`
}

type RoutePlan struct {
	SwapInfo SwapInfo `json:"swapInfo"`
	Percent  int      `json:"percent"`
}

type SwapInfo struct {
	AmmKey     string `json:"ammKey"`
	Label      string `json:"label"`
	InputMint  string `json:"inputMint"`
	OutputMint string `json:"outputMint"`
	InAmount   string `json:"inAmount"`
	OutAmount  string `json:"outAmount"`
	FeeAmount  string `json:"feeAmount"`
	FeeMint    string `json:"feeMint"`
}

// NewWalletManager 创建新的钱包管理器
func NewWalletManager(network string) (*WalletManager, error) {
	endpoint, ok := config.NetworkConfig[network]
	if !ok {
		return nil, fmt.Errorf("unsupported network: %s", network)
	}

	return &WalletManager{
		Client:  client.NewClient(endpoint),
		Network: network,
		// tokenCache: make(map[string]common.PublicKey),
	}, nil
}

// CreateAccount 创建新账户
func (wm *WalletManager) CreateAccount() (string, error) {
	account := types.NewAccount()
	wm.Account = account

	// Create a JSON file to save the private key
	// Truncate the public key for the filename
	truncatedPublicKey := account.PublicKey.ToBase58()[:10] // Use first 10 characters
	filename := fmt.Sprintf("assets/wallet_%s.json", truncatedPublicKey)
	file, err := os.Create(filename)
	if err != nil {
		return "", fmt.Errorf("unable to create file: %v", err)
	}
	defer file.Close()

	// Write private key to JSON file
	if err := json.NewEncoder(file).Encode(account.PrivateKey); err != nil {
		return "", fmt.Errorf("unable to write to file: %v", err)
	}

	fmt.Printf("New account created with Public Key: %s, Private Key saved to %s\n", account.PublicKey.ToBase58(), filename)

	return account.PublicKey.ToBase58(), nil
}

// Mint需要和集群匹配，否则会出现“incorrect program id”的错误
func (wm *WalletManager) CreateTokenAccount(ctx context.Context, mintAddr string) (string, error) {
	mintPubkey := common.PublicKeyFromString(mintAddr)
	ata, _, err := common.FindAssociatedTokenAddress(wm.Account.PublicKey, mintPubkey)
	if err != nil {
		return "", fmt.Errorf("find ata error, err: %v", err)
	}
	fmt.Println("Associated Token Address, ata:", ata.ToBase58())

	recentBlockhashResponse, err := wm.Client.GetLatestBlockhash(ctx)
	if err != nil {
		return "", fmt.Errorf("get recent block hash error, err: %w", err)
	}

	createTokenAccountInstruction := associated_token_account.Create(associated_token_account.CreateParam{
		Funder:                 wm.Account.PublicKey,
		Owner:                  wm.Account.PublicKey,
		Mint:                   mintPubkey,
		AssociatedTokenAccount: ata,
	})

	message := types.NewMessage(types.NewMessageParam{
		FeePayer:        wm.Account.PublicKey,
		RecentBlockhash: recentBlockhashResponse.Blockhash,
		Instructions: []types.Instruction{
			createTokenAccountInstruction,
		},
	})
	fmt.Println("message:", message)
	tx, err := types.NewTransaction(types.NewTransactionParam{
		Signers: []types.Account{wm.Account},
		Message: message,
	})
	if err != nil {
		return "", fmt.Errorf("generate tx error, err: %v", err)
	}
	fmt.Println("tx:", tx)
	txhash, err := wm.Client.SendTransaction(ctx, tx)
	if err != nil {
		return "", fmt.Errorf("send raw tx error, err: %v", err)
	}

	fmt.Println("txhash:", txhash)
	return ata.ToBase58(), nil
}

// RequestAirdrop 请求空投
func (wm *WalletManager) RequestAirdrop(publicKey string, amount uint64) bool {
	_, err := wm.Client.RequestAirdrop(
		context.Background(),
		publicKey,
		amount,
	)
	if err != nil {
		fmt.Printf("Failed to request airdrop: %v\n", err)
		return false
	}
	fmt.Println("Airdrop successful!")
	return true
}

// LoadAccount 从私钥文件加载账户
func (wm *WalletManager) LoadAccount(keyPath string) (string, error) {
	data, err := os.ReadFile(keyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read key file: %w", err)
	}

	var privateKey []byte
	err = json.Unmarshal(data, &privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal private key: %w", err)
	}
	account, err := types.AccountFromBytes(privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to create account from bytes: %w", err)
	}

	wm.Account = account
	return account.PublicKey.ToBase58(), nil
}

// CheckAmount 检查指定代币的余额
func (wm *WalletManager) CheckAmount(ctx context.Context, mintAddr string) (uint64, error) {
	if wm.Account.PublicKey.ToBase58() == "" {
		return 0, errors.New("no account loaded")
	}

	// mintAddr为SOL_MINT_ADDR，则直接获取SOL余额
	if mintAddr == SOL_MINT_ADDR {
		balance, err := wm.Client.GetBalance(
			ctx,
			wm.Account.PublicKey.ToBase58(),
		)
		if err != nil {
			return 0, fmt.Errorf("failed to get SOL balance: %w", err)
		}
		return balance, nil
	}

	mintPubkey := common.PublicKeyFromString(mintAddr)
	// 获取关联代币的账户地址
	ata, _, err := common.FindAssociatedTokenAddress(wm.Account.PublicKey, mintPubkey)
	if err != nil {
		return 0, fmt.Errorf("failed to find associated token address: %w", err)
	}
	// 获取SPL代币余额
	tokenAmount, err := wm.Client.GetTokenAccountBalance(
		ctx,
		ata.ToBase58(),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to get token balance: %w", err)
	}
	balance := tokenAmount.Amount
	return balance, nil
}

// Transfer 转账功能
func (wm *WalletManager) TransferSOL(ctx context.Context, toAddress string, amount uint64) (string, error) {
	senderPubKey := wm.Account.PublicKey
	receiverPubKey := common.PublicKeyFromString(toAddress)

	// 检查余额是否足够
	balance, err := wm.CheckAmount(ctx, SOL_MINT_ADDR)
	if err != nil {
		return "", fmt.Errorf("failed to check balance: %v", err)
	}
	if balance < amount {
		return "", fmt.Errorf("insufficient balance: have %d lamports, need %d lamports", balance, amount)
	}

	// 获取最近的区块哈希（用于构建交易）
	recentBlockhashResponse, err := wm.Client.GetLatestBlockhash(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get latest blockhash: %v", err)
	}

	transferInstruction := system.Transfer(system.TransferParam{
		From:   senderPubKey,   // 发送账户的公钥
		To:     receiverPubKey, // 接收账户的公钥
		Amount: amount,
	})

	message := types.NewMessage(types.NewMessageParam{
		FeePayer:        senderPubKey,
		RecentBlockhash: recentBlockhashResponse.Blockhash,
		Instructions: []types.Instruction{
			transferInstruction,
		},
	})

	// create a transfer tx
	tx, err := types.NewTransaction(types.NewTransactionParam{
		Signers: []types.Account{wm.Account},
		Message: message,
	})
	if err != nil {
		log.Fatalf("failed to new a transaction, err: %v", err)
	}

	// 发送交易
	txhash, err := wm.Client.SendTransaction(ctx, tx)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction : %v", err)
	}
	log.Println("txhash:", txhash)
	return txhash, nil
}

// // Buy 市价买入代币
// func (wm *WalletManager) Buy(ctx context.Context, mintAddr string, amountInSOL float64) error {
// 	// 构建报价请求
// 	quoteURL := fmt.Sprintf("%s?inputMint=%s&outputMint=%s&amount=%d&slippageBps=%d",
// 		JupiterQuoteAPI,
// 		SOL_MINT_ADDR, // SOL 的代币地址
// 		mintAddr,
// 		uint64(amountInSOL*1e9), // 转换为 lamports
// 		defaultSlippageBps,
// 	)

// 	// 获取报价
// 	quote, err := wm.getQuote(quoteURL)
// 	if err != nil {
// 		return fmt.Errorf("failed to get quote: %w", err)
// 	}

// 	// 执行交换
// 	err = wm.executeSwap(ctx, quote)
// 	if err != nil {
// 		return fmt.Errorf("swap failed: %w", err)
// 	}

// 	fmt.Printf("Successfully bought %s tokens\n", mintAddr)
// 	return nil
// }

// Sell 市价卖出代币
func (wm *WalletManager) Sell(ctx context.Context, mintAddr string, amount float64) error {
	// 检查代币余额
	balance, err := wm.CheckAmount(ctx, mintAddr)
	if err != nil {
		return fmt.Errorf("failed to check balance: %w", err)
	}

	if float64(balance) < amount {
		return fmt.Errorf("insufficient balance: have %v, need %v", balance, amount)
	}

	// 构建报价请求
	quoteURL := fmt.Sprintf("%s?inputMint=%s&outputMint=%s&amount=%d&slippageBps=%d",
		JupiterQuoteAPI,
		mintAddr,
		SOL_MINT_ADDR, // SOL
		uint64(amount),
		defaultSlippageBps,
	)

	// 获取报价
	quote, err := wm.GetQuote(quoteURL)
	if err != nil {
		return fmt.Errorf("failed to get quote: %w", err)
	}
	fmt.Println("Quote:", quote)
	// 执行交换
	// err = wm.executeSwap(ctx, quote)
	// if err != nil {
	// 	return fmt.Errorf("swap failed: %w", err)
	// }

	// fmt.Printf("Successfully sold %s tokens\n", mintAddr)
	return nil
}

// 内部辅助方法

// getQuote 从 Jupiter 获取报价
func (wm *WalletManager) GetQuote(quoteURL string) (*QuoteResponse, error) {
	fmt.Println("Fetching quote from URL:", quoteURL) // 输出请求的URL

	resp, err := http.Get(quoteURL)
	if err != nil {
		fmt.Println("Error fetching quote:", err) // 输出错误信息
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return nil, err
	}
	fmt.Println("Received response body:", string(body)) // 输出响应体

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Println("Received non-OK status code:", resp.StatusCode, "with body:", string(body)) // 输出状态码和响应体
		return nil, fmt.Errorf("quote failed: %s", string(body))
	}

	var quote QuoteResponse
	if err := json.Unmarshal(body, &quote); err != nil {
		fmt.Println("Error decoding JSON:", err)
		return nil, err
	}

	fmt.Println("Successfully fetched quote:", quote)
	return &quote, nil
}

// executeSwap 执行代币交换
func (wm *WalletManager) executeSwap(ctx context.Context, quote *QuoteResponse) error {
	// 构建交换请求
	swapReq := struct {
		UserPublicKey string        `json:"userPublicKey"`
		QuoteResp     QuoteResponse `json:"quoteResponse"`
	}{
		UserPublicKey: wm.Account.PublicKey.ToBase58(),
		QuoteResp:     *quote,
	}

	// 发送交换请求
	reqBody, err := json.Marshal(swapReq)
	if err != nil {
		return err
	}
	fmt.Println("Sending swap request:", string(reqBody))
	resp, err := http.Post(JupiterSwapAPI, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("swap request failed: %s", string(body))
	}

	// 处理响应，获取交易指令
	var swapResp struct {
		SwapTransaction []byte `json:"swapTransaction"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&swapResp); err != nil {
		return err
	}

	// 使用 Transaction 对象进行初始化，假设您有 swapTransaction 数据
	// tx, err := types.NewTransaction(swapResp.SwapTransaction)

	// // 签名交易
	// tx.Sign([]types.Account{wm.Account})

	// // 发送交易
	// sig, err := wm.Client.SendTransaction(ctx, tx)
	// if err != nil {
	// 	return err
	// }

	return nil
}

func CreateMint(feePayer types.Account, mintAuthority types.Account) {
	c := client.NewClient(rpc.DevnetRPCEndpoint)

	// create an mint account
	mint := types.NewAccount()
	fmt.Println("mint:", mint.PublicKey.ToBase58())

	// get rent
	rentExemptionBalance, err := c.GetMinimumBalanceForRentExemption(
		context.Background(),
		token.MintAccountSize,
	)
	if err != nil {
		log.Fatalf("get min balacne for rent exemption, err: %v", err)
	}

	res, err := c.GetLatestBlockhash(context.Background())
	if err != nil {
		log.Fatalf("get recent block hash error, err: %v\n", err)
	}

	tx, err := types.NewTransaction(types.NewTransactionParam{
		Message: types.NewMessage(types.NewMessageParam{
			FeePayer:        feePayer.PublicKey,
			RecentBlockhash: res.Blockhash,
			Instructions: []types.Instruction{
				system.CreateAccount(system.CreateAccountParam{
					From:     feePayer.PublicKey,
					New:      mint.PublicKey,
					Owner:    common.TokenProgramID,
					Lamports: rentExemptionBalance,
					Space:    token.MintAccountSize,
				}),
				token.InitializeMint(token.InitializeMintParam{
					Decimals:   8,
					Mint:       mint.PublicKey,
					MintAuth:   mintAuthority.PublicKey,
					FreezeAuth: nil,
				}),
			},
		}),
		Signers: []types.Account{feePayer, mint},
	})
	if err != nil {
		log.Fatalf("generate tx error, err: %v\n", err)
	}

	txhash, err := c.SendTransaction(context.Background(), tx)
	if err != nil {
		log.Fatalf("send tx error, err: %v\n", err)
	}

	log.Println("txhash:", txhash)
}
