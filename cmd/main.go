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
	Date string  // 交易日期（区块链时间戳）
	Type string  // 交易类型: Buy or Sell
	GOAT float64 // GOAT 数量
	SOL  float64 // SOL 数量
	Txn  string  // 交易签名
}

// 处理交易数据并写入 CSV 文件
func writeTransactionToCSV(transactions []TransactionData) {
	file, err := os.Create("transactions.csv")
	if err != nil {
		log.Fatalf("Unable to create file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Write([]string{"Date", "Type", "GOAT", "SOL", "Txn"})

	for _, txn := range transactions {
		record := []string{
			txn.Date,
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
	maxTransactionsStr := os.Getenv("MAX_TRANSACTIONS")

	if rpcURL == "" || tokenMintAddress == "" || solMintAddress == "" || goatMintAddress == "" || maxTransactionsStr == "" {
		log.Fatal("Missing required environment variables")
	}

	// 解析 MAX_TRANSACTIONS 环境变量
	maxTransactions, err := strconv.Atoi(maxTransactionsStr)
	if err != nil || maxTransactions <= 0 {
		log.Fatalf("Invalid MAX_TRANSACTIONS value: %v", maxTransactionsStr)
	}

	c := client.NewClient(rpcURL)

	signatures, err := c.GetSignaturesForAddress(context.TODO(), tokenMintAddress)
	if err != nil {
		log.Fatalf("Failed to fetch signatures: %v", err)
	}

	fmt.Printf("Found %d transactions for the token\n", len(signatures))

	limiter := rate.NewLimiter(rate.Every(1*time.Second), 5)
	var transactions []TransactionData

	// 仅处理限制条数的交易
	for i := 0; i < len(signatures) && i < maxTransactions; i++ {
		if err := limiter.Wait(context.TODO()); err != nil {
			log.Printf("Rate limiter error: %v", err)
			continue
		}

		sig := signatures[i]
		tx, err := c.GetTransaction(context.TODO(), sig.Signature)
		if err != nil {
			log.Printf("Failed to fetch transaction for signature %s: %v", sig.Signature, err)
			continue
		}

		var txnType string
		var txnFound bool
		var goatBalances []float64
		var solBalances []float64

		for _, preBalance := range tx.Meta.PreTokenBalances {
			for _, postBalance := range tx.Meta.PostTokenBalances {
				if preBalance.Mint == postBalance.Mint {
					if postBalance.Mint == solMintAddress {
						preSOL, _ := strconv.ParseFloat(preBalance.UITokenAmount.UIAmountString, 64)
						postSOL, _ := strconv.ParseFloat(postBalance.UITokenAmount.UIAmountString, 64)

						if postSOL > preSOL {
							txnType = "Buy"
							solBalances = append(solBalances, postSOL-preSOL)
						} else if postSOL < preSOL {
							txnType = "Sell"
							solBalances = append(solBalances, preSOL-postSOL)
						}
					}

					if postBalance.Mint == goatMintAddress {
						preGOAT, _ := strconv.ParseFloat(preBalance.UITokenAmount.UIAmountString, 64)
						postGOAT, _ := strconv.ParseFloat(postBalance.UITokenAmount.UIAmountString, 64)

						goatBalances = append(goatBalances, postGOAT-preGOAT)
					}
				}
			}
		}

		blockTime := time.Unix(*tx.BlockTime, 0).Format("2006-01-02 15:04:05")

		if !txnFound {
			lastSol := solBalances[len(solBalances)-1]
			lastGoat := goatBalances[len(goatBalances)-1]

			if lastGoat == 0 || lastSol == 0 {
				continue
			}

			transactions = append(transactions, TransactionData{
				Date: blockTime,
				Type: txnType,
				GOAT: roundTo6Decimal(lastGoat),
				SOL:  roundTo9Decimal(lastSol),
				Txn:  sig.Signature,
			})
			txnFound = true
		}
	}

	writeTransactionToCSV(transactions)
	fmt.Println("CSV file created successfully!")
}

func roundTo6Decimal(value float64) float64 {
	return math.Abs(math.Trunc(value*1e6) / 1e6)
	
}

func roundTo9Decimal(value float64) float64 {
	return math.Abs(math.Trunc(value*1e9) / 1e9)
}
