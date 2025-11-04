package wallet

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"time"

	breez_sdk_common "github.com/breez/breez-sdk-spark-go/breez_sdk_common"
	breez_sdk_spark "github.com/breez/breez-sdk-spark-go/breez_sdk_spark"
	"github.com/breez/tiny-spark/config"
)

type Wallet struct {
	sdk    *breez_sdk_spark.BreezSdk
	config *config.Config
}

type Balance struct {
	LightningBalanceSats int64
	MaxPayableSats       int64
	MaxReceivableSats    int64
}

type Transaction struct {
	ID            string
	AmountSats    int64
	FeeSats       int64
	Status        string
	Type          string
	Description   string
	Timestamp     time.Time
	PaymentHash   string
}

type ReceivePaymentResponse struct {
	PaymentRequest string
	AmountSats     int64
	FeeSats        int64
	Description    string
	ExpiresAt      time.Time
}

type PaymentResponse struct {
	PaymentHash string
	AmountSats  int64
	FeeSats     int64
	Status      string
	Preimage    string
	CompletedAt time.Time
}

type TokenBalance struct {
	TokenID  string
	Balance  string
	Name     string
	Ticker   string
	Decimals int
}

// NewWallet initializes a new Breez SDK wallet
func NewWallet(cfg *config.Config) (*Wallet, error) {
	// Create working directory if it doesn't exist
	if err := createWorkingDir(cfg.BreezWorkingDir); err != nil {
		return nil, fmt.Errorf("failed to create working directory: %w", err)
	}

	// Create SDK configuration
	network := networkFromString(cfg.BreezNetwork)
	sdkConfig := breez_sdk_spark.DefaultConfig(network)
	if sdkConfig.ApiKey != nil {
		*sdkConfig.ApiKey = cfg.BreezAPIKey
	} else {
		sdkConfig.ApiKey = &cfg.BreezAPIKey
	}
	sdkConfig.SyncIntervalSecs = 60 // Use longer sync interval for better data

	// Create seed from mnemonic
	seed := breez_sdk_spark.SeedMnemonic{
		Mnemonic: cfg.BreezMnemonic,
	}

	// Connect to SDK
	request := breez_sdk_spark.ConnectRequest{
		Config:     sdkConfig,
		Seed:       seed,
		StorageDir: cfg.BreezWorkingDir,
	}

	sdk, err := breez_sdk_spark.Connect(request)

	// Handle error using official SDK pattern
	if sdkErr := err.(*breez_sdk_spark.SdkError); sdkErr != nil {
		return nil, fmt.Errorf("failed to connect to Breez SDK: %w", err)
	}

	// Wait longer for initial sync
	time.Sleep(10 * time.Second)

	wallet := &Wallet{
		sdk:    sdk,
		config: cfg,
	}

	return wallet, nil
}

// Close closes the SDK connection
func (w *Wallet) Close() error {
	if w.sdk != nil {
		return w.sdk.Disconnect()
	}
	return nil
}

// GetBalance retrieves the wallet balance
func (w *Wallet) GetBalance(ctx context.Context) (*Balance, error) {
	req := breez_sdk_spark.GetInfoRequest{}
	info, err := w.sdk.GetInfo(req)

	// Handle error using official SDK pattern
	if sdkErr := err.(*breez_sdk_spark.SdkError); sdkErr != nil {
		return nil, fmt.Errorf("failed to get wallet info: %w", err)
	}

	// Try both BalanceSats and TokenBalances
	balanceSats := int64(info.BalanceSats)

	// Check if there are token balances (might be the actual Lightning balance)
	if len(info.TokenBalances) > 0 {
		for _, balance := range info.TokenBalances {
			tokenBalance := balance.Balance.Int64()
			if tokenBalance > 0 {
				balanceSats = tokenBalance
				break
			}
		}
	}

	return &Balance{
		LightningBalanceSats: balanceSats,
		MaxPayableSats:       balanceSats,
		MaxReceivableSats:    balanceSats,
	}, nil
}

