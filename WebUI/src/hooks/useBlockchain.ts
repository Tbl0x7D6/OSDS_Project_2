// React hooks for blockchain data

import { useState, useEffect, useCallback } from 'react';
import { BlockchainAPI, isErrorOutput } from '../services/api';
import type {
  WalletOutput,
  BlockchainStatusOutput,
  WalletStatusOutput,
} from '../types/blockchain';

export function useGenerateWallet() {
  const [loading, setLoading] = useState(false);
  const [wallet, setWallet] = useState<WalletOutput | null>(null);
  const [error, setError] = useState<string | null>(null);

  const generate = useCallback(async () => {
    setLoading(true);
    setError(null);

    try {
      const result = await BlockchainAPI.generateWallet();

      if (isErrorOutput(result)) {
        setError(result.error);
        setWallet(null);
      } else {
        setWallet(result);
        setError(null);
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setWallet(null);
    } finally {
      setLoading(false);
    }
  }, []);

  return { wallet, loading, error, generate };
}

export function useBlockchainStatus(
  minerAddr?: string,
  includeDetail: boolean = false,
  autoRefresh: boolean = false,
  refreshInterval: number = 5000
) {
  const [status, setStatus] = useState<BlockchainStatusOutput | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchStatus = useCallback(async () => {
    setLoading(true);
    setError(null);

    try {
      const result = await BlockchainAPI.getBlockchainStatus(
        minerAddr,
        includeDetail
      );

      if (isErrorOutput(result)) {
        setError(result.error);
        setStatus(null);
      } else {
        setStatus(result);
        setError(null);
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setStatus(null);
    } finally {
      setLoading(false);
    }
  }, [minerAddr, includeDetail]);

  useEffect(() => {
    fetchStatus();

    if (autoRefresh) {
      const interval = setInterval(fetchStatus, refreshInterval);
      return () => clearInterval(interval);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [minerAddr, includeDetail, autoRefresh, refreshInterval]);

  return { status, loading, error, refresh: fetchStatus };
}

export function useWalletBalance(
  address?: string,
  minerAddr?: string,
  autoRefresh: boolean = false,
  refreshInterval: number = 10000
) {
  const [balance, setBalance] = useState<WalletStatusOutput | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchBalance = useCallback(async () => {
    if (!address) {
      setBalance(null);
      setError('No address provided');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const result = await BlockchainAPI.getWalletBalance(address, minerAddr);

      if (isErrorOutput(result)) {
        setError(result.error);
        setBalance(null);
      } else {
        setBalance(result);
        setError(null);
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setBalance(null);
    } finally {
      setLoading(false);
    }
  }, [address, minerAddr]);

  useEffect(() => {
    if (address) {
      fetchBalance();

      if (autoRefresh) {
        const interval = setInterval(fetchBalance, refreshInterval);
        return () => clearInterval(interval);
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [address, minerAddr, autoRefresh, refreshInterval]);

  return { balance, loading, error, refresh: fetchBalance };
}
