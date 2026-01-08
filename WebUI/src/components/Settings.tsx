// Settings Component

import { useState } from 'react';
import {
  Box,
  Flex,
  HStack,
  VStack,
  Text,
  Card,
  Input,
  Button
} from '@chakra-ui/react';
import { FiSettings, FiServer } from 'react-icons/fi';
import { toaster } from './ui/toaster';
import { useConfig } from '../hooks/useConfig';

export function Settings() {
  const config = useConfig();
  const [minerAddress, setMinerAddress] = useState(config.minerAddress);
  const [apiUrl, setApiUrl] = useState(config.apiUrl);

  const handleSave = () => {
    // 使用 useConfig hook 保存配置
    config.updateConfig({ minerAddress, apiUrl });
    
    toaster.create({
      title: '设置已保存',
      description: '配置已成功保存',
      type: 'success',
      duration: 3000,
    });
  };

  return (
    <Box>
      <HStack gap={3} mb={6}>
        <FiSettings size={28} />
        <Text fontSize="2xl" fontWeight="bold">
          设置
        </Text>
      </HStack>

      <VStack align="stretch" gap={6}>
        <Card.Root>
          <Card.Header>
            <Flex align="center" gap={2}>
              <FiServer />
              <Text fontSize="lg" fontWeight="semibold">
                网络配置
              </Text>
            </Flex>
          </Card.Header>
          <Card.Body>
            <VStack align="stretch" gap={4}>
              <Box>
                <Text fontWeight="semibold" mb={2}>
                  矿工节点地址
                </Text>
                <Input
                  value={minerAddress}
                  onChange={(e) => setMinerAddress(e.target.value)}
                  placeholder="localhost:8001"
                />
                <Text fontSize="sm" color="fg.muted" mt={1}>
                  用于查询区块链状态和提交交易
                </Text>
              </Box>

              <Box>
                <Text fontWeight="semibold" mb={2}>
                  API 服务地址
                </Text>
                <Input
                  value={apiUrl}
                  onChange={(e) => setApiUrl(e.target.value)}
                  placeholder="http://localhost:3000/api"
                />
                <Text fontSize="sm" color="fg.muted" mt={1}>
                  后端 API 服务的地址
                </Text>
              </Box>

              <Button onClick={handleSave} colorScheme="blue">
                保存设置
              </Button>
            </VStack>
          </Card.Body>
        </Card.Root>

        <Card.Root>
          <Card.Header>
            <Text fontSize="lg" fontWeight="semibold">
              关于
            </Text>
          </Card.Header>
          <Card.Body>
            <VStack align="stretch" gap={2}>
              <Flex justify="space-between">
                <Text color="fg.muted">应用名称:</Text>
                <Text fontWeight="semibold">Bitcoin Visualizer</Text>
              </Flex>
              <Flex justify="space-between">
                <Text color="fg.muted">版本:</Text>
                <Text>1.0.0</Text>
              </Flex>
              <Flex justify="space-between">
                <Text color="fg.muted">技术栈:</Text>
                <Text>React + Chakra UI + Vite</Text>
              </Flex>
            </VStack>
          </Card.Body>
        </Card.Root>
      </VStack>
    </Box>
  );
}
