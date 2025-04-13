# Bitcoin Wallet Tool

A comprehensive command-line utility for Bitcoin wallet management, address generation, and blockchain exploration.

## Features

- **Check Wallet Balance**: Query the balance of any Bitcoin address
- **View Transaction History**: Explore transaction details for any Bitcoin address
- **Generate Addresses from Seed Phrase**: Create addresses using different BIP standards (BIP32, BIP44, BIP49, BIP84)
- **Multi-language Support**: Available in English, Chinese, Japanese, and Russian
- **Mempool API Integration**: Connects to multiple Mempool.space API endpoints with automatic failover
- **Terminal UI**: Retro-styled terminal interface with color coding and visual effects

## Installation

```bash
# Clone the repository
git clone https://github.com/nordzlos/bitcoin-wallet-tool.git

# Navigate to the directory
cd bitcoin-wallet-tool

# Build the application
go build -o bitcoin-wallet-tool

# Run the application
./bitcoin-wallet-tool
