import { useState } from 'react';
import {
  Box,
  Container,
  Flex,
  HStack,
  VStack,
  Text,
  Tabs,
  Badge,
} from '@chakra-ui/react';
import {
  FiDatabase,
  FiCreditCard,
  FiPackage,
  FiSettings,
  FiActivity,
} from 'react-icons/fi';
import { BlockchainDashboard } from './components/BlockchainDashboard';
import { WalletManager } from './components/WalletManager';
import { BlockExplorer } from './components/BlockExplorer';
import { Settings } from './components/Settings';
import { ColorModeButton } from './components/ui/color-mode';

function App() {
  const [activeTab, setActiveTab] = useState('dashboard');

  return (
    <Box minH="100vh" bg="bg.subtle">
      {/* Header */}
      <Box
        bg="bg"
        borderBottom="1px"
        borderColor="border"
        py={4}
        position="sticky"
        top={0}
        zIndex={10}
        shadow="sm"
      >
        <Container maxW="7xl">
          <Flex justify="space-between" align="center">
            <HStack gap={3}>
              <Box
                bg="blue.solid"
                p={2}
                borderRadius="lg"
                color="white"
              >
                <FiActivity size={24} />
              </Box>
              <VStack align="start" gap={0}>
                <Text fontSize="xl" fontWeight="bold">
                  Bitcoin Visualizer
                </Text>
                <Text fontSize="sm" color="fg.muted">
                  区块链可视化系统
                </Text>
              </VStack>
            </HStack>
            <HStack>
              <Badge colorScheme="green" fontSize="sm" px={3} py={1}>
                在线
              </Badge>
              <ColorModeButton />
            </HStack>
          </Flex>
        </Container>
      </Box>

      {/* Main Content */}
      <Container maxW="7xl" py={8}>
        <Tabs.Root
          value={activeTab}
          onValueChange={(e) => setActiveTab(e.value)}
          variant="enclosed"
        >
          <Tabs.List mb={6}>
            <Tabs.Trigger value="dashboard">
              <HStack gap={2}>
                <FiDatabase />
                <Text>区块链状态</Text>
              </HStack>
            </Tabs.Trigger>
            <Tabs.Trigger value="wallet">
              <HStack gap={2}>
                <FiCreditCard />
                <Text>钱包管理</Text>
              </HStack>
            </Tabs.Trigger>
            <Tabs.Trigger value="explorer">
              <HStack gap={2}>
                <FiPackage />
                <Text>区块浏览器</Text>
              </HStack>
            </Tabs.Trigger>
            <Tabs.Trigger value="settings">
              <HStack gap={2}>
                <FiSettings />
                <Text>设置</Text>
              </HStack>
            </Tabs.Trigger>
          </Tabs.List>

          <Tabs.Content value="dashboard">
            <BlockchainDashboard />
          </Tabs.Content>

          <Tabs.Content value="wallet">
            <WalletManager />
          </Tabs.Content>

          <Tabs.Content value="explorer">
            <BlockExplorer />
          </Tabs.Content>

          <Tabs.Content value="settings">
            <Settings />
          </Tabs.Content>
        </Tabs.Root>
      </Container>

      {/* Footer */}
      <Box
        as="footer"
        bg="bg"
        borderTop="1px"
        borderColor="border"
        py={6}
        mt={12}
      >
        <Container maxW="7xl">
          <Flex justify="space-between" align="center" fontSize="sm" color="fg.muted">
            <Text>
              © 2026 Bitcoin Visualizer. Built with React + Chakra UI.
            </Text>
            <Text>
              Powered by Go Blockchain
            </Text>
          </Flex>
        </Container>
      </Box>
    </Box>
  );
}

export default App;
