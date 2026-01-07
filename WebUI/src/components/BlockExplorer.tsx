// Block Explorer Component

import { useState } from 'react';
import {
  Box,
  Flex,
  HStack,
  VStack,
  Text,
  Card,
  Badge,
  Spinner,
  Button
} from '@chakra-ui/react';
import { FiPackage, FiRefreshCw, FiClock, FiHash } from 'react-icons/fi';
import { useBlockchainStatus } from '../hooks/useBlockchain';
import type { BlockOutput, TransactionOutput } from '../types/blockchain';

export function BlockExplorer() {
  const { status, loading, error, refresh } = useBlockchainStatus(
    'localhost:8001',
    true, // 包含详细信息
    false
  );

  const [expandedBlocks, setExpandedBlocks] = useState<Set<number>>(new Set());

  const toggleBlock = (index: number) => {
    const newExpanded = new Set(expandedBlocks);
    if (newExpanded.has(index)) {
      newExpanded.delete(index);
    } else {
      newExpanded.add(index);
    }
    setExpandedBlocks(newExpanded);
  };

  if (loading && !status) {
    return (
      <Flex justify="center" align="center" minH="200px">
        <Spinner size="xl" colorPalette="blue" />
      </Flex>
    );
  }

  if (error) {
    return (
      <Card.Root colorPalette="red" bg="red.subtle" borderColor="red.muted">
        <Card.Body>
          <Text color="red.fg">错误: {error}</Text>
          <Button onClick={refresh} mt={2} size="sm">
            重试
          </Button>
        </Card.Body>
      </Card.Root>
    );
  }

  if (!status || !status.blocks) {
    return (
      <Box>
        <Button onClick={refresh} loading={loading}>
          <FiRefreshCw />
          加载区块
        </Button>
      </Box>
    );
  }

  const blocks = [...status.blocks].reverse(); // 最新的在前

  return (
    <Box>
      <Flex justify="space-between" align="center" mb={6}>
        <HStack gap={3}>
          <FiPackage size={28} />
          <Text fontSize="2xl" fontWeight="bold">
            区块浏览器
          </Text>
        </HStack>
        <Button onClick={refresh} size="sm" loading={loading} variant="outline">
          <FiRefreshCw />
          刷新
        </Button>
      </Flex>

      <Text mb={4} color="fg.muted">
        总共 {blocks.length} 个区块
      </Text>

      <VStack align="stretch" gap={4}>
        {blocks.map((block) => (
          <BlockCard
            key={block.index}
            block={block}
            expanded={expandedBlocks.has(block.index)}
            onToggle={() => toggleBlock(block.index)}
          />
        ))}
      </VStack>
    </Box>
  );
}

interface BlockCardProps {
  block: BlockOutput;
  expanded: boolean;
  onToggle: () => void;
}

function BlockCard({ block, expanded, onToggle }: BlockCardProps) {
  return (
    <Card.Root>
      <Card.Body>
        <Flex justify="space-between" align="start" mb={3}>
          <HStack gap={3}>
            <Badge colorScheme="blue" fontSize="lg" px={3} py={1}>
              区块 #{block.index}
            </Badge>
            {block.index === 0 && (
              <Badge colorScheme="purple">创世区块</Badge>
            )}
          </HStack>
          <Button size="sm" variant="ghost" onClick={onToggle}>
            {expanded ? '收起' : '展开'}
          </Button>
        </Flex>

        <VStack align="stretch" gap={2}>
          <Flex justify="space-between">
            <HStack>
              <FiHash size={16} />
              <Text color="fg.muted">哈希:</Text>
            </HStack>
            <Text fontFamily="mono" fontSize="sm">
              {block.hash.substring(0, 32)}...
            </Text>
          </Flex>

          <Flex justify="space-between">
            <HStack>
              <FiClock size={16} />
              <Text color="fg.muted">时间:</Text>
            </HStack>
            <Text fontSize="sm">
              {new Date(block.timestamp * 1000).toLocaleString()}
            </Text>
          </Flex>

          <Flex justify="space-between">
            <Text color="fg.muted">矿工:</Text>
            <Badge>{block.miner_id}</Badge>
          </Flex>

          <Flex justify="space-between">
            <Text color="fg.muted">Nonce:</Text>
            <Text fontFamily="mono">{block.nonce}</Text>
          </Flex>

          <Flex justify="space-between">
            <Text color="fg.muted">难度:</Text>
            <Badge colorPalette="orange">{block.difficulty}</Badge>
          </Flex>

          <Flex justify="space-between">
            <Text color="fg.muted">交易数:</Text>
            <Badge colorPalette="green">{block.transactions.length}</Badge>
          </Flex>

          {expanded && (
            <Box mt={4}>
              <Text fontWeight="semibold" mb={3}>
                交易详情
              </Text>
              <VStack align="stretch" gap={3}>
                {block.transactions.map((tx, index) => (
                  <TransactionCard key={tx.id} tx={tx} index={index} />
                ))}
              </VStack>

              {block.index > 0 && (
                <Box mt={4}>
                  <Text fontWeight="semibold" mb={2}>
                    前一区块哈希
                  </Text>
                  <Box
                    p={2}
                    bg="bg.muted"
                    borderRadius="md"
                    fontFamily="mono"
                    fontSize="sm"
                    wordBreak="break-all"
                  >
                    {block.prev_hash}
                  </Box>
                </Box>
              )}

              <Box mt={4}>
                <Text fontWeight="semibold" mb={2}>
                  完整哈希
                </Text>
                <Box
                  p={2}
                  bg="bg.muted"
                  borderRadius="md"
                  fontFamily="mono"
                  fontSize="sm"
                  wordBreak="break-all"
                >
                  {block.hash}
                </Box>
              </Box>
            </Box>
          )}
        </VStack>
      </Card.Body>
    </Card.Root>
  );
}

interface TransactionCardProps {
  tx: TransactionOutput;
  index: number;
}

function TransactionCard({ tx, index }: TransactionCardProps) {
  return (
    <Box p={3} bg="bg.muted" borderRadius="md">
      <Flex justify="space-between" align="center" mb={2}>
        <HStack>
          <Badge colorPalette="purple">TX #{index}</Badge>
          {tx.is_coinbase && <Badge colorPalette="green">Coinbase</Badge>}
        </HStack>
      </Flex>

      <VStack align="stretch" gap={2} fontSize="sm">
        <Box>
          <Text fontWeight="semibold" mb={1}>
            交易ID:
          </Text>
          <Text fontFamily="mono" fontSize="xs" wordBreak="break-all">
            {tx.id}
          </Text>
        </Box>

        {!tx.is_coinbase && (
          <Box>
            <Text fontWeight="semibold" mb={1}>
              输入 ({tx.inputs.length}):
            </Text>
            {tx.inputs.map((input, i) => (
              <Box key={i} pl={3} fontSize="xs">
                <Text>• {input.txid.substring(0, 16)}... [{input.out_index}]</Text>
              </Box>
            ))}
          </Box>
        )}

        <Box>
          <Text fontWeight="semibold" mb={1}>
            输出 ({tx.outputs.length}):
          </Text>
          {tx.outputs.map((output, i) => (
            <Box key={i} pl={3} fontSize="xs">
              <Flex justify="space-between">
                <Text>• {output.scriptpubkey.substring(0, 20)}...</Text>
                <Text colorPalette="green" color="green.fg" fontWeight="semibold">
                  {(output.value / 100000000).toFixed(8)} BTC
                </Text>
              </Flex>
            </Box>
          ))}
        </Box>
      </VStack>
    </Box>
  );
}
