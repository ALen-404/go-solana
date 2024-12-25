package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/blocto/solana-go-sdk/client"
	"github.com/joho/godotenv"
	"golang.org/x/time/rate"
)

// 交易数据结构
type TransactionData struct {
	Date       string  // 交易日期（区块链时间戳的格式化时间）
	Timestamp  int64   // 交易日期的 Unix 时间戳
	Type       string  // 交易类型: Buy or Sell
	GOAT       float64 // GOAT 数量
	SOL        float64 // SOL 数量
	Txn        string  // 交易签名
}

// 加载 .env 文件
func loadEnv() error {
	err := godotenv.Load()
	if err != nil {
		return fmt.Errorf("Error loading .env file: %v", err)
	}
	return nil
}

// 初始化 Solana 客户端和请求速率限制器
func initializeClient(rpcURL string) (*client.Client, *rate.Limiter) {
	c := client.NewClient(rpcURL)
	limiter := rate.NewLimiter(rate.Every(time.Second/30), 30)
	return c, limiter
}

// 获取交易签名
func fetchSignatures(c *client.Client, tokenMintAddress string, lastSignature string, limit int) ([]string, error) {
	config := client.GetSignaturesForAddressConfig{
		Limit:  limit,
		Before: lastSignature,
	}
	signatures, err := c.GetSignaturesForAddressWithConfig(context.Background(), tokenMintAddress, config)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch signatures: %v", err)
	}

	// Extract the signatures from the response
	var signatureList []string
	for _, sig := range signatures {
		signatureList = append(signatureList, sig.Signature)
	}

	return signatureList, nil
}

// 处理单个交易
func processTransaction(tx *client.Transaction, sig string, solMintAddress, goatMintAddress, exchangeRouter string) (TransactionData, error) {
	var txnType string
	var goatChange, solChange float64

	// 遍历 PreTokenBalances 和 PostTokenBalances
	for _, preBalance := range tx.Meta.PreTokenBalances {
		if preBalance.Owner == exchangeRouter {
			for _, postBalance := range tx.Meta.PostTokenBalances {
				if postBalance.Owner == exchangeRouter && preBalance.Mint == postBalance.Mint {
					// 处理 SOL 余额变化
					if postBalance.Mint == solMintAddress {
						preSOL, _ := strconv.ParseFloat(preBalance.UITokenAmount.UIAmountString, 64)
						postSOL, _ := strconv.ParseFloat(postBalance.UITokenAmount.UIAmountString, 64)
						if postSOL > preSOL {
							txnType = "Buy"
							solChange = postSOL - preSOL
						} else if postSOL < preSOL {
							txnType = "Sell"
							solChange = preSOL - postSOL
						}
					}

					// 处理 GOAT 余额变化
					if postBalance.Mint == goatMintAddress {
						preGOAT, _ := strconv.ParseFloat(preBalance.UITokenAmount.UIAmountString, 64)
						postGOAT, _ := strconv.ParseFloat(postBalance.UITokenAmount.UIAmountString, 64)
						goatChange = postGOAT - preGOAT
					}
				}
			}
		}
	}

	// 如果找到交易数据
	if solChange != 0 || goatChange != 0 {
		blockTimeUnix := *tx.BlockTime
		blockTime := time.Unix(blockTimeUnix, 0).Format("2006-01-02 15:04:05")
		return TransactionData{
			Date:      blockTime,
			Timestamp: blockTimeUnix,
			Type:      txnType,
			GOAT:      roundTo6Decimal(goatChange),
			SOL:       roundTo9Decimal(solChange),
			Txn:       sig,
		}, nil
	}

	return TransactionData{}, fmt.Errorf("No relevant transaction data found")
}

