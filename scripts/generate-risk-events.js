const { Web3 } = require('web3');

const web3 = new Web3('http://localhost:8545');

// 简单的重入攻击合约 ABI
const VULNERABLE_CONTRACT_ABI = [
  {
    "inputs": [],
    "name": "withdraw",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [],
    "name": "deposit",
    "outputs": [],
    "stateMutability": "payable",
    "type": "function"
  }
];

// 简单的 ERC20 转账 ABI
const ERC20_ABI = [
  {
    "inputs": [{"name": "to", "type": "address"}, {"name": "amount", "type": "uint256"}],
    "name": "transfer",
    "outputs": [{"name": "", "type": "bool"}],
    "stateMutability": "nonpayable",
    "type": "function"
  }
];

async function generateRiskEvents() {
  console.log('🚀 开始生成风险事件...\n');

  const accounts = await web3.eth.getAccounts();
  const sender = accounts[0];
  const receiver = accounts[1];

  console.log(`发送账户: ${sender}`);
  console.log(`接收账户: ${receiver}\n`);

  let eventCount = 0;

  setInterval(async () => {
    try {
      const rand = Math.random();

      if (rand < 0.6) {
        // 60% - 普通转账（中危）
        const amount = web3.utils.toWei('0.1', 'ether');
        const tx = await web3.eth.sendTransaction({
          from: sender,
          to: receiver,
          value: amount,
          gas: 21000
        });
        console.log(`✅ [中危] 普通转账: ${tx.transactionHash}`);
        eventCount++;

      } else if (rand < 0.9) {
        // 30% - 大额转账（高危）
        const amount = web3.utils.toWei('10', 'ether');
        const tx = await web3.eth.sendTransaction({
          from: sender,
          to: receiver,
          value: amount,
          gas: 21000
        });
        console.log(`⚠️  [高危] 大额转账: ${tx.transactionHash}`);
        eventCount++;

      } else {
        // 10% - 高 Gas 消耗（严重）
        const tx = await web3.eth.sendTransaction({
          from: sender,
          to: receiver,
          value: web3.utils.toWei('0.1', 'ether'),
          gas: 500000, // 高 Gas
          data: '0x' + '00'.repeat(1000) // 大量数据
        });
        console.log(`🔴 [严重] 高Gas消耗: ${tx.transactionHash}`);
        eventCount++;
      }

      console.log(`📊 已生成事件: ${eventCount}\n`);

    } catch (error) {
      console.error('❌ 错误:', error.message);
    }
  }, 5000); // 每5秒生成一个事件
}

// 启动
generateRiskEvents().catch(console.error);

console.log('⏰ 每5秒生成一个风险事件...');
console.log('按 Ctrl+C 停止\n');
