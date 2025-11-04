# tiny-spark

A  CLI client for spark, implementing all major Lightning Network and Bitcoin payment features based on [Breez Nodeless SDK](https://sdk-doc-spark.breez.technology/)
## Features

### Core Wallet Operations
- **Balance Query**: Display Lightning wallet balance and spendable limits
- **Transaction History**: View and filter transaction history with detailed status
- **Payment Details**: Retrieve specific payment information by ID

### Payment Reception
- **Lightning Invoices**: Create BOLT11 invoices for receiving Lightning payments
- **Bitcoin Addresses**: Generate on-chain Bitcoin addresses for deposits
- **Spark Addresses**: Create Spark addresses for instant zero-fee transfers

### Payment Sending
- **Lightning Payments**: Pay BOLT11 invoices via Lightning Network
- **On-chain Bitcoin**: Send Bitcoin to any on-chain address with configurable fees
- **Spark Transfers**: Send to Spark addresses for instant settlement
- **LNURL Support**: Pay LNURL addresses and Lightning addresses

### Token Support
- **Token Balances**: View balances for all supported tokens in the wallet
- **Token Metadata**: Access token information including names, tickers, and decimals

## Installation

```bash
# Clone the repository
git clone https://github.com/aljazceru/tiny-spark.git
cd tiny-spark

# Build the client
go build -o tiny-spark

# Ensure your .env file contains BREEZ_API_KEY and BREEZ_MNEMONIC
```

## Configuration

The client uses environment variables for configuration. **Only `BREEZ_API_KEY` and `BREEZ_MNEMONIC` are required** - all other variables have sensible defaults.

```bash
# Required variables - Must be set
BREEZ_API_KEY=your_breez_api_key
BREEZ_MNEMONIC="your twelve word mnemonic phrase"
```

## Usage

### Basic Commands

```bash
# Show wallet balance
./tiny-spark balance

# Show transaction history (default 10 transactions)
./tiny-spark transactions
./tiny-spark transactions 20  # Show last 20 transactions

# Show token balances
./tiny-spark tokens

# Get specific payment details
./tiny-spark payment <payment_id>

# Show help
./tiny-spark help
```

### Receiving Payments

```bash
# Create Lightning invoice
./tiny-spark receive lightning 5000 "Coffee payment"

# Create Bitcoin address
./tiny-spark receive bitcoin

# Create Spark address
./tiny-spark receive spark
```

### Sending Payments

```bash
# Pay Lightning invoice
./tiny-spark send lightning lnbc1... 5000

# Send to Bitcoin address
./tiny-spark send bitcoin bc1q... 50000

# Send to Spark address
./tiny-spark send spark spark... 25000

# Pay LNURL address
./tiny-spark send lnurl user@example.com 5000
```

## Examples

### Daily Operations

```bash
# Check current balance
./tiny-spark balance

# Create invoice for receiving payment
./tiny-spark receive lightning 10000 "Web development services"

# Check recent transactions
./tiny-spark transactions 5

# Pay an invoice
./tiny-spark send lightning lnbc1... 10000

# Verify payment status
./tiny-spark payment <payment_hash>
```

### Business Integration

```bash
# Generate payment address for customer
./tiny-spark receive lightning 25000 "Invoice #12345"

# Accept Bitcoin payment
./tiny-spark receive bitcoin

# Send payment to supplier
./tiny-spark send bitcoin bc1q... 100000

# Check all token balances
./tiny-spark tokens
```

## Command Reference

| Command | Description | Example |
|---------|-------------|---------|
| `balance` | Show wallet balance and limits | `./tiny-spark balance` |
| `transactions [N]` | Show last N transactions | `./tiny-spark transactions 15` |
| `receive <type> <amount> [desc]` | Create payment request | `./tiny-spark receive lightning 5000 "Payment"` |
| `send <type> <dest> <amount>` | Send payment | `./tiny-spark send lightning lnbc1... 5000` |
| `payment <id>` | Show payment details | `./tiny-spark payment abc123...` |
| `tokens` | Show token balances | `./tiny-spark tokens` |

### Payment Types

**Receive Types:**
- `lightning` / `ln` - Create BOLT11 Lightning invoice
- `bitcoin` / `btc` - Generate Bitcoin address
- `spark` - Create Spark address

**Send Types:**
- `lightning` / `ln` - Pay Lightning invoice
- `bitcoin` / `btc` - Send to Bitcoin address
- `spark` - Send to Spark address
- `lnurl` - Pay LNURL/Lightning address

## Example Output

```
Breez Tiny Spark
==================

Wallet Balance:
----------------
Lightning Balance: 5000 sats
Max Payable:       5000 sats
Max Receivable:    5000 sats

Last 10 Transactions:
--------------------
TIME              TYPE     AMOUNT  FEE  STATUS    DESCRIPTION
----              ----     ------  ---  ------    -----------
2025-11-01 15:36  receive  +12     +3   Complete  Payment
2025-10-31 15:56  receive  +5      0    Complete  Payment
2025-10-31 15:55  receive  +20     +1   Complete  Payment
2025-10-31 09:15  receive  +100    0    Complete  Payment

Payment Request Created:
Type:        Lightning
Amount:      5000 sats
Fee:         0 sats
Description: Coffee payment
Expires:     2025-11-05 10:19:36

Payment Request:
lnbc50u1p5sn3fgpp5f432vrt88n6876wt6kx7en8xj7kv99rh7qd9fcm793y7y7vz92sssp5xk2etegmu098jnza9aspfkgg39tm5ar2lndmpyjzd3ynuts8n2rqxq9z0rgqnp4qvyndeaqzman7h898jxm98dzkm0mlrsx36s93smrur7h0azyyuxc5rzjq25carzepgd4vqsyn44jrk85ezrpju92xyrk9apw4cdjh6yrwt5jgqqqqrt49lmtcqqqqqqqqqqq86qq9qrzjqwghf7zxvfkxq5a6sr65g0gdkv768p83mhsnt0msszapamzx2qvuxqqqqrt49lmtcqqqqqqqqqqq86qq9qcqzpgdq523jhxapqwpshjmt9de6q9qyyssqv30v9dmqjgjgnc2xupsvhhmyqtjgf2tm3mgh9gqxwrfhef4yamczn6hauvvwzqwxhda6mdrjamcg72rz2f7nrrgwkllnf40x0703yecq298zxl
```

## Security Considerations

- Keep your mnemonic and API key secure
- Use mainnet only for production transactions
- Verify payment details before sending
- Consider using testnet for development and testing

## Development

This client is based on the official Breez SDK Spark Go examples and implements:

- **SDK Initialization**: Proper SDK connection and configuration
- **Error Handling**: Comprehensive error handling with user-friendly messages
- **Type Safety**: Proper Go type handling for all SDK operations
- **Transaction Management**: Complete payment lifecycle support
- **Multi-Asset Support**: Bitcoin and token balance management

## Requirements

- Go 1.21+
- Breez API credentials (`BREEZ_API_KEY`)
- Valid mnemonic phrase (`BREEZ_MNEMONIC`)
- Network connectivity for Lightning/Bitcoin operations

**Environment Setup:**
```bash
# Minimum required .env file
BREEZ_API_KEY=your_breez_api_key
BREEZ_MNEMONIC="your twelve word mnemonic phrase"

# Run the application
./tiny-spark
```


