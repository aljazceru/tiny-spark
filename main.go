package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/breez/tiny-spark/config"
	"github.com/breez/tiny-spark/wallet"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := os.Args[1]

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize wallet
	w, err := wallet.NewWallet(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize wallet: %v", err)
	}
	defer w.Close()

	ctx := context.Background()

	switch command {
	case "balance", "bal":
		showBalance(ctx, w)
	case "transactions", "tx":
		limit := 10
		if len(os.Args) > 2 {
			if l, err := strconv.Atoi(os.Args[2]); err == nil {
				limit = l
			}
		}
		showTransactions(ctx, w, limit)
	case "receive":
		if len(os.Args) < 4 {
			fmt.Println("Usage: tiny-client receive <type> <amount> [description]")
			fmt.Println("Types: lightning, bitcoin, spark")
			return
		}
		receivePayment(ctx, w, os.Args[2], os.Args[3], strings.Join(os.Args[4:], " "))
	case "send":
		if len(os.Args) < 4 {
			fmt.Println("Usage: tiny-client send <type> <destination> <amount>")
			fmt.Println("Types: lightning, bitcoin, spark, lnurl")
			return
		}
		sendPayment(ctx, w, os.Args[2], os.Args[3], os.Args[4])
	case "payment":
		if len(os.Args) < 3 {
			fmt.Println("Usage: tiny-client payment <payment_id>")
			return
		}
		showPayment(ctx, w, os.Args[2])
	case "tokens":
		showTokens(ctx, w)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
	}
}

func printUsage() {
	fmt.Println("Breez Tiny Spark")
	fmt.Println("==================")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  tiny-spark <command> [arguments]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  balance, bal                    Show wallet balance")
	fmt.Println("  transactions, tx [limit]       Show transaction history (default 10)")
	fmt.Println("  receive <type> <amount> [desc]  Create payment request")
	fmt.Println("  send <type> <dest> <amount>    Send payment")
	fmt.Println("  payment <id>                   Show payment details")
	fmt.Println("  tokens                         Show token balances")
	fmt.Println("  help                           Show this help")
	fmt.Println()
	fmt.Println("Receive types:")
	fmt.Println("  lightning    Create Lightning invoice")
	fmt.Println("  bitcoin      Create Bitcoin address")
	fmt.Println("  spark        Create Spark address")
	fmt.Println()
	fmt.Println("Send types:")
	fmt.Println("  lightning    Pay Lightning invoice")
	fmt.Println("  bitcoin      Send to Bitcoin address")
	fmt.Println("  spark        Send to Spark address")
	fmt.Println("  lnurl        Pay LNURL address")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  tiny-spark balance")
	fmt.Println("  tiny-spark receive lightning 5000 'Coffee payment'")
	fmt.Println("  tiny-spark send lightning lnbc1... 5000")
	fmt.Println("  tiny-spark transactions 20")
}

func showBalance(ctx context.Context, w *wallet.Wallet) {
	fmt.Println("Wallet Balance:")
	fmt.Println("----------------")
	balance, err := w.GetBalance(ctx)
	if err != nil {
		log.Fatalf("Failed to get balance: %v", err)
	}

	fmt.Printf("Lightning Balance: %d sats\n", balance.LightningBalanceSats)
	fmt.Printf("Max Payable:       %d sats\n", balance.MaxPayableSats)
	fmt.Printf("Max Receivable:    %d sats\n", balance.MaxReceivableSats)
}

func showTransactions(ctx context.Context, w *wallet.Wallet, limit int) {
	fmt.Printf("Last %d Transactions:\n", limit)
	fmt.Println(strings.Repeat("-", 20))

	transactions, err := w.GetTransactions(ctx, limit)
	if err != nil {
		log.Fatalf("Failed to get transactions: %v", err)
	}

	if len(transactions) == 0 {
		fmt.Println("No transactions found")
		return
	}

	// Use tabwriter for nice formatting
	tabWriter := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tabWriter, "TIME\tTYPE\tAMOUNT\tFEE\tSTATUS\tDESCRIPTION")
	fmt.Fprintln(tabWriter, "----\t----\t------\t---\t------\t-----------")

	for _, tx := range transactions {
		timestamp := tx.Timestamp.Format("2006-01-02 15:04")
		amountStr := formatAmount(tx.AmountSats)
		feeStr := formatAmount(tx.FeeSats)
		description := truncateString(tx.Description, 20)
		if description == "" {
			description = "-"
		}

		fmt.Fprintf(tabWriter, "%s\t%s\t%s\t%s\t%s\t%s\n",
			timestamp, tx.Type, amountStr, feeStr, tx.Status, description)
	}
	tabWriter.Flush()
}