// 将交易数据写入 CSV 文件
func writeTransactionsToCSV(transactions []TransactionData, appendMode bool) error {
	var file *os.File
	var err error

	if appendMode {
		file, err = os.OpenFile("transactions.csv", os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	} else {
		file, err = os.Create("transactions.csv")
	}

	if err != nil {
		return fmt.Errorf("Unable to open file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入表头（仅在非追加模式时）
	if !appendMode {
		writer.Write([]string{"Date", "Timestamp", "Type", "GOAT", "SOL", "Txn"})
	}

	for _, txn := range transactions {
		record := []string{
			txn.Date,
			strconv.FormatInt(txn.Timestamp, 10), // 时间戳
			txn.Type,
			strconv.FormatFloat(txn.GOAT, 'f', 6, 64),
			strconv.FormatFloat(txn.SOL, 'f', 9, 64),
			txn.Txn,
		}
		writer.Write(record)
	}

	return nil
}

// 处理交易的 rounds
func roundTo6Decimal(value float64) float64 {
	return math.Abs(math.Trunc(value*1e6) / 1e6)
}

func roundTo9Decimal(value float64) float64 {
	return math.Abs(math.Trunc(value*1e9) / 1e9)
}

func main() {
	// 加载 .env 文件
	err := loadEnv()
	if err != nil {
		log.Fatalf(err.Error())
	}

	// 读取配置
	rpcURL := os.Getenv("RPC_URL")
	tokenMintAddress := os.Getenv("TOKEN_MINT_ADDRESS")
	solMintAddress := os.Getenv("SOL_MINT_ADDRESS")
	goatMintAddress := os.Getenv("GOAT_MINT_ADDRESS")
	exchangeRouter := os.Getenv("EXCHANGE_ROUTER")
	maxTransactionsStr := os.Getenv("MAX_TRANSACTIONS")

	if rpcURL == "" || tokenMintAddress == "" || solMintAddress == "" || goatMintAddress == "" || exchangeRouter == "" || maxTransactionsStr == "" {
		log.Fatal("Missing required environment variables")
	}

	// 解析 MAX_TRANSACTIONS 环境变量
	maxTransactions, err := strconv.Atoi(maxTransactionsStr)
	if err != nil || maxTransactions <= 0 {
		log.Fatalf("Invalid MAX_TRANSACTIONS value: %v", maxTransactionsStr)
	}

	// 初始化客户端和速率限制器
	c, limiter := initializeClient(rpcURL)

	var transactions []TransactionData
	var lastSignature string
	appendMode := false

	for len(transactions) < maxTransactions {
		// 分批查询交易签名
		log.Printf("Fetching transactions... Last Signature: %s", lastSignature)
		signatures, err := fetchSignatures(c, tokenMintAddress, lastSignature, 1000)
		if err != nil {
			log.Fatalf(err.Error())
		}

		if len(signatures) == 0 {
			log.Println("No more signatures available, stopping.")
			break // 无更多数据，退出
		}

		log.Printf("Fetched %d signatures", len(signatures))
		batchTransactions := []TransactionData{}

		for _, sig := range signatures {
			log.Printf("Processing transaction: %s", sig)

			if err := limiter.Wait(context.TODO()); err != nil {
				log.Printf("Rate limiter error: %v", err)
				continue
			}

			tx, err := c.GetTransaction(context.TODO(), sig)
			if err != nil {
				log.Printf("Failed to fetch transaction for signature %s: %v", sig, err)
				continue
			}

			// 检查交易是否成功
			if tx.Meta.Err != nil {
				log.Printf("Transaction %s failed with error: %v", sig, tx.Meta.Err)
				continue
			}

			// 处理交易
			txnData, err := processTransaction(tx, sig, solMintAddress, goatMintAddress, exchangeRouter)
			if err != nil {
				log.Printf("Error processing transaction %s: %v", sig, err)
				continue
			}

			// 将交易数据添加到批次中
			batchTransactions = append(batchTransactions, txnData)

			// 达到限制时退出循环
			if len(transactions)+len(batchTransactions) >= maxTransactions {
				break
			}
		}

		// 将当前批次写入 CSV
		err = writeTransactionsToCSV(batchTransactions, appendMode)
		if err != nil {
			log.Fatalf("Error writing to CSV: %v", err)
		}

		appendMode = true
		transactions = append(transactions, batchTransactions...)

		// 更新 lastSignature 为当前批次的最后一个签名
		lastSignature = signatures[len(signatures)-1]
	}

	fmt.Println("CSV file created successfully!")
}
