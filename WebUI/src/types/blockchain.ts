// TypeScript types for blockchain client responses

export interface WalletOutput {
  address: string;
  private_key: string;
  created_at: string;
}

export interface MinerStatus {
  ID: string;
  ChainLength: number;
  PendingTxs: number;
  Peers: number;
  Mining: boolean;
}

export interface TxInput {
  txid: string;
  out_index: number;
  scriptsig: string;
}

export interface TxOutput {
  value: number;
  scriptpubkey: string;
}

export interface TransactionOutput {
  id: string;
  inputs: TxInput[];
  outputs: TxOutput[];
  is_coinbase: boolean;
}

export interface BlockOutput {
  index: number;
  hash: string;
  prev_hash: string;
  timestamp: number;
  nonce: number;
  difficulty: number;
  miner_id: string;
  transactions: TransactionOutput[];
}

export interface BlockchainStatusOutput {
  chain_length: number;
  difficulty: number;
  latest_block_hash: string;
  latest_block_index: number;
  latest_block_miner: string;
  latest_block_time: number;
  total_transactions: number;
  miner_status?: MinerStatus;
  blocks?: BlockOutput[];
}

export interface UTXOOutput {
  txid: string;
  out_index: number;
  value: number;
  value_btc: number;
  scriptpubkey: string;
}

export interface WalletStatusOutput {
  address: string;
  balance: number;
  balance_btc: number;
  utxos: UTXOOutput[];
  utxo_count: number;
}

export interface ErrorOutput {
  error: string;
}

export const SATOSHI_PER_BTC = 100_000_000;

export function satoshiToBTC(satoshi: number): number {
  return satoshi / SATOSHI_PER_BTC;
}

export function btcToSatoshi(btc: number): number {
  return Math.floor(btc * SATOSHI_PER_BTC);
}