func receivePayment(ctx context.Context, w *wallet.Wallet, paymentType, amountStr, description string) {
	amount, err := strconv.ParseUint(amountStr, 10, 64)
	if err != nil {
		log.Fatalf("Invalid amount: %v", err)
	}

	if description == "" {
		description = "Payment request"
	}

	var response *wallet.ReceivePaymentResponse

	switch strings.ToLower(paymentType) {
	case "lightning", "ln":
		response, err = w.ReceiveLightningInvoice(ctx, amount, description)
	case "bitcoin", "btc":
		response, err = w.ReceiveBitcoinAddress(ctx)
	case "spark":
		response, err = w.ReceiveSparkAddress(ctx)
	default:
		log.Fatalf("Unknown receive type: %s", paymentType)
	}

	if err != nil {
		log.Fatalf("Failed to create %s payment request: %v", paymentType, err)
	}

	fmt.Printf("Payment Request Created:\n")
	fmt.Printf("Type:        %s\n", strings.Title(paymentType))
	fmt.Printf("Amount:      %d sats\n", response.AmountSats)
	fmt.Printf("Fee:         %d sats\n", response.FeeSats)
	fmt.Printf("Description: %s\n", response.Description)
	fmt.Printf("Expires:     %s\n", response.ExpiresAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("\nPayment Request:\n%s\n", response.PaymentRequest)
}

func sendPayment(ctx context.Context, w *wallet.Wallet, paymentType, destination, amountStr string) {
	var response *wallet.PaymentResponse
	var err error

	switch strings.ToLower(paymentType) {
	case "lightning", "ln":
		response, err = w.SendLightningInvoice(ctx, destination)
	case "bitcoin", "btc":
		amount, err2 := strconv.ParseInt(amountStr, 10, 64)
		if err2 != nil {
			log.Fatalf("Invalid amount: %v", err2)
		}
		response, err = w.SendBitcoinAddress(ctx, destination, amount)
	case "spark":
		amount, err2 := strconv.ParseInt(amountStr, 10, 64)
		if err2 != nil {
			log.Fatalf("Invalid amount: %v", err2)
		}
		response, err = w.SendSparkAddress(ctx, destination, amount)
	case "lnurl":
		amount, err2 := strconv.ParseUint(amountStr, 10, 64)
		if err2 != nil {
			log.Fatalf("Invalid amount: %v", err2)
		}
		response, err = w.LnUrlPay(ctx, destination, amount, "Payment via LNURL")
	default:
		log.Fatalf("Unknown send type: %s", paymentType)
	}

	if err != nil {
		log.Fatalf("Failed to send %s payment: %v", paymentType, err)
	}

	fmt.Printf("Payment Sent:\n")
	fmt.Printf("Payment Hash: %s\n", response.PaymentHash)
	fmt.Printf("Amount:       %d sats\n", response.AmountSats)
	fmt.Printf("Fee:          %d sats\n", response.FeeSats)
	fmt.Printf("Status:       %s\n", response.Status)
	fmt.Printf("Completed:    %s\n", response.CompletedAt.Format("2006-01-02 15:04:05"))
}

func showPayment(ctx context.Context, w *wallet.Wallet, paymentID string) {
	payment, err := w.GetPayment(ctx, paymentID)
	if err != nil {
		log.Fatalf("Failed to get payment: %v", err)
	}

	fmt.Printf("Payment Details:\n")
	fmt.Printf("ID:          %s\n", payment.ID)
	fmt.Printf("Type:        %s\n", payment.Type)
	fmt.Printf("Amount:      %s sats\n", formatAmount(payment.AmountSats))
	fmt.Printf("Fee:         %s sats\n", formatAmount(payment.FeeSats))
	fmt.Printf("Status:      %s\n", payment.Status)
	fmt.Printf("Description: %s\n", payment.Description)
	fmt.Printf("Time:        %s\n", payment.Timestamp.Format("2006-01-02 15:04:05"))
}

func showTokens(ctx context.Context, w *wallet.Wallet) {
	fmt.Println("Token Balances:")
	fmt.Println("---------------")

	tokens, err := w.GetTokenBalances(ctx)
	if err != nil {
		log.Fatalf("Failed to get token balances: %v", err)
	}

	if len(tokens) == 0 {
		fmt.Println("No tokens found")
		return
	}

	tabWriter := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tabWriter, "TOKEN ID\tNAME\tTICKER\tBALANCE")
	fmt.Fprintln(tabWriter, "---------\t----\t------\t-------")

	for _, token := range tokens {
		fmt.Fprintf(tabWriter, "%s\t%s\t%s\t%s\n",
			token.TokenID, token.Name, token.Ticker, token.Balance)
	}
	tabWriter.Flush()
}

// formatAmount formats satoshi amount with proper sign
func formatAmount(sats int64) string {
	if sats == 0 {
		return "0"
	}

	if sats > 0 {
		return fmt.Sprintf("+%d", sats)
	}
	return fmt.Sprintf("%d", sats)
}

// formatStatus makes the status more readable
func formatStatus(status string) string {
	switch status {
	case "complete":
		return "Complete"
	case "pending":
		return "Pending"
	case "failed":
		return "Failed"
	default:
		return status
	}
}

// truncateString truncates a string to max length with ellipsis if needed
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}