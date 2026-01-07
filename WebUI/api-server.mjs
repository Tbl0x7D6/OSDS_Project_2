// Simple Express API Server for Blockchain CLI
// Place this file in the WebUI directory and run: node api-server.js

import express from 'express';
import { exec } from 'child_process';
import { promisify } from 'util';
import cors from 'cors';

const execAsync = promisify(exec);
const app = express();

// Middleware
app.use(cors());
app.use(express.json());

// CLI path
const CLI_PATH = '../bin/client';
const DEFAULT_MINER = 'localhost:8001';

/**
 * Execute CLI command and return JSON result
 */
async function executeCLI(command) {
  try {
    const { stdout, stderr } = await execAsync(command);
    if (stderr) {
      console.error('CLI stderr:', stderr);
    }
    return JSON.parse(stdout);
  } catch (error) {
    // Try to parse stdout even if command failed
    if (error.stdout) {
      try {
        return JSON.parse(error.stdout);
      } catch {
        // Fall through to error return
      }
    }
    return { error: error.message || 'Unknown error' };
  }
}

/**
 * POST /api/wallet/generate
 * Generate a new wallet
 */
app.post('/api/wallet/generate', async (req, res) => {
  try {
    const result = await executeCLI(`${CLI_PATH} wallet`);
    
    if (result.error) {
      return res.status(500).json(result);
    }
    
    res.json(result);
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

/**
 * GET /api/blockchain/status
 * Get blockchain status
 * Query params: miner, detail
 */
app.get('/api/blockchain/status', async (req, res) => {
  try {
    const miner = req.query.miner || DEFAULT_MINER;
    const detail = req.query.detail === 'true';
    
    const cmd = `${CLI_PATH} blockchain -miner ${miner}${detail ? ' -detail' : ''}`;
    const result = await executeCLI(cmd);
    
    if (result.error) {
      return res.status(500).json(result);
    }
    
    res.json(result);
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

/**
 * GET /api/wallet/:address/balance
 * Get wallet balance
 * Query params: miner
 */
app.get('/api/wallet/:address/balance', async (req, res) => {
  try {
    const address = req.params.address;
    const miner = req.query.miner || DEFAULT_MINER;
    
    if (!address) {
      return res.status(400).json({ error: 'Address is required' });
    }
    
    const cmd = `${CLI_PATH} balance -address ${address} -miner ${miner}`;
    const result = await executeCLI(cmd);
    
    if (result.error) {
      return res.status(500).json(result);
    }
    
    res.json(result);
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

/**
 * POST /api/transaction/transfer
 * Send a transfer transaction
 * Body: { from, privateKey, inputs, to, amount, changeTo, miner }
 * inputs format: "txid:outindex,txid:outindex,..."
 */
app.post('/api/transaction/transfer', async (req, res) => {
  try {
    const { from, privateKey, inputs, to, amount, changeTo, miner } = req.body;
    
    if (!from || !privateKey || !inputs || !to || !amount) {
      return res.status(400).json({ 
        error: 'Missing required fields: from, privateKey, inputs, to, amount' 
      });
    }
    
    const minerAddr = miner || DEFAULT_MINER;
    const changeAddr = changeTo || '';
    
    let cmd = `${CLI_PATH} transfer -from "${from}" -privkey "${privateKey}" -inputs "${inputs}" -to "${to}" -amount ${amount} -miner ${minerAddr}`;
    
    if (changeAddr) {
      cmd += ` -changeto "${changeAddr}"`;
    }
    
    const result = await executeCLI(cmd);
    
    if (result.error) {
      return res.status(500).json(result);
    }
    
    res.json(result);
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
});

/**
 * GET /api/health
 * Health check
 */
app.get('/api/health', (req, res) => {
  res.json({ status: 'ok', timestamp: new Date().toISOString() });
});

// Start server
const PORT = process.env.PORT || 3000;
app.listen(PORT, () => {
  console.log(`✅ API Server running on http://localhost:${PORT}`);
  console.log('');
  console.log('Available endpoints:');
  console.log(`  POST   http://localhost:${PORT}/api/wallet/generate`);
  console.log(`  GET    http://localhost:${PORT}/api/blockchain/status`);
  console.log(`  GET    http://localhost:${PORT}/api/wallet/:address/balance`);
  console.log(`  POST   http://localhost:${PORT}/api/transaction/transfer`);
  console.log(`  GET    http://localhost:${PORT}/api/health`);
  console.log('');
  console.log('Using CLI path:', CLI_PATH);
  console.log('Default miner:', DEFAULT_MINER);
});

export default app;
