// Wallet Manager Component

import { useState } from 'react';
import {
  Box,
  Flex,
  HStack,
  VStack,
  Text,
  Card,
  Input,
  Badge,
  Spinner,
  Button
} from '@chakra-ui/react';
import { FiCreditCard, FiPlus, FiEye, FiEyeOff, FiCopy, FiCheck } from 'react-icons/fi';
import { useGenerateWallet, useWalletBalance } from '../hooks/useBlockchain';
import { useConfig } from '../hooks/useConfig';
import { toaster } from './ui/toaster';

export function WalletManager() {
  const { minerAddress } = useConfig();
  const { wallet, loading: generating, generate } = useGenerateWallet();
  const [currentAddress, setCurrentAddress] = useState('');
  const [showPrivateKey, setShowPrivateKey] = useState(false);
  const [copied, setCopied] = useState<string | null>(null);

  const {
    balance,
    loading: balanceLoading,
    error: balanceError,
    refresh: refreshBalance,
  } = useWalletBalance(currentAddress, minerAddress, true, 10000);

  const copyToClipboard = async (text: string, label: string) => {
    await navigator.clipboard.writeText(text);
    setCopied(label);
    toaster.create({
      title: '已复制',
      description: `${label}已复制到剪贴板`,
      type: 'success',
      duration: 2000,
    });
    setTimeout(() => setCopied(null), 2000);
  };

  const handleGenerate = async () => {
    await generate();
  };

  const handleCheckBalance = () => {
    if (wallet) {
      setCurrentAddress(wallet.address);
    }
  };

  return (
    <Box>
      <HStack gap={3} mb={6}>
        <FiCreditCard size={28} />
        <Text fontSize="2xl" fontWeight="bold">
          钱包管理
        </Text>
      </HStack>

      {/* Generate Wallet */}
      <Card.Root mb={6}>
        <Card.Header>
          <Text fontSize="lg" fontWeight="semibold">
            生成新钱包
          </Text>
        </Card.Header>
        <Card.Body>
          <Button
            onClick={handleGenerate}
            loading={generating}
            size="lg"
            colorScheme="blue"
            width="full"
          >
            <FiPlus />
            生成钱包
          </Button>

          {wallet && (
            <VStack align="stretch" gap={4} mt={4}>
              <Box>
                <Flex justify="space-between" align="center" mb={2}>
                  <Text fontWeight="semibold">钱包地址 (公钥)</Text>
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() => copyToClipboard(wallet.address, '地址')}
                  >
                    {copied === '地址' ? <FiCheck /> : <FiCopy />}
                  </Button>
                </Flex>
                <Box
                  p={3}
                  bg="gray.50"
                  borderRadius="md"
                  fontFamily="mono"
                  fontSize="sm"
                  wordBreak="break-all"
                  _dark={{ bg: 'gray.800' }}
                >
                  {wallet.address}
                </Box>
              </Box>

              <Box>
                <Flex justify="space-between" align="center" mb={2}>
                  <Text fontWeight="semibold" color="red.600" _dark={{ color: 'red.400' }}>
                    私钥 (请妥善保管!)
                  </Text>
                  <HStack>
                    <Button
                      size="sm"
                      variant="ghost"
                      onClick={() => setShowPrivateKey(!showPrivateKey)}
                    >
                      {showPrivateKey ? <FiEyeOff /> : <FiEye />}
                    </Button>
                    <Button
                      size="sm"
                      variant="ghost"
                      onClick={() => copyToClipboard(wallet.private_key, '私钥')}
                    >
                      {copied === '私钥' ? <FiCheck /> : <FiCopy />}
                    </Button>
                  </HStack>
                </Flex>
                <Box
                  p={3}
                  colorPalette="red"
                  bg="red.subtle"
                  borderRadius="md"
                  fontFamily="mono"
                  fontSize="sm"
                  wordBreak="break-all"
                  display={showPrivateKey ? 'block' : 'none'}
                >
                  {wallet.private_key}
                </Box>
                {!showPrivateKey && (
                  <Box
                    p={3}
                    colorPalette="red"
                    bg="red.subtle"
                    borderRadius="md"
                    fontFamily="mono"
                    fontSize="sm"
                    wordBreak="break-all"
                    color="fg.subtle"
                  >
                    ••••••••••••••••••••••••••••••••
                  </Box>
                )}
              </Box>

              <Box>
                <Text fontWeight="semibold" mb={2}>
                  创建时间
                </Text>
                <Text color="fg.muted">
                  {new Date(wallet.created_at).toLocaleString()}
                </Text>
              </Box>

              <Button onClick={handleCheckBalance} colorScheme="green">
                查看余额
              </Button>
            </VStack>
          )}
        </Card.Body>
      </Card.Root>

      {/* Check Balance */}
      <Card.Root mb={6}>
        <Card.Header>
          <Text fontSize="lg" fontWeight="semibold">
            查询钱包余额
          </Text>
        </Card.Header>
        <Card.Body>
          <VStack align="stretch" gap={4}>
            <Box>
              <Text mb={2}>钱包地址</Text>
              <Input
                value={currentAddress}
                onChange={(e) => setCurrentAddress(e.target.value)}
                placeholder="输入钱包地址 (公钥)"
                fontFamily="mono"
              />
            </Box>
            <Button
              onClick={refreshBalance}
              loading={balanceLoading}
              disabled={!currentAddress}
            >
              查询余额
            </Button>
          </VStack>
        </Card.Body>
      </Card.Root>

      {/* Balance Display */}
      {balanceError && (
        <Card.Root colorPalette="red" bg="red.subtle" borderColor="red.muted" mb={6}>
          <Card.Body>
            <Text color="red.fg">错误: {balanceError}</Text>
          </Card.Body>
        </Card.Root>
      )}

      {balance && !balanceError && (
        <Card.Root>
          <Card.Header>
            <Flex justify="space-between" align="center">
              <Text fontSize="lg" fontWeight="semibold">
                钱包余额
              </Text>
              {balanceLoading && <Spinner size="sm" />}
            </Flex>
          </Card.Header>
          <Card.Body>
            <VStack align="stretch" gap={4}>
              <Box>
                <Text fontSize="3xl" fontWeight="bold" colorPalette="green" color="green.fg">
                  {balance.balance_btc.toFixed(8)} BTC
                </Text>
                <Text color="fg.muted" fontSize="sm">
                  ({balance.balance.toLocaleString()} satoshi)
                </Text>
              </Box>

              <Box>
                <Flex justify="space-between" mb={2}>
                  <Text fontWeight="semibold">UTXOs</Text>
                  <Badge>{balance.utxo_count} 个</Badge>
                </Flex>

                {balance.utxo_count === 0 ? (
                  <Text color="fg.subtle">此地址没有未花费的交易输出</Text>
                ) : (
                  <VStack align="stretch" gap={3}>
                    {balance.utxos.map((utxo, index) => (
                      <Box
                        key={`${utxo.txid}-${utxo.out_index}`}
                        p={3}
                        bg="bg.muted"
                        borderRadius="md"
                      >
                        <Flex justify="space-between" mb={2}>
                          <Badge>#{index + 1}</Badge>
                          <Text fontWeight="semibold" colorPalette="green" color="green.fg">
                            {utxo.value_btc.toFixed(8)} BTC
                          </Text>
                        </Flex>
                        <VStack align="stretch" gap={1} fontSize="sm">
                          <Flex justify="space-between">
                            <Text color="fg.muted">交易ID:</Text>
                            <Text fontFamily="mono">
                              {utxo.txid.substring(0, 16)}...
                            </Text>
                          </Flex>
                          <Flex justify="space-between">
                            <Text color="fg.muted">输出索引:</Text>
                            <Text>{utxo.out_index}</Text>
                          </Flex>
                        </VStack>
                      </Box>
                    ))}
                  </VStack>
                )}
              </Box>
            </VStack>
          </Card.Body>
        </Card.Root>
      )}
    </Box>
  );
}
