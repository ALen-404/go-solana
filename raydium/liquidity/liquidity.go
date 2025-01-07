package liquidity

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/ALen-404/go-solana/raydium/layouts"
	"github.com/ALen-404/go-solana/raydium/utils"
)

type Liquidity struct {
	connection *rpc.Client
}

func New(connection *rpc.Client) *Liquidity {
	return &Liquidity{
		connection: connection,
	}
}

func (l *Liquidity) FetchInfo(poolKeys *layouts.ApiPoolInfoV4) (*LiquidityPoolInfo, error) {
	var LiquidityPoolInfo LiquidityPoolInfo
	instructions := l.makeSimulatePoolInfoInstruction(poolKeys)

	logs, err := l.simulateAmountsOut(instructions)
	if err != nil {
		return nil, err
	}

	if len(logs.Value.Logs) == 0 {
		return nil, fmt.Errorf("pool info unavailable")
	}

	for _, log := range logs.Value.Logs {
		if strings.Contains(log, "GetPoolData") {
			jsonLog := l.parseLog2Json(log, "GetPoolData")
			json.Unmarshal([]byte(jsonLog), &LiquidityPoolInfo)
			return &LiquidityPoolInfo, nil
		}
	}
	return nil, fmt.Errorf("pool info unavailable")
}

func (l *Liquidity) GetAmountsOut(poolKeys *layouts.ApiPoolInfoV4, amountIn *utils.TokenAmount, slippage *utils.Percent) (*AmountsOut, error) {
    // Fetch pool information
    poolInfo, err := l.FetchInfo(poolKeys)
    if err != nil {
        return &AmountsOut{}, err
    }

    // Reserves in pool (Base and Quote)
    reserves := []uint64{poolInfo.BaseReserve, poolInfo.QuoteReserve}
    
    // Define the tokens for base and quote
    tokens := []utils.Token{
        *utils.NewToken("", poolKeys.BaseMint.String(), poolInfo.BaseDecimals),
        *utils.NewToken("", poolKeys.QuoteMint.String(), poolInfo.QuoteDecimals),
    }

    // If the input token is not the base, reverse the reserves and tokens
    if amountIn.Mint != poolKeys.BaseMint.String() {
        for i, j := 0, len(reserves)-1; i < j; i, j = i+1, j-1 {
            reserves[i], reserves[j] = reserves[j], reserves[i]
        }
        for i, j := 0, len(tokens)-1; i < j; i, j = i+1, j-1 {
            tokens[i], tokens[j] = tokens[j], tokens[i]
        }
    }

    // Convert reserves to big.Int
    reserveIn := big.NewInt(int64(reserves[0]))
    reserveOut := big.NewInt(int64(reserves[1]))

    // Define the tokens
    inTok, outTok := tokens[0], tokens[1]

    // Adjust amountIn based on token decimals
    amountIn = utils.NewTokenAmount(&inTok, amountIn.Amount*math.Pow(10, float64(inTok.Decimals)))

    // Convert reserves and amountIn to big.Float for precise floating-point calculations
    reserveInFloat := new(big.Float).SetInt(reserveIn)
    reserveOutFloat := new(big.Float).SetInt(reserveOut)

    // Compute the denominator
    denominator := new(big.Float).Add(reserveInFloat, new(big.Float).SetInt(big.NewInt(int64(amountIn.Amount))))

    // Calculate the amount out
    amountOutFloat := new(big.Float).Mul(reserveOutFloat, new(big.Float).SetInt(big.NewInt(int64(amountIn.Amount))))
    amountOutFloat.Quo(amountOutFloat, denominator)

    // Convert amountOut to float64
    amountOut, _ := amountOutFloat.Float64()

    // Create the output token amount
    amountOutTok := utils.NewTokenAmount(&outTok, amountOut)

    // Calculate the minimum output amount based on slippage
    minAmountOut := utils.NewTokenAmount(
        &outTok,
        float64(uint64(amountOutTok.Amount)*uint64(float64(slippage.Denominator)-float64(slippage.Numerator))/slippage.Denominator),
    )

    return &AmountsOut{
        AmountIn:     amountIn,
        AmountOut:    amountOutTok,
        MinAmountOut: minAmountOut,
    }, nil
}


func (l *Liquidity) parseLog2Json(log string, keyword string) string {
	jsonData := strings.Split(log, keyword+": ")[1]
	return jsonData
}

func (l *Liquidity) makeSimulatePoolInfoInstruction(poolKeys *layouts.ApiPoolInfoV4) []solana.Instruction {
	layout := &SimulateStruct{
		Instruction:  12,
		SimulateType: 0,
	}
	data, err := layout.Encode()

	if err != nil {
		panic(err)
	}

	keys := solana.AccountMetaSlice{}
	keys.Append(solana.Meta(poolKeys.ID))
	keys.Append(solana.Meta(poolKeys.Authority))
	keys.Append(solana.Meta(poolKeys.OpenOrders))
	keys.Append(solana.Meta(poolKeys.BaseVault))
	keys.Append(solana.Meta(poolKeys.QuoteVault))
	keys.Append(solana.Meta(poolKeys.LpMint))
	keys.Append(solana.Meta(poolKeys.MarketId))
	keys.Append(solana.Meta(poolKeys.MarketEventQueue))

	return []solana.Instruction{
		solana.NewInstruction(
			poolKeys.ProgramId,
			keys,
			data,
		),
	}
}

func (l *Liquidity) simulateAmountsOut(instructions []solana.Instruction) (*rpc.SimulateTransactionResponse, error) {
	feePayer := solana.MustPublicKeyFromBase58("RaydiumSimuLateTransaction11111111111111111")
	recent, err := l.connection.GetLatestBlockhash(context.Background(), rpc.CommitmentFinalized)
	if err != nil {
		return &rpc.SimulateTransactionResponse{}, err
	}
	tx, _ := solana.NewTransaction(
		instructions,
		recent.Value.Blockhash,
		solana.TransactionPayer(feePayer),
	)
	tx.Signatures = make([]solana.Signature, 1)
	tx.Signatures[0] = solana.MustSignatureFromBase58("1111111111111111111111111111111111111111111111111111111111111111") // If you know better way to do this, feel free to change
	return l.connection.SimulateTransaction(context.Background(), tx)
}
