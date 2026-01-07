// API service for interacting with blockchain CLI

import type {
  WalletOutput,
  BlockchainStatusOutput,
  WalletStatusOutput,
  ErrorOutput,
} from '../types/blockchain';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:3000/api';

export class BlockchainAPI {
  /**
   * Generate a new wallet
   */
  static async generateWallet(): Promise<WalletOutput | ErrorOutput> {
    try {
      const response = await fetch(`${API_BASE_URL}/wallet/generate`, {
        method: 'POST',
      });
      return await response.json();
    } catch (error: unknown) {
      return { error: error instanceof Error ? error.message : 'Unknown error' };
    }
  }

  /**
   * Get blockchain status
   */
  static async getBlockchainStatus(
    minerAddr?: string,
    includeDetail: boolean = false
  ): Promise<BlockchainStatusOutput | ErrorOutput> {
    try {
      const params = new URLSearchParams();
      if (minerAddr) params.append('miner', minerAddr);
      if (includeDetail) params.append('detail', 'true');

      const response = await fetch(`${API_BASE_URL}/blockchain/status?${params}`);
      return await response.json();
    } catch (error: unknown) {
      return { error: error instanceof Error ? error.message : 'Unknown error' };
    }
  }

  /**
   * Get wallet balance
   */
  static async getWalletBalance(
    address: string,
    minerAddr?: string
  ): Promise<WalletStatusOutput | ErrorOutput> {
    try {
      const params = new URLSearchParams();
      if (minerAddr) params.append('miner', minerAddr);

      const response = await fetch(
        `${API_BASE_URL}/wallet/${address}/balance?${params}`
      );
      return await response.json();
    } catch (error: unknown) {
      return { error: error instanceof Error ? error.message : 'Unknown error' };
    }
  }
}

export function isErrorOutput(response: unknown): response is ErrorOutput {
  return (
    typeof response === 'object' &&
    response !== null &&
    'error' in response
  );
}
