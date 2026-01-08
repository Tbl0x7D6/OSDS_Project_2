// Block Explorer Component - 类似 mempool.space 风格

import { useState, useRef, useMemo } from 'react';
import {
  Box,
  Flex,
  HStack,
  VStack,
  Text,
  Card,
  Badge,
  Spinner,
  Button,
  Grid,
  Separator,
} from '@chakra-ui/react';
import {
  FiPackage,
  FiRefreshCw,
  FiChevronLeft,
  FiChevronRight,
  FiUser,
  FiLayers,
  FiActivity,
  FiHash,
} from 'react-icons/fi';
import { useBlockchainStatus } from '../hooks/useBlockchain';
import { useConfig } from '../hooks/useConfig';
import type { BlockOutput, TransactionOutput } from '../types/blockchain';

// Helper function to get short ID (first 6 characters)
const shortID = (id: string): string => {
  if (!id) return '';
  return id.length <= 6 ? id : id.substring(0, 6);
};

// 格式化时间
const formatTimeAgo = (timestamp: number): string => {
  const now = Date.now();
  const diff = now - timestamp * 1000;
  const minutes = Math.floor(diff / 60000);
  if (minutes < 1) return '刚刚';
  if (minutes < 60) return `${minutes}分钟前`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}小时前`;
  const days = Math.floor(hours / 24);
  return `${days}天前`;
};

export function BlockExplorer() {
  const { minerAddress } = useConfig();
  const { status, loading, error, refresh } = useBlockchainStatus(
    minerAddress,
    true, // 包含详细信息
    false
  );

  const [selectedBlockIndex, setSelectedBlockIndex] = useState<number | null>(null);
  const scrollContainerRef = useRef<HTMLDivElement>(null);

  // 滚动控制
  const scrollLeft = () => {
    if (scrollContainerRef.current) {
      scrollContainerRef.current.scrollBy({ left: -200, behavior: 'smooth' });
    }
  };

  const scrollRight = () => {
    if (scrollContainerRef.current) {
      scrollContainerRef.current.scrollBy({ left: 200, behavior: 'smooth' });
    }
  };

  // 处理区块数据
  const statusBlocks = status?.blocks;
  const blocks = useMemo(() => {
    if (!statusBlocks) return [];
    return [...statusBlocks].reverse(); // 最新的在前
  }, [statusBlocks]);

  // 当前选中的区块（默认选中最新区块）
  const effectiveSelectedIndex = selectedBlockIndex ?? (blocks.length > 0 ? blocks[0].index : null);
  const selectedBlock = blocks.find((b) => b.index === effectiveSelectedIndex);

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

  return (
    <Box>
      {/* 顶部标题 */}
      <Flex justify="space-between" align="center" mb={6}>
        <HStack gap={3}>
          <Box bg="bg.emphasized" p={2} borderRadius="lg" color="fg">
            <FiPackage size={24} />
          </Box>
          <VStack align="start" gap={0}>
            <Text fontSize="2xl" fontWeight="bold">
              区块浏览器
            </Text>
            <Text fontSize="sm" color="fg.muted">
              共 {blocks.length} 个区块
            </Text>
          </VStack>
        </HStack>
        <Button onClick={refresh} size="sm" loading={loading} variant="outline">
          <FiRefreshCw />
          刷新
        </Button>
      </Flex>

      {/* 横向滚动的区块列表 */}
      <Card.Root mb={6} bg="bg.emphasized" overflow="hidden">
        <Card.Body p={0}>
          <Box position="relative">
            {/* 左侧滚动按钮 */}
            <Button
              position="absolute"
              left={2}
              top="50%"
              transform="translateY(-50%)"
              zIndex={10}
              size="sm"
              variant="subtle"
              onClick={scrollLeft}
              borderRadius="full"
            >
              <FiChevronLeft />
            </Button>

            {/* 右侧滚动按钮 */}
            <Button
              position="absolute"
              right={2}
              top="50%"
              transform="translateY(-50%)"
              zIndex={10}
              size="sm"
              variant="subtle"
              onClick={scrollRight}
              borderRadius="full"
            >
              <FiChevronRight />
            </Button>

            {/* 区块滚动容器 */}
            <Box
              ref={scrollContainerRef}
              overflowX="auto"
              py={6}
              px={12}
              css={{
                scrollbarWidth: 'thin',
                scrollbarColor: 'var(--chakra-colors-border) transparent',
                '&::-webkit-scrollbar': {
                  height: '8px',
                },
                '&::-webkit-scrollbar-track': {
                  background: 'transparent',
                },
                '&::-webkit-scrollbar-thumb': {
                  background: 'var(--chakra-colors-border)',
                  borderRadius: '4px',
                },
              }}
            >
              <Flex gap={4} flexWrap="nowrap">
                {blocks.map((block) => (
                  <Block3DCard
                    key={block.index}
                    block={block}
                    isSelected={effectiveSelectedIndex === block.index}
                    onClick={() => setSelectedBlockIndex(block.index)}
                  />
                ))}
              </Flex>
            </Box>
          </Box>
        </Card.Body>
      </Card.Root>

      {/* 选中区块的详细信息 */}
      {selectedBlock && (
        <BlockDetailPanel block={selectedBlock} />
      )}
    </Box>
  );
}

// 3D 区块卡片组件
interface Block3DCardProps {
  block: BlockOutput;
  isSelected: boolean;
  onClick: () => void;
}

function Block3DCard({ block, isSelected, onClick }: Block3DCardProps) {
  const totalBTC = block.transactions.reduce((sum, tx) => {
    return sum + tx.outputs.reduce((s, o) => s + o.value, 0);
  }, 0) / 100000000;

  return (
    <Box
      onClick={onClick}
      cursor="pointer"
      flexShrink={0}
      transform={isSelected ? 'scale(1.05)' : 'scale(1)'}
      transition="all 0.2s ease"
      _hover={{ transform: 'scale(1.05)' }}
    >
      {/* 3D 区块容器 */}
      <Box position="relative" w="140px" h="160px">
        {/* 区块主体 - 顶部 */}
        <Box
          position="absolute"
          top={0}
          left={0}
          w="140px"
          h="120px"
          bg={isSelected ? 'colorPalette.solid' : 'bg.subtle'}
          borderRadius="lg"
          boxShadow={isSelected ? 'md' : 'sm'}
          border="1px solid"
          borderColor={isSelected ? 'colorPalette.focusRing' : 'border'}
          display="flex"
          flexDirection="column"
          justifyContent="space-between"
          p={3}
          color={isSelected ? 'colorPalette.contrast' : 'fg'}
          overflow="hidden"
          data-colorpalette="blue"
        >
          {/* 费用信息 */}
          <Box>
            <Text fontSize="xs" color={isSelected ? 'colorPalette.contrast' : 'fg.muted'}>
              ~{block.difficulty} bits
            </Text>
          </Box>

          {/* BTC 总量 */}
          <Box>
            <Text fontSize="lg" fontWeight="bold">
              {totalBTC.toFixed(3)} BTC
            </Text>
            <Text fontSize="xs" color={isSelected ? 'colorPalette.contrast' : 'fg.muted'}>
              {block.transactions.length} 笔交易
            </Text>
          </Box>

          {/* 出块时间 */}
          <Text fontSize="xs" color={isSelected ? 'colorPalette.contrast' : 'fg.muted'}>
            {formatTimeAgo(block.timestamp)}
          </Text>
        </Box>

      </Box>

      {/* 区块高度标签 */}
      <Text
        textAlign="center"
        mt={2}
        fontWeight="bold"
        color={isSelected ? 'fg' : 'fg.muted'}
        fontSize="lg"
      >
        #{block.index}
      </Text>

      {/* 矿工标识 */}
      <HStack justify="center" mt={1}>
        <Box color="fg.muted"><FiUser size={12} /></Box>
        <Text fontSize="xs" color="fg.muted">
          {shortID(block.miner_id) || 'Unknown'}
        </Text>
      </HStack>
    </Box>
  );
}

// 区块详情面板
interface BlockDetailPanelProps {
  block: BlockOutput;
}

function BlockDetailPanel({ block }: BlockDetailPanelProps) {
  const totalBTC = block.transactions.reduce((sum, tx) => {
    return sum + tx.outputs.reduce((s, o) => s + o.value, 0);
  }, 0) / 100000000;

  return (
    <Card.Root>
      <Card.Header>
        <Flex justify="space-between" align="center">
          <HStack gap={3}>
            <Badge colorPalette="blue" fontSize="lg" px={4} py={2}>
              区块 #{block.index}
            </Badge>
            {block.index === 0 && (
              <Badge colorPalette="teal">创世区块</Badge>
            )}
          </HStack>
          <Text color="fg.muted" fontSize="sm">
            {new Date(block.timestamp * 1000).toLocaleString()}
          </Text>
        </Flex>
      </Card.Header>

      <Card.Body>
        {/* 区块统计信息 */}
        <Grid templateColumns="repeat(4, 1fr)" gap={4} mb={6}>
          <StatCard
            icon={<FiHash />}
            label="Nonce"
            value={block.nonce.toString()}
          />
          <StatCard
            icon={<FiActivity />}
            label="难度"
            value={`${block.difficulty} bits`}
            color="orange"
          />
          <StatCard
            icon={<FiLayers />}
            label="交易数"
            value={block.transactions.length.toString()}
            color="green"
          />
          <StatCard
            icon={<FiPackage />}
            label="总金额"
            value={`${totalBTC.toFixed(4)} BTC`}
            color="blue"
          />
        </Grid>

        <Separator mb={6} />

        {/* 哈希信息 */}
        <VStack align="stretch" gap={4} mb={6}>
          <Box>
            <Text fontWeight="semibold" mb={2} color="fg.muted" fontSize="sm">
              区块哈希
            </Text>
            <Box
              p={3}
              bg="bg.muted"
              borderRadius="md"
              fontFamily="mono"
              fontSize="sm"
              wordBreak="break-all"
            >
              {block.hash}
            </Box>
          </Box>

          {block.index > 0 && (
            <Box>
              <Text fontWeight="semibold" mb={2} color="fg.muted" fontSize="sm">
                前一区块哈希
              </Text>
              <Box
                p={3}
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

          <Box>
            <Text fontWeight="semibold" mb={2} color="fg.muted" fontSize="sm">
              矿工 ID
            </Text>
            <Box
              p={3}
              bg="bg.muted"
              borderRadius="md"
              fontFamily="mono"
              fontSize="sm"
              wordBreak="break-all"
            >
              {block.miner_id || '未知'}
            </Box>
          </Box>
        </VStack>

        <Separator mb={6} />

        {/* 交易列表 */}
        <Box>
          <Text fontWeight="semibold" mb={4} fontSize="lg">
            交易列表 ({block.transactions.length})
          </Text>
          <VStack align="stretch" gap={3}>
            {block.transactions.map((tx, index) => (
              <TransactionCard key={tx.id} tx={tx} index={index} />
            ))}
          </VStack>
        </Box>
      </Card.Body>
    </Card.Root>
  );
}

// 统计卡片组件
interface StatCardProps {
  icon: React.ReactNode;
  label: string;
  value: string;
  color?: string;
}

function StatCard({ icon, label, value, color = 'gray' }: StatCardProps) {
  return (
    <Box
      p={4}
      bg="bg.muted"
      borderRadius="lg"
      borderLeft="4px solid"
      borderLeftColor="border.emphasized"
      data-colorpalette={color}
    >
      <HStack color="fg.muted" mb={2}>
        {icon}
        <Text fontSize="sm" color="fg.muted">
          {label}
        </Text>
      </HStack>
      <Text fontSize="lg" fontWeight="bold" color="fg">
        {value}
      </Text>
    </Box>
  );
}
// 交易卡片组件
interface TransactionCardProps {
  tx: TransactionOutput;
  index: number;
}

function TransactionCard({ tx, index }: TransactionCardProps) {
  const totalOutput = tx.outputs.reduce((sum, o) => sum + o.value, 0) / 100000000;

  return (
    <Box
      p={4}
      bg="bg.muted"
      borderRadius="lg"
      border="1px solid"
      borderColor="border"
    >
      <Flex justify="space-between" align="center" mb={3}>
        <HStack>
          <Badge colorPalette="blue" variant="subtle">
            TX #{index}
          </Badge>
          {tx.is_coinbase && (
            <Badge colorPalette="teal" variant="solid">
              Coinbase
            </Badge>
          )}
        </HStack>
        <Text fontSize="sm" fontWeight="bold" color="fg.success">
          {totalOutput.toFixed(8)} BTC
        </Text>
      </Flex>

      <Box mb={3}>
        <Text fontSize="xs" color="fg.muted" mb={1}>
          交易 ID
        </Text>
        <Text fontFamily="mono" fontSize="xs" wordBreak="break-all">
          {tx.id}
        </Text>
      </Box>

      <Grid templateColumns="1fr auto 1fr" gap={4} alignItems="start">
        {/* 输入 */}
        <Box>
          <Text fontSize="xs" fontWeight="semibold" color="fg.muted" mb={2}>
            输入 ({tx.inputs.length})
          </Text>
          {tx.is_coinbase ? (
            <Box p={2} bg="bg.success" borderRadius="md">
              <Text fontSize="xs" color="fg.success">
                区块奖励
              </Text>
            </Box>
          ) : (
            <VStack align="stretch" gap={1}>
              {tx.inputs.map((input, i) => (
                <Box key={i} p={2} bg="bg" borderRadius="md" fontSize="xs">
                  <Text fontFamily="mono" truncate>
                    {shortID(input.txid)}:{input.out_index}
                  </Text>
                </Box>
              ))}
            </VStack>
          )}
        </Box>

        {/* 箭头 */}
        <Flex align="center" h="full" pt={6}>
          <Text fontSize="xl" color="fg.muted">
            →
          </Text>
        </Flex>

        {/* 输出 */}
        <Box>
          <Text fontSize="xs" fontWeight="semibold" color="fg.muted" mb={2}>
            输出 ({tx.outputs.length})
          </Text>
          <VStack align="stretch" gap={1}>
            {tx.outputs.map((output, i) => (
              <Box key={i} p={2} bg="bg" borderRadius="md" fontSize="xs">
                <Text fontFamily="mono" truncate mb={1}>
                  {shortID(output.scriptpubkey)}
                </Text>
                <Text fontWeight="bold" color="fg.success">
                  {(output.value / 100000000).toFixed(8)} BTC
                </Text>
              </Box>
            ))}
          </VStack>
        </Box>
      </Grid>
    </Box>
  );
}
