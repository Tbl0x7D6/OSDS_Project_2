// Transfer Manager Component - Send transactions to other wallets

import { useState } from 'react';
import {
  Box,
  Flex,
  HStack,
  VStack,
  Text,
  Card,
  Input,
  Button,
  Badge,
  Separator,
  Checkbox,
} from '@chakra-ui/react';
import { FiSend, FiRefreshCw, FiAlertCircle, FiCheckCircle } from 'react-icons/fi';
import { BlockchainAPI, isErrorOutput } from '../services/api';
import { toaster } from './ui/toaster';
import type { WalletStatusOutput, UTXOOutput, TransferInput } from '../types/blockchain';
import { satoshiToBTC, btcToSatoshi } from '../types/blockchain';

export function TransferManager() {
  const [minerAddress] = useState('localhost:8001');
  
  // Wallet info
  const [senderAddress, setSenderAddress] = useState('');
  const [privateKey, setPrivateKey] = useState('');
  
  // Wallet balance and UTXOs
  const [walletStatus, setWalletStatus] = useState<WalletStatusOutput | null>(null);
  const [loadingBalance, setLoadingBalance] = useState(false);
  const [balanceError, setBalanceError] = useState<string | null>(null);
  
  // UTXO selection
  const [selectedUTXOs, setSelectedUTXOs] = useState<Set<string>>(new Set());
  
  // Transfer details
  const [recipientAddress, setRecipientAddress] = useState('');
  const [amountBTC, setAmountBTC] = useState('');
  const [changeAddress, setChangeAddress] = useState('');
  const [useCustomChange, setUseCustomChange] = useState(false);
  
  // Transfer status
  const [sending, setSending] = useState(false);
  const [transferResult, setTransferResult] = useState<{ success: boolean; message: string; txid?: string } | null>(null);

  // Load wallet balance and UTXOs
  const loadBalance = async () => {
    if (!senderAddress) {
      setBalanceError('请输入发送者地址');
      return;
    }

    setLoadingBalance(true);
    setBalanceError(null);
    setWalletStatus(null);
    setSelectedUTXOs(new Set());

    try {
      const result = await BlockchainAPI.getWalletBalance(senderAddress, minerAddress);
      
      if (isErrorOutput(result)) {
        setBalanceError(result.error);
      } else {
        setWalletStatus(result);
        if (result.utxo_count === 0) {
          setBalanceError('此地址没有可用的UTXO');
        }
      }
    } catch (error) {
      setBalanceError(error instanceof Error ? error.message : '加载失败');
    } finally {
      setLoadingBalance(false);
    }
  };

  // Toggle UTXO selection
  const toggleUTXO = (utxo: UTXOOutput) => {
    const key = `${utxo.txid}:${utxo.out_index}`;
    const newSelected = new Set(selectedUTXOs);
    
    if (newSelected.has(key)) {
      newSelected.delete(key);
    } else {
      newSelected.add(key);
    }
    
    setSelectedUTXOs(newSelected);
  };

  // Calculate selected UTXO total
  const getSelectedTotal = (): number => {
    if (!walletStatus) return 0;
    
    let total = 0;
    walletStatus.utxos.forEach(utxo => {
      const key = `${utxo.txid}:${utxo.out_index}`;
      if (selectedUTXOs.has(key)) {
        total += utxo.value;
      }
    });
    
    return total;
  };

  // Select all UTXOs
  const selectAllUTXOs = () => {
    if (!walletStatus) return;
    
    const allKeys = new Set(
      walletStatus.utxos.map(utxo => `${utxo.txid}:${utxo.out_index}`)
    );
    setSelectedUTXOs(allKeys);
  };

  // Clear all selections
  const clearAllUTXOs = () => {
    setSelectedUTXOs(new Set());
  };

  // Send transfer
  const handleSendTransfer = async () => {
    // Validation
    if (!senderAddress || !privateKey || !recipientAddress || !amountBTC) {
      toaster.create({
        title: '信息不完整',
        description: '请填写所有必填字段',
        type: 'error',
      });
      return;
    }

    if (selectedUTXOs.size === 0) {
      toaster.create({
        title: '未选择UTXO',
        description: '请至少选择一个UTXO作为输入',
        type: 'error',
      });
      return;
    }

    const amountSatoshi = btcToSatoshi(parseFloat(amountBTC));
    const selectedTotal = getSelectedTotal();

    if (amountSatoshi <= 0) {
      toaster.create({
        title: '金额无效',
        description: '转账金额必须大于0',
        type: 'error',
      });
      return;
    }

    if (amountSatoshi > selectedTotal) {
      toaster.create({
        title: '余额不足',
        description: `选中的UTXO总额: ${satoshiToBTC(selectedTotal).toFixed(8)} BTC，需要: ${amountBTC} BTC`,
        type: 'error',
      });
      return;
    }

    // Build inputs string
    const inputsStr = Array.from(selectedUTXOs).join(',');

    const transferData: TransferInput = {
      from: senderAddress,
      privateKey: privateKey,
      inputs: inputsStr,
      to: recipientAddress,
      amount: amountSatoshi,
      miner: minerAddress,
    };

    if (useCustomChange && changeAddress) {
      transferData.changeTo = changeAddress;
    }

    setSending(true);
    setTransferResult(null);

    try {
      const result = await BlockchainAPI.sendTransfer(transferData);

      if (isErrorOutput(result)) {
        setTransferResult({
          success: false,
          message: result.error,
        });
        toaster.create({
          title: '转账失败',
          description: result.error,
          type: 'error',
          duration: 5000,
        });
      } else {
        setTransferResult({
          success: result.success,
          message: result.message || (result.success ? '转账成功！' : result.error || '转账失败'),
          txid: result.txid,
        });
        
        if (result.success) {
          toaster.create({
            title: '转账成功',
            description: `交易ID: ${result.txid.substring(0, 16)}...`,
            type: 'success',
            duration: 5000,
          });

          // Reset form
          setSelectedUTXOs(new Set());
          setRecipientAddress('');
          setAmountBTC('');
          setChangeAddress('');
          
          // Reload balance after a short delay
          setTimeout(() => {
            loadBalance();
          }, 1000);
        } else {
          toaster.create({
            title: '转账失败',
            description: result.error || '未知错误',
            type: 'error',
            duration: 5000,
          });
        }
      }
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : '转账失败';
      setTransferResult({
        success: false,
        message: errorMsg,
      });
      toaster.create({
        title: '转账失败',
        description: errorMsg,
        type: 'error',
        duration: 5000,
      });
    } finally {
      setSending(false);
    }
  };

  return (
    <Box>
      <HStack gap={3} mb={6}>
        <FiSend size={28} />
        <Text fontSize="2xl" fontWeight="bold">
          转账管理
        </Text>
      </HStack>

      {/* Sender Wallet Info */}
      <Card.Root mb={6}>
        <Card.Header>
          <Text fontSize="lg" fontWeight="semibold">
            发送者钱包
          </Text>
        </Card.Header>
        <Card.Body>
          <VStack align="stretch" gap={4}>
            <Box>
              <Text mb={2} fontWeight="medium">发送者地址 (公钥)</Text>
              <Input
                value={senderAddress}
                onChange={(e) => setSenderAddress(e.target.value)}
                placeholder="输入发送者的公钥地址"
                fontFamily="mono"
                fontSize="sm"
              />
            </Box>

            <Box>
              <Text mb={2} fontWeight="medium" color="red.600" _dark={{ color: 'red.400' }}>
                私钥 (请勿泄露)
              </Text>
              <Input
                type="password"
                value={privateKey}
                onChange={(e) => setPrivateKey(e.target.value)}
                placeholder="输入私钥"
                fontFamily="mono"
                fontSize="sm"
                colorPalette="red"
              />
            </Box>

            <Button
              onClick={loadBalance}
              loading={loadingBalance}
              disabled={!senderAddress}
              colorScheme="blue"
            >
              <FiRefreshCw />
              加载余额和UTXO
            </Button>
          </VStack>
        </Card.Body>
      </Card.Root>

      {/* Balance Error */}
      {balanceError && (
        <Card.Root colorPalette="red" bg="red.subtle" borderColor="red.muted" mb={6}>
          <Card.Body>
            <HStack>
              <FiAlertCircle />
              <Text color="red.fg">{balanceError}</Text>
            </HStack>
          </Card.Body>
        </Card.Root>
      )}

      {/* Wallet Balance and UTXO Selection */}
      {walletStatus && !balanceError && (
        <Card.Root mb={6}>
          <Card.Header>
            <Flex justify="space-between" align="center">
              <Text fontSize="lg" fontWeight="semibold">
                选择UTXO
              </Text>
              <Badge colorScheme="green">
                余额: {walletStatus.balance_btc.toFixed(8)} BTC
              </Badge>
            </Flex>
          </Card.Header>
          <Card.Body>
            <VStack align="stretch" gap={4}>
              {/* Selection summary */}
              <Flex justify="space-between" align="center" p={3} bg="bg.muted" borderRadius="md">
                <VStack align="start" gap={1}>
                  <Text fontSize="sm" color="fg.muted">已选择 {selectedUTXOs.size} 个UTXO</Text>
                  <Text fontSize="lg" fontWeight="bold" colorPalette="green" color="green.fg">
                    总计: {satoshiToBTC(getSelectedTotal()).toFixed(8)} BTC
                  </Text>
                </VStack>
                <HStack>
                  <Button size="sm" onClick={selectAllUTXOs} variant="outline">
                    全选
                  </Button>
                  <Button size="sm" onClick={clearAllUTXOs} variant="outline">
                    清空
                  </Button>
                </HStack>
              </Flex>

              {/* UTXO list */}
              <VStack align="stretch" gap={3}>
                {walletStatus.utxos.map((utxo, index) => {
                  const key = `${utxo.txid}:${utxo.out_index}`;
                  const isSelected = selectedUTXOs.has(key);

                  return (
                    <Card.Root
                      key={key}
                      variant={isSelected ? 'elevated' : 'outline'}
                      bg={isSelected ? 'green.subtle' : 'bg'}
                      borderColor={isSelected ? 'green.muted' : 'border'}
                      cursor="pointer"
                      onClick={() => toggleUTXO(utxo)}
                    >
                      <Card.Body>
                        <Flex justify="space-between" align="center">
                          <HStack>
                            <Checkbox.Root checked={isSelected}>
                              <Checkbox.HiddenInput />
                              <Checkbox.Control>
                                <Checkbox.Indicator />
                              </Checkbox.Control>
                            </Checkbox.Root>
                            <Badge>#{index + 1}</Badge>
                          </HStack>
                          <Text fontWeight="bold" colorPalette="green" color="green.fg">
                            {utxo.value_btc.toFixed(8)} BTC
                          </Text>
                        </Flex>
                        <VStack align="stretch" gap={1} fontSize="sm" mt={2}>
                          <Flex justify="space-between">
                            <Text color="fg.muted">交易ID:</Text>
                            <Text fontFamily="mono" fontSize="xs">
                              {utxo.txid.substring(0, 16)}...
                            </Text>
                          </Flex>
                          <Flex justify="space-between">
                            <Text color="fg.muted">输出索引:</Text>
                            <Text>{utxo.out_index}</Text>
                          </Flex>
                          <Flex justify="space-between">
                            <Text color="fg.muted">金额:</Text>
                            <Text>{utxo.value.toLocaleString()} satoshi</Text>
                          </Flex>
                        </VStack>
                      </Card.Body>
                    </Card.Root>
                  );
                })}
              </VStack>
            </VStack>
          </Card.Body>
        </Card.Root>
      )}

      {/* Transfer Form */}
      {walletStatus && !balanceError && selectedUTXOs.size > 0 && (
        <Card.Root mb={6}>
          <Card.Header>
            <Text fontSize="lg" fontWeight="semibold">
              转账详情
            </Text>
          </Card.Header>
          <Card.Body>
            <VStack align="stretch" gap={4}>
              <Box>
                <Text mb={2} fontWeight="medium">接收者地址</Text>
                <Input
                  value={recipientAddress}
                  onChange={(e) => setRecipientAddress(e.target.value)}
                  placeholder="输入接收者的公钥地址"
                  fontFamily="mono"
                  fontSize="sm"
                />
              </Box>

              <Box>
                <Text mb={2} fontWeight="medium">转账金额 (BTC)</Text>
                <Input
                  type="number"
                  step="0.00000001"
                  value={amountBTC}
                  onChange={(e) => setAmountBTC(e.target.value)}
                  placeholder="0.00000000"
                />
                <Text fontSize="sm" color="fg.muted" mt={1}>
                  可用: {satoshiToBTC(getSelectedTotal()).toFixed(8)} BTC
                </Text>
              </Box>

              <Box>
                <Checkbox.Root
                  checked={useCustomChange}
                  onCheckedChange={(details) => setUseCustomChange(details.checked === true)}
                >
                  <Checkbox.HiddenInput />
                  <Checkbox.Control>
                    <Checkbox.Indicator />
                  </Checkbox.Control>
                  <Checkbox.Label>
                    <Text fontWeight="medium">使用自定义找零地址</Text>
                  </Checkbox.Label>
                </Checkbox.Root>
                {useCustomChange && (
                  <Input
                    value={changeAddress}
                    onChange={(e) => setChangeAddress(e.target.value)}
                    placeholder="找零地址 (留空则返回发送者地址)"
                    fontFamily="mono"
                    fontSize="sm"
                    mt={2}
                  />
                )}
              </Box>

              <Separator />

              {/* Summary */}
              <Box p={3} bg="bg.muted" borderRadius="md">
                <VStack align="stretch" gap={2} fontSize="sm">
                  <Flex justify="space-between">
                    <Text color="fg.muted">输入总额:</Text>
                    <Text fontWeight="semibold">{satoshiToBTC(getSelectedTotal()).toFixed(8)} BTC</Text>
                  </Flex>
                  <Flex justify="space-between">
                    <Text color="fg.muted">转账金额:</Text>
                    <Text fontWeight="semibold">{amountBTC || '0.00000000'} BTC</Text>
                  </Flex>
                  <Flex justify="space-between">
                    <Text color="fg.muted">找零:</Text>
                    <Text fontWeight="semibold" colorPalette="blue" color="blue.fg">
                      {(satoshiToBTC(getSelectedTotal()) - parseFloat(amountBTC || '0')).toFixed(8)} BTC
                    </Text>
                  </Flex>
                </VStack>
              </Box>

              <Button
                onClick={handleSendTransfer}
                loading={sending}
                disabled={!recipientAddress || !amountBTC || parseFloat(amountBTC) <= 0}
                colorScheme="green"
                size="lg"
              >
                <FiSend />
                发送转账
              </Button>
            </VStack>
          </Card.Body>
        </Card.Root>
      )}

      {/* Transfer Result */}
      {transferResult && (
        <Card.Root
          colorPalette={transferResult.success ? 'green' : 'red'}
          bg={transferResult.success ? 'green.subtle' : 'red.subtle'}
          borderColor={transferResult.success ? 'green.muted' : 'red.muted'}
        >
          <Card.Body>
            <HStack mb={3}>
              {transferResult.success ? <FiCheckCircle size={24} /> : <FiAlertCircle size={24} />}
              <Text fontSize="lg" fontWeight="bold">
                {transferResult.success ? '转账成功' : '转账失败'}
              </Text>
            </HStack>
            <Text mb={2}>{transferResult.message}</Text>
            {transferResult.txid && (
              <Box mt={3} p={2} bg="bg" borderRadius="md">
                <Text fontSize="sm" color="fg.muted" mb={1}>交易ID:</Text>
                <Text fontFamily="mono" fontSize="xs" wordBreak="break-all">
                  {transferResult.txid}
                </Text>
              </Box>
            )}
          </Card.Body>
        </Card.Root>
      )}
    </Box>
  );
}
