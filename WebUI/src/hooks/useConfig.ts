// Custom hook for managing application configuration
import { useState, useEffect, useCallback } from 'react';

export interface AppConfig {
  minerAddress: string;
  apiUrl: string;
}

const DEFAULT_CONFIG: AppConfig = {
  minerAddress: 'localhost:8001',
  apiUrl: 'http://localhost:3000/api',
};

const CONFIG_KEYS = {
  MINER_ADDRESS: 'minerAddress',
  API_URL: 'apiUrl',
};

/**
 * Custom hook for managing application configuration stored in localStorage
 * @returns Config object with minerAddress, apiUrl and updateConfig function
 */
export function useConfig() {
  const [config, setConfig] = useState<AppConfig>(() => {
    // Initialize from localStorage or use defaults
    const minerAddress = localStorage.getItem(CONFIG_KEYS.MINER_ADDRESS) || DEFAULT_CONFIG.minerAddress;
    const apiUrl = localStorage.getItem(CONFIG_KEYS.API_URL) || DEFAULT_CONFIG.apiUrl;
    
    return {
      minerAddress,
      apiUrl,
    };
  });

  // Listen for storage changes from other tabs/windows
  useEffect(() => {
    const handleStorageChange = (e: StorageEvent) => {
      if (e.key === CONFIG_KEYS.MINER_ADDRESS || e.key === CONFIG_KEYS.API_URL) {
        setConfig({
          minerAddress: localStorage.getItem(CONFIG_KEYS.MINER_ADDRESS) || DEFAULT_CONFIG.minerAddress,
          apiUrl: localStorage.getItem(CONFIG_KEYS.API_URL) || DEFAULT_CONFIG.apiUrl,
        });
      }
    };

    window.addEventListener('storage', handleStorageChange);
    return () => window.removeEventListener('storage', handleStorageChange);
  }, []);

  const updateConfig = useCallback((newConfig: Partial<AppConfig>) => {
    setConfig((prev) => {
      const updated = { ...prev, ...newConfig };
      
      // Save to localStorage
      if (newConfig.minerAddress !== undefined) {
        localStorage.setItem(CONFIG_KEYS.MINER_ADDRESS, newConfig.minerAddress);
      }
      if (newConfig.apiUrl !== undefined) {
        localStorage.setItem(CONFIG_KEYS.API_URL, newConfig.apiUrl);
      }
      
      return updated;
    });
  }, []);

  return {
    ...config,
    updateConfig,
  };
}
