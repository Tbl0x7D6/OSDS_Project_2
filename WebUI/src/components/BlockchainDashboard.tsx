// Blockchain Status Dashboard Component

import { Box, Flex, HStack, VStack, Text, Badge, Card, Stat, Spinner, Button } from '@chakra-ui/react';
import { FiRefreshCw, FiDatabase, FiCpu, FiActivity, FiHash } from 'react-icons/fi';
import { useBlockchainStatus } from '../hooks/useBlockchain';

// Helper function to get short ID (first 6 characters)
const shortID = (id: string): string => {
  if (!id) return '';
  return id.length <= 6 ? id : id.substring(0, 6);
};

export function BlockchainDashboard() {
  const { status, loading, error, refresh } = useBlockchainStatus(
    'localhost:8001',
    false,
    true,
    5000
  );

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

  if (!status) return null;

  return (
    <Box>
      <Flex justify="space-between" align="center" mb={6}>
        <HStack gap={3}>
          <FiDatabase size={28} />
          <Text fontSize="2xl" fontWeight="bold">
            区块链状态
          </Text>
        </HStack>
        <Button
          onClick={refresh}
          size="sm"
          loading={loading}
          variant="outline"
        >
          <FiRefreshCw />
          刷新
        </Button>
      </Flex>

      {/* Stats Grid */}
      <Box
        display="grid"
        gridTemplateColumns={{ base: '1fr', md: 'repeat(2, 1fr)', lg: 'repeat(4, 1fr)' }}
        gap={4}
        mb={6}
      >
        <Card.Root>
          <Card.Body>
            <Stat.Root>
              <Stat.Label>
                <HStack>
                  <FiDatabase />
                  <Text>链长度</Text>
                </HStack>
              </Stat.Label>
              <Stat.ValueText fontSize="3xl" fontWeight="bold">
                {status.chain_length}
              </Stat.ValueText>
              <Stat.HelpText>区块数量</Stat.HelpText>
            </Stat.Root>
          </Card.Body>
        </Card.Root>

        <Card.Root>
          <Card.Body>
            <Stat.Root>
              <Stat.Label>
                <HStack>
                  <FiCpu />
                  <Text>难度</Text>
                </HStack>
              </Stat.Label>
              <Stat.ValueText fontSize="3xl" fontWeight="bold">
                {status.difficulty}
              </Stat.ValueText>
              <Stat.HelpText>挖矿难度</Stat.HelpText>
            </Stat.Root>
          </Card.Body>
        </Card.Root>

        <Card.Root>
          <Card.Body>
            <Stat.Root>
              <Stat.Label>
                <HStack>
                  <FiActivity />
                  <Text>总交易数</Text>
                </HStack>
              </Stat.Label>
              <Stat.ValueText fontSize="3xl" fontWeight="bold">
                {status.total_transactions}
              </Stat.ValueText>
              <Stat.HelpText>全链交易</Stat.HelpText>
            </Stat.Root>
          </Card.Body>
        </Card.Root>

        <Card.Root>
          <Card.Body>
            <Stat.Root>
              <Stat.Label>
                <HStack>
                  <FiHash />
                  <Text>最新区块</Text>
                </HStack>
              </Stat.Label>
              <Stat.ValueText fontSize="3xl" fontWeight="bold">
                #{status.latest_block_index}
              </Stat.ValueText>
              <Stat.HelpText>区块高度</Stat.HelpText>
            </Stat.Root>
          </Card.Body>
        </Card.Root>
      </Box>

      {/* Latest Block Info */}
      <Card.Root mb={6}>
        <Card.Header>
          <Text fontSize="lg" fontWeight="semibold">
            最新区块信息
          </Text>
        </Card.Header>
        <Card.Body>
          <VStack align="stretch" gap={3}>
            <Flex justify="space-between">
              <Text color="fg.muted">哈希:</Text>
              <Text fontFamily="mono" fontSize="sm">
                {status.latest_block_hash.substring(0, 32)}...
              </Text>
            </Flex>
            <Flex justify="space-between">
              <Text color="fg.muted">矿工:</Text>
              <Badge colorPalette="blue">{shortID(status.latest_block_miner)}</Badge>
            </Flex>
            <Flex justify="space-between">
              <Text color="fg.muted">时间:</Text>
              <Text>{new Date(status.latest_block_time * 1000).toLocaleString()}</Text>
            </Flex>
          </VStack>
        </Card.Body>
      </Card.Root>

      {/* Miner Status */}
      {status.miner_status && (
        <Card.Root>
          <Card.Header>
            <Text fontSize="lg" fontWeight="semibold">
              矿工节点状态
            </Text>
          </Card.Header>
          <Card.Body>
            <Box
              display="grid"
              gridTemplateColumns={{ base: '1fr', md: 'repeat(2, 1fr)' }}
              gap={4}
            >
              <Flex justify="space-between">
                <Text color="fg.muted">节点ID:</Text>
                <Badge>{shortID(status.miner_status.ID)}</Badge>
              </Flex>
              <Flex justify="space-between">
                <Text color="fg.muted">待处理交易:</Text>
                <Badge colorPalette="orange">{status.miner_status.PendingTxs}</Badge>
              </Flex>
              <Flex justify="space-between">
                <Text color="fg.muted">连接节点:</Text>
                <Badge colorPalette="green">{status.miner_status.Peers}</Badge>
              </Flex>
              <Flex justify="space-between">
                <Text color="fg.muted">挖矿状态:</Text>
                <Badge colorPalette={status.miner_status.Mining ? 'green' : 'gray'}>
                  {status.miner_status.Mining ? '🔨 挖矿中' : '⏸️ 空闲'}
                </Badge>
              </Flex>
            </Box>
          </Card.Body>
        </Card.Root>
      )}
    </Box>
  );
}