// GetTransactions retrieves transaction history
func (w *Wallet) GetTransactions(ctx context.Context, limit int) ([]*Transaction, error) {
	offsetPtr := uint32(0)
	limitPtr := uint32(limit)
	if limitPtr < 10 {
		limitPtr = 100 // Use higher limit like the WebAssembly example
	}

	// Simple request structure exactly matching the WebAssembly example
	req := breez_sdk_spark.ListPaymentsRequest{
		Offset: &offsetPtr,
		Limit:  &limitPtr,
	}

	response, err := w.sdk.ListPayments(req)

	// Handle error using official SDK pattern
	if sdkErr := err.(*breez_sdk_spark.SdkError); sdkErr != nil {
		return nil, fmt.Errorf("failed to get transaction history: %w", err)
	}

	transactions := make([]*Transaction, len(response.Payments))
	for i, payment := range response.Payments {
		var txType string

		// Get raw amounts from SDK
		rawAmount := payment.Amount.Int64()
		fee := payment.Fees.Int64()
		var amount int64

		// Use PaymentType enum for classification and amount sign correction
		switch payment.PaymentType {
		case breez_sdk_spark.PaymentTypeReceive:
			txType = "receive"
			// Keep amount positive for receive transactions
			amount = rawAmount
		case breez_sdk_spark.PaymentTypeSend:
			txType = "send"
			// Make amount negative for send transactions
			amount = -rawAmount
		default:
			// Fallback to amount-based classification
			if rawAmount > 0 {
				txType = "receive"
				amount = rawAmount
			} else {
				txType = "send"
				amount = rawAmount
			}
		}

		// Convert payment status to readable format
		var statusStr string
		switch payment.Status {
		case breez_sdk_spark.PaymentStatusPending:
			statusStr = "Pending"
		case breez_sdk_spark.PaymentStatusCompleted:
			statusStr = "Complete"
		case breez_sdk_spark.PaymentStatusFailed:
			statusStr = "Failed"
		default:
			statusStr = string(payment.Status)
		}

		// Use generic description for now
		description := "Payment"

		transactions[i] = &Transaction{
			ID:          payment.Id,
			AmountSats:  amount,
			FeeSats:     fee,
			Status:      statusStr,
			Type:        txType,
			Description: description,
			Timestamp:   time.Unix(int64(payment.Timestamp), 0),
			PaymentHash: payment.Id,
		}
	}

	return transactions, nil
}

// ReceiveLightningInvoice creates a Lightning invoice for receiving payments
func (w *Wallet) ReceiveLightningInvoice(ctx context.Context, amountSats uint64, description string) (*ReceivePaymentResponse, error) {
	request := breez_sdk_spark.ReceivePaymentRequest{
		PaymentMethod: breez_sdk_spark.ReceivePaymentMethodBolt11Invoice{
			Description: description,
			AmountSats:  &amountSats,
		},
	}

	response, err := w.sdk.ReceivePayment(request)
	if sdkErr := err.(*breez_sdk_spark.SdkError); sdkErr != nil {
		return nil, fmt.Errorf("failed to create lightning invoice: %w", err)
	}

	return &ReceivePaymentResponse{
		PaymentRequest: response.PaymentRequest,
		FeeSats:        response.Fee.Int64(),
		AmountSats:     int64(amountSats),
		Description:    description,
		ExpiresAt:      time.Now().Add(24 * time.Hour),
	}, nil
}

// ReceiveBitcoinAddress creates a Bitcoin address for receiving on-chain payments
func (w *Wallet) ReceiveBitcoinAddress(ctx context.Context) (*ReceivePaymentResponse, error) {
	request := breez_sdk_spark.ReceivePaymentRequest{
		PaymentMethod: breez_sdk_spark.ReceivePaymentMethodBitcoinAddress{},
	}

	response, err := w.sdk.ReceivePayment(request)
	if sdkErr := err.(*breez_sdk_spark.SdkError); sdkErr != nil {
		return nil, fmt.Errorf("failed to create bitcoin address: %w", err)
	}

	return &ReceivePaymentResponse{
		PaymentRequest: response.PaymentRequest,
		FeeSats:        response.Fee.Int64(),
		AmountSats:     0, // User-specified amount
		Description:    "Bitcoin address deposit",
		ExpiresAt:      time.Now().Add(24 * time.Hour),
	}, nil
}

// ReceiveSparkAddress creates a Spark address for receiving payments
func (w *Wallet) ReceiveSparkAddress(ctx context.Context) (*ReceivePaymentResponse, error) {
	request := breez_sdk_spark.ReceivePaymentRequest{
		PaymentMethod: breez_sdk_spark.ReceivePaymentMethodSparkAddress{},
	}

	response, err := w.sdk.ReceivePayment(request)
	if sdkErr := err.(*breez_sdk_spark.SdkError); sdkErr != nil {
		return nil, fmt.Errorf("failed to create spark address: %w", err)
	}

	return &ReceivePaymentResponse{
		PaymentRequest: response.PaymentRequest,
		FeeSats:        response.Fee.Int64(),
		AmountSats:     0,
		Description:    "Spark address deposit",
		ExpiresAt:      time.Now().Add(24 * time.Hour),
	}, nil
}

