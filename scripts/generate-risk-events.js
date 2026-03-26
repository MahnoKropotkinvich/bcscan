const { Web3 } = require('web3');

const web3 = new Web3('http://localhost:8545');

// ==================== 风险事件生成器 ====================

async function main() {
  console.log('=== BCScan 风险事件生成器 (v2 - 含特权攻击场景) ===\n');

  const accounts = await web3.eth.getAccounts();
  if (accounts.length < 4) {
    console.error('需要至少 4 个账户，请确认 Ganache 配置');
    process.exit(1);
  }

  console.log('可用账户:');
  for (let i = 0; i < Math.min(accounts.length, 6); i++) {
    const bal = await web3.eth.getBalance(accounts[i]);
    console.log(`  [${i}] ${accounts[i]}  ${web3.utils.fromWei(bal, 'ether').substring(0, 10)} ETH`);
  }
  console.log();

  let eventCount = 0;
  let deployedContracts = []; // 跟踪已部署的 ownable 合约

  // ==================== 工具函数 ====================

  async function pickSenderWithBalance(minWei) {
    const shuffled = [...accounts].sort(() => Math.random() - 0.5);
    for (const acct of shuffled) {
      const bal = BigInt(await web3.eth.getBalance(acct));
      const reserve = BigInt(web3.utils.toWei('0.1', 'ether'));
      if (bal - reserve >= BigInt(minWei)) {
        return acct;
      }
    }
    return null;
  }

  async function safeSend(params) {
    try {
      const tx = await web3.eth.sendTransaction(params);
      eventCount++;
      return tx;
    } catch (err) {
      if (err.message && err.message.includes('insufficient balance')) {
        console.log(`  ⚠ 余额不足，跳过`);
      } else {
        // 交易 revert 也算一笔交易（会被 RMS 捕获）
        console.log(`  ⚠ 交易失败/revert: ${err.message?.substring(0, 80)}`);
        eventCount++; // revert 的交易也计数
      }
      return null;
    }
  }

  // ==================== 场景生成函数 ====================

  // 场景1: 普通 ETH 转账（低风险，作为背景噪音）
  async function normalTransfer() {
    const amount = web3.utils.toWei(String(0.01 + Math.random() * 0.09), 'ether');
    const from = await pickSenderWithBalance(amount);
    if (!from) { console.log('  跳过：无可用发送方'); return; }

    let to;
    do { to = accounts[Math.floor(Math.random() * Math.min(accounts.length, 6))]; } while (to === from);

    const tx = await safeSend({ from, to, value: amount, gas: 21000 });
    if (tx) {
      console.log(`[低危] 普通转账 ${web3.utils.fromWei(amount, 'ether').substring(0, 6)} ETH: ${tx.transactionHash}`);
    }
  }

  // 场景2: 大额 ETH 转账（高风险）
  async function largeTransfer() {
    const ethAmount = 5 + Math.random() * 10;
    const amount = web3.utils.toWei(String(ethAmount), 'ether');
    const from = await pickSenderWithBalance(amount);
    if (!from) { console.log('  跳过：无可用发送方'); return; }

    let to;
    do { to = accounts[Math.floor(Math.random() * Math.min(accounts.length, 6))]; } while (to === from);

    const tx = await safeSend({ from, to, value: amount, gas: 21000 });
    if (tx) {
      console.log(`[高危] 大额转账 ${ethAmount.toFixed(2)} ETH: ${from.substring(0, 10)} → ${to.substring(0, 10)} ${tx.transactionHash}`);
      const returnAmount = web3.utils.toWei(String(ethAmount * 0.8), 'ether');
      await safeSend({ from: to, to: from, value: returnAmount, gas: 21000 });
      console.log(`  ↩ 回转 ${(ethAmount * 0.8).toFixed(2)} ETH`);
    }
  }

  // 场景3: 高 Gas 消耗交易
  async function highGasTransaction() {
    const from = await pickSenderWithBalance(web3.utils.toWei('0.1', 'ether'));
    if (!from) { console.log('  跳过：无可用发送方'); return; }

    let to;
    do { to = accounts[Math.floor(Math.random() * Math.min(accounts.length, 6))]; } while (to === from);

    const paddingSize = 2000 + Math.floor(Math.random() * 8000);
    const tx = await safeSend({
      from, to,
      value: web3.utils.toWei('0.01', 'ether'),
      gas: 500000 + Math.floor(Math.random() * 500000),
      data: '0x' + 'ff'.repeat(paddingSize)
    });
    if (tx) {
      console.log(`[中危] 高Gas消耗 (data=${paddingSize * 2}B): ${tx.transactionHash}`);
    }
  }

  // 场景4: 快速连续转账
  async function rapidBurstTransfers() {
    const ethAmount = 1 + Math.random() * 4;
    const from = await pickSenderWithBalance(web3.utils.toWei(String(ethAmount * 5 + 1), 'ether'));
    if (!from) { console.log('  跳过：无可用发送方'); return; }

    let to;
    do { to = accounts[Math.floor(Math.random() * Math.min(accounts.length, 6))]; } while (to === from);

    console.log(`[严重] 开始快速连续转账（5轮来回）...`);
    for (let i = 0; i < 5; i++) {
      const roundAmount = web3.utils.toWei(String(ethAmount * (0.8 + Math.random() * 0.4)), 'ether');
      const tx1 = await safeSend({ from, to, value: roundAmount, gas: 21000 });
      if (!tx1) break;
      const tx2 = await safeSend({ from: to, to: from, value: roundAmount, gas: 21000 });
      if (!tx2) break;
    }
    console.log(`[严重] 快速连续转账完成`);
  }

  // 场景5: 部署合约（模拟特权合约，后续攻击场景的基础）
  // 合约 runtime code = STOP (0x00)：接受所有调用并成功返回
  // 这样我们可以向它发送任何 function selector 并被 RMS 捕获
  async function deployOwnableContract() {
    const from = await pickSenderWithBalance(web3.utils.toWei('3', 'ether'));
    if (!from) { console.log('  跳过：无可用发送方'); return; }

    try {
      // Init code: 把 runtime code (STOP) 复制到内存并返回
      const initCode = '0x600180600c5f395ff300';
      const tx = await web3.eth.sendTransaction({
        from,
        data: initCode,
        gas: 500000
      });
      const addr = tx.contractAddress;
      if (!addr) {
        console.log('  ⚠ 合约部署未返回地址');
        return;
      }

      deployedContracts.push({ address: addr, owner: from });
      console.log(`[高危] 部署合约: ${addr} (owner: ${from.substring(0, 10)})`);
      eventCount++;

      // 存入一些 ETH 到合约 (用 deposit selector)
      await safeSend({
        from, to: addr,
        value: web3.utils.toWei('2', 'ether'),
        gas: 100000,
        data: '0xd0e30db0' // deposit()
      });
      console.log(`  💰 存入 2 ETH 到合约`);
    } catch (e) {
      console.log(`  ⚠ 合约部署失败: ${e.message?.substring(0, 80)}`);
      eventCount++;
    }
  }

  // 场景6: ⚠️ 特权提升攻击 —— 非 owner 账户尝试调用 transferOwnership
  async function privilegeEscalationAttack() {
    if (deployedContracts.length === 0) {
      console.log('  无已部署合约，先部署一个...');
      await deployOwnableContract();
      if (deployedContracts.length === 0) return;
    }

    // 随机选一个已部署的合约
    const target = deployedContracts[Math.floor(Math.random() * deployedContracts.length)];

    // 选一个非 owner 的账户作为攻击者
    let attacker;
    do {
      attacker = accounts[Math.floor(Math.random() * Math.min(accounts.length, 6))];
    } while (attacker === target.owner);

    console.log(`[严重] 🔓 特权提升攻击:`);
    console.log(`  攻击者: ${attacker.substring(0, 10)}`);
    console.log(`  目标合约: ${target.address.substring(0, 10)} (owner: ${target.owner.substring(0, 10)})`);

    // 1. 尝试 transferOwnership (0xf2fde38b)
    const newOwnerArg = attacker.substring(2).toLowerCase().padStart(64, '0');
    const transferOwnershipData = '0xf2fde38b' + newOwnerArg;

    const tx1 = await safeSend({
      from: attacker,
      to: target.address,
      gas: 200000,
      data: transferOwnershipData
    });
    if (tx1) {
      console.log(`  📝 transferOwnership() 调用: ${tx1.transactionHash} (可能 revert)`);
    }

    // 2. 尝试 withdraw (0x3ccfd60b)
    const tx2 = await safeSend({
      from: attacker,
      to: target.address,
      gas: 200000,
      data: '0x3ccfd60b'
    });
    if (tx2) {
      console.log(`  📝 withdraw() 调用: ${tx2.transactionHash} (可能 revert)`);
    }
  }

  // 场景7: 合约滥用 —— 同一账户短时间内多次调用同一合约
  async function contractAbusePattern() {
    if (deployedContracts.length === 0) {
      console.log('  无已部署合约，先部署一个...');
      await deployOwnableContract();
      if (deployedContracts.length === 0) return;
    }

    const target = deployedContracts[Math.floor(Math.random() * deployedContracts.length)];
    const abuser = await pickSenderWithBalance(web3.utils.toWei('1', 'ether'));
    if (!abuser) { console.log('  跳过：无可用发送方'); return; }

    console.log(`[高危] 🔄 合约滥用模式:`);
    console.log(`  滥用者: ${abuser.substring(0, 10)}`);
    console.log(`  目标合约: ${target.address.substring(0, 10)}`);

    // 快速连续对同一合约发送多笔交易
    const callCount = 6 + Math.floor(Math.random() * 6); // 6~12 次
    console.log(`  发起 ${callCount} 次快速调用...`);

    for (let i = 0; i < callCount; i++) {
      // 混合不同类型的调用
      const callType = Math.random();
      let data, value;

      if (callType < 0.4) {
        // deposit()
        data = '0xd0e30db0';
        value = web3.utils.toWei(String(0.01 + Math.random() * 0.05), 'ether');
      } else if (callType < 0.7) {
        // getBalance() - 虽然是 view 但发 tx
        data = '0x12065fe0';
        value = '0';
      } else {
        // transferOwnership (unauthorized)
        const randomAddr = web3.utils.randomHex(20);
        data = '0xf2fde38b' + randomAddr.substring(2).padStart(64, '0');
        value = '0';
      }

      await safeSend({
        from: abuser,
        to: target.address,
        gas: 200000,
        value,
        data
      });
    }
    console.log(`  ✅ 合约滥用模式完成（${callCount} 笔调用）`);
  }

  // 场景8: 分散转账（洗钱模拟）
  async function scatterTransfers() {
    const ethPerTx = 2 + Math.random() * 3;
    const numTargets = 3 + Math.floor(Math.random() * 3);
    const totalNeeded = web3.utils.toWei(String(ethPerTx * numTargets + 1), 'ether');
    const from = await pickSenderWithBalance(totalNeeded);
    if (!from) { console.log('  跳过：无可用发送方'); return; }

    console.log(`[高危] 开始分散转账（${numTargets}笔）...`);
    const recipients = [];
    for (let i = 0; i < numTargets; i++) {
      let to;
      do { to = accounts[Math.floor(Math.random() * Math.min(accounts.length, 6))]; } while (to === from);
      const amount = web3.utils.toWei(String(ethPerTx * (0.8 + Math.random() * 0.4)), 'ether');
      const tx = await safeSend({ from, to, value: amount, gas: 21000 });
      if (tx) {
        recipients.push({ to, amount });
      }
    }
    console.log(`[高危] 分散转账完成，回收资金...`);
    for (const r of recipients) {
      const returnAmount = BigInt(r.amount) * 80n / 100n;
      await safeSend({ from: r.to, to: from, value: returnAmount.toString(), gas: 21000 });
    }
    console.log(`  ↩ 资金回收完成`);
  }

  // ==================== 主循环 ====================

  console.log('开始循环生成风险事件...\n');

  // 先部署 2 个合约，确保后续攻击场景有目标
  console.log('=== 初始化：部署合约 ===\n');
  await deployOwnableContract();
  await deployOwnableContract();
  console.log(`\n已部署 ${deployedContracts.length} 个 Ownable 合约\n`);

  const scenarios = [
    { fn: normalTransfer,            weight: 20, name: '普通转账' },
    { fn: largeTransfer,             weight: 15, name: '大额转账' },
    { fn: highGasTransaction,        weight: 12, name: '高Gas消耗' },
    { fn: rapidBurstTransfers,       weight: 8,  name: '快速连续转账' },
    { fn: deployOwnableContract,     weight: 5,  name: '部署合约' },
    { fn: privilegeEscalationAttack, weight: 15, name: '特权提升攻击' },
    { fn: contractAbusePattern,      weight: 15, name: '合约滥用' },
    { fn: scatterTransfers,          weight: 10, name: '分散转账' },
  ];

  const totalWeight = scenarios.reduce((sum, s) => sum + s.weight, 0);

  async function runRandomScenario() {
    const rand = Math.random() * totalWeight;
    let cumulative = 0;

    for (const scenario of scenarios) {
      cumulative += scenario.weight;
      if (rand < cumulative) {
        try {
          await scenario.fn();
        } catch (err) {
          console.error(`场景 [${scenario.name}] 执行失败:`, err.message?.substring(0, 100));
        }
        break;
      }
    }

    console.log(`--- 已生成 ${eventCount} 笔交易 | 已部署 ${deployedContracts.length} 个合约 ---\n`);
  }

  // 初始 burst
  console.log('=== 初始化：快速生成 10 笔交易 ===\n');
  for (let i = 0; i < 10; i++) {
    await runRandomScenario();
    await sleep(500);
  }

  // 持续生成
  console.log('\n=== 进入持续生成模式（每 3-8 秒一笔） ===\n');
  while (true) {
    await runRandomScenario();
    const delay = 3000 + Math.floor(Math.random() * 5000);
    await sleep(delay);
  }
}

function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

main().catch(err => {
  console.error('致命错误:', err);
  process.exit(1);
});
