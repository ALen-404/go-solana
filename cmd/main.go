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

// 处理交易数据并写入 CSV 文件
func writeTransactionsToCSV(transactions []TransactionData, appendMode bool) {
	var file *os.File
	var err error

	if appendMode {
		file, err = os.OpenFile("transactions.csv", os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	} else {
		file, err = os.Create("transactions.csv")
	}

	if err != nil {
		log.Fatalf("Unable to open file: %v", err)
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
}

func main() {
	// 加载 .env 文件
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

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

	c := client.NewClient(rpcURL)

	limiter := rate.NewLimiter(rate.Every(1*time.Second), 5)
	var transactions []TransactionData
	var lastSignature string
	appendMode := false

	for len(transactions) < maxTransactions {
		// 分批查询交易签名
		config := client.GetSignaturesForAddressConfig{
			Limit:  1000,
			Before: lastSignature,
		}

		log.Printf("Fetching transactions... Last Signature: %s", lastSignature)

		signatures, err := c.GetSignaturesForAddressWithConfig(context.Background(), tokenMintAddress, config)
		if err != nil {
			log.Fatalf("Failed to fetch signatures: %v", err)
		}

		if len(signatures) == 0 {
			log.Println("No more signatures available, stopping.")
			break // 无更多数据，退出
		}

		log.Printf("Fetched %d signatures", len(signatures))
		batchTransactions := []TransactionData{}

		for _, sig := range signatures {
			log.Printf("Processing transaction: %s", sig.Signature)

			if err := limiter.Wait(context.TODO()); err != nil {
				log.Printf("Rate limiter error: %v", err)
				continue
			}

			tx, err := c.GetTransaction(context.TODO(), sig.Signature)
			if err != nil {
				log.Printf("Failed to fetch transaction for signature %s: %v", sig.Signature, err)
				continue
			}

			// 检查交易是否成功
			if tx.Meta.Err != nil {
				log.Printf("Transaction %s failed with error: %v", sig.Signature, tx.Meta.Err)
				continue
			}

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
				batchTransactions = append(batchTransactions, TransactionData{
					Date:      blockTime,
					Timestamp: blockTimeUnix,
					Type:      txnType,
					GOAT:      roundTo6Decimal(goatChange),
					SOL:       roundTo9Decimal(solChange),
					Txn:       sig.Signature,
				})
				log.Printf("Transaction added: %+v", batchTransactions[len(batchTransactions)-1])
			}

			// 达到限制时退出循环
			if len(transactions)+len(batchTransactions) >= maxTransactions {
				break
			}
		}

		// 将当前批次写入 CSV
		writeTransactionsToCSV(batchTransactions, appendMode)
		appendMode = true
		transactions = append(transactions, batchTransactions...)

		// 更新 lastSignature 为当前批次的最后一个签名
		lastSignature = signatures[len(signatures)-1].Signature
	}

	fmt.Println("CSV file created successfully!")
}

func roundTo6Decimal(value float64) float64 {
	return math.Abs(math.Trunc(value*1e6) / 1e6)
}

func roundTo9Decimal(value float64) float64 {
	return math.Abs(math.Trunc(value*1e9) / 1e9)
}