// SendLightningInvoice pays a Lightning invoice
func (w *Wallet) SendLightningInvoice(ctx context.Context, bolt11 string) (*PaymentResponse, error) {
	// Prepare the payment first
	prepareReq := breez_sdk_spark.PrepareSendPaymentRequest{
		PaymentRequest: bolt11,
		Amount:         nil, // Let SDK determine amount from invoice
	}

	prepareResp, err := w.sdk.PrepareSendPayment(prepareReq)
	if sdkErr := err.(*breez_sdk_spark.SdkError); sdkErr != nil {
		return nil, fmt.Errorf("failed to prepare lightning payment: %w", err)
	}

	// Send the payment
	sendReq := breez_sdk_spark.SendPaymentRequest{
		PrepareResponse: prepareResp,
	}

	response, err := w.sdk.SendPayment(sendReq)
	if sdkErr := err.(*breez_sdk_spark.SdkError); sdkErr != nil {
		return nil, fmt.Errorf("failed to send lightning payment: %w", err)
	}

	return &PaymentResponse{
		PaymentHash:   response.Payment.Id,
		AmountSats:    response.Payment.Amount.Int64(),
		FeeSats:       response.Payment.Fees.Int64(),
		Status:        string(response.Payment.Status),
		CompletedAt:   time.Unix(int64(response.Payment.Timestamp), 0),
	}, nil
}

// SendBitcoinAddress sends Bitcoin to an on-chain address
func (w *Wallet) SendBitcoinAddress(ctx context.Context, address string, amountSats int64) (*PaymentResponse, error) {
	// Convert int64 to big.Int for SDK
	amount := big.NewInt(amountSats)

	// Prepare the payment
	prepareReq := breez_sdk_spark.PrepareSendPaymentRequest{
		PaymentRequest: address,
		Amount:         &amount,
	}

	prepareResp, err := w.sdk.PrepareSendPayment(prepareReq)
	if sdkErr := err.(*breez_sdk_spark.SdkError); sdkErr != nil {
		return nil, fmt.Errorf("failed to prepare onchain payment: %w", err)
	}

	// Send the payment with medium confirmation speed
	var options breez_sdk_spark.SendPaymentOptions = breez_sdk_spark.SendPaymentOptionsBitcoinAddress{
		ConfirmationSpeed: breez_sdk_spark.OnchainConfirmationSpeedMedium,
	}

	sendReq := breez_sdk_spark.SendPaymentRequest{
		PrepareResponse: prepareResp,
		Options:         &options,
	}

	response, err := w.sdk.SendPayment(sendReq)
	if sdkErr := err.(*breez_sdk_spark.SdkError); sdkErr != nil {
		return nil, fmt.Errorf("failed to send onchain payment: %w", err)
	}

	return &PaymentResponse{
		PaymentHash:   response.Payment.Id,
		AmountSats:    response.Payment.Amount.Int64(),
		FeeSats:       response.Payment.Fees.Int64(),
		Status:        string(response.Payment.Status),
		CompletedAt:   time.Unix(int64(response.Payment.Timestamp), 0),
	}, nil
}

// SendSparkAddress sends to a Spark address
func (w *Wallet) SendSparkAddress(ctx context.Context, sparkAddress string, amountSats int64) (*PaymentResponse, error) {
	// Convert int64 to big.Int for SDK
	amount := big.NewInt(amountSats)

	// Prepare the payment
	prepareReq := breez_sdk_spark.PrepareSendPaymentRequest{
		PaymentRequest: sparkAddress,
		Amount:         &amount,
	}

	prepareResp, err := w.sdk.PrepareSendPayment(prepareReq)
	if sdkErr := err.(*breez_sdk_spark.SdkError); sdkErr != nil {
		return nil, fmt.Errorf("failed to prepare spark payment: %w", err)
	}

	// Send the payment
	sendReq := breez_sdk_spark.SendPaymentRequest{
		PrepareResponse: prepareResp,
	}

	response, err := w.sdk.SendPayment(sendReq)
	if sdkErr := err.(*breez_sdk_spark.SdkError); sdkErr != nil {
		return nil, fmt.Errorf("failed to send spark payment: %w", err)
	}

	return &PaymentResponse{
		PaymentHash:   response.Payment.Id,
		AmountSats:    response.Payment.Amount.Int64(),
		FeeSats:       response.Payment.Fees.Int64(),
		Status:        string(response.Payment.Status),
		CompletedAt:   time.Unix(int64(response.Payment.Timestamp), 0),
	}, nil
}

