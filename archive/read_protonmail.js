const ProtonMail = require('protonmail-api');

async function readLatestEmail() {
  // 从环境变量获取用户名和密码
  const username = process.env.PM_USERNAME;
  const password = process.env.PM_PASSWORD;

  if (!username || !password) {
    console.error('错误: 请设置环境变量 PM_USERNAME 和 PM_PASSWORD');
    process.exit(1);
  }

  try {
    console.log('正在连接到 ProtonMail...');

    // 使用静态方法创建并连接 ProtonMail 实例
    const pm = await ProtonMail.connect({
      username: username,
      password: password
    });

    console.log('登录成功！');

    // 获取邮件列表
    const messages = await pm.getEmails();

    if (!messages || messages.length === 0) {
      console.log('没有找到邮件');
      await pm.close();
      return;
    }

    // 获取最新的邮件（第一封）
    const latestMessage = messages[0];

    // 输出发件人名称
    console.log('\n最新邮件信息:');
    console.log('发件人名:', latestMessage.from.name || '未知');
    console.log('发件人地址:', latestMessage.from.address || '未知');
    console.log('主题:', latestMessage.subject || '无主题');
    console.log('时间:', latestMessage.time.toLocaleString('zh-CN'));

    // 关闭连接
    await pm.close();
    console.log('\n已断开连接');

  } catch (error) {
    console.error('发生错误:', error.message);
    process.exit(1);
  }
}

// 运行主函数
readLatestEmail();