// GetPayment retrieves a specific payment by ID
func (w *Wallet) GetPayment(ctx context.Context, paymentID string) (*Transaction, error) {
	req := breez_sdk_spark.GetPaymentRequest{
		PaymentId: paymentID,
	}

	response, err := w.sdk.GetPayment(req)
	if sdkErr := err.(*breez_sdk_spark.SdkError); sdkErr != nil {
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	payment := response.Payment
	var txType string
	if payment.Amount.Int64() > 0 {
		txType = "receive"
	} else {
		txType = "send"
	}

	// Convert payment status to readable format
	var statusStr string
	switch payment.Status {
	case breez_sdk_spark.PaymentStatusPending:
		statusStr = "Pending"
	case breez_sdk_spark.PaymentStatusCompleted:
		statusStr = "Complete"
	case breez_sdk_spark.PaymentStatusFailed:
		statusStr = "Failed"
	default:
		statusStr = string(payment.Status)
	}

	return &Transaction{
		ID:          payment.Id,
		AmountSats:  payment.Amount.Int64(),
		FeeSats:     payment.Fees.Int64(),
		Status:      statusStr,
		Type:        txType,
		Description: "Payment",
		Timestamp:   time.Unix(int64(payment.Timestamp), 0),
		PaymentHash: payment.Id,
	}, nil
}

// LnUrlPay prepares and sends LNURL payments
func (w *Wallet) LnUrlPay(ctx context.Context, lnurlAddress string, amountSats uint64, comment string) (*PaymentResponse, error) {
	// Parse the LNURL address
	input, err := w.sdk.Parse(lnurlAddress)
	if sdkErr := err.(*breez_sdk_spark.SdkError); sdkErr != nil {
		return nil, fmt.Errorf("failed to parse lnurl address: %w", err)
	}

	switch inputType := input.(type) {
	case breez_sdk_common.InputTypeLightningAddress:
		validateSuccessActionUrl := true

		prepareReq := breez_sdk_spark.PrepareLnurlPayRequest{
			AmountSats:               amountSats,
			PayRequest:               inputType.Field0.PayRequest,
			Comment:                  &comment,
			ValidateSuccessActionUrl: &validateSuccessActionUrl,
		}

		prepareResp, err := w.sdk.PrepareLnurlPay(prepareReq)
		if sdkErr := err.(*breez_sdk_spark.SdkError); sdkErr != nil {
			return nil, fmt.Errorf("failed to prepare lnurl pay: %w", err)
		}

		// Send the LNURL payment
		payReq := breez_sdk_spark.LnurlPayRequest{
			PrepareResponse: prepareResp,
		}

		response, err := w.sdk.LnurlPay(payReq)
		if sdkErr := err.(*breez_sdk_spark.SdkError); sdkErr != nil {
			return nil, fmt.Errorf("failed to send lnurl payment: %w", err)
		}

		return &PaymentResponse{
			PaymentHash:   response.Payment.Id,
			AmountSats:    response.Payment.Amount.Int64(),
			FeeSats:       response.Payment.Fees.Int64(),
			Status:        string(response.Payment.Status),
			CompletedAt:   time.Unix(int64(response.Payment.Timestamp), 0),
		}, nil
	}

	return nil, fmt.Errorf("unsupported LNURL address type")
}

// GetTokenBalances retrieves token balances
func (w *Wallet) GetTokenBalances(ctx context.Context) ([]*TokenBalance, error) {
	ensureSynced := false
	info, err := w.sdk.GetInfo(breez_sdk_spark.GetInfoRequest{
		EnsureSynced: &ensureSynced,
	})

	if sdkErr := err.(*breez_sdk_spark.SdkError); sdkErr != nil {
		return nil, fmt.Errorf("failed to get token balances: %w", err)
	}

	var balances []*TokenBalance
	for tokenId, tokenBalance := range info.TokenBalances {
		balances = append(balances, &TokenBalance{
			TokenID:   tokenId,
			Balance:   tokenBalance.Balance.String(),
			Name:      tokenBalance.TokenMetadata.Name,
			Ticker:    tokenBalance.TokenMetadata.Ticker,
			Decimals:  int(tokenBalance.TokenMetadata.Decimals),
		})
	}

	return balances, nil
}

// Helper functions

// createWorkingDir creates the working directory if it doesn't exist
func createWorkingDir(path string) error {
	if path == "" {
		return fmt.Errorf("working directory path cannot be empty")
	}

	// Create the directory with all necessary parent directories
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create working directory %s: %w", path, err)
	}

	// Verify the directory exists and is accessible
	if stat, err := os.Stat(path); err != nil {
		return fmt.Errorf("working directory %s is not accessible: %w", path, err)
	} else if !stat.IsDir() {
		return fmt.Errorf("working directory path %s is not a directory", path)
	}

	return nil
}

// networkFromString converts network string to SDK Network type
func networkFromString(network string) breez_sdk_spark.Network {
	switch network {
	case "mainnet":
		return breez_sdk_spark.NetworkMainnet
	case "testnet":
		return breez_sdk_spark.NetworkRegtest // Use regtest for testnet as fallback
	default:
		return breez_sdk_spark.NetworkRegtest
	}
}