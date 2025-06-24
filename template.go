package main

func getIndexTemplate() string {
	return `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>订阅转换工具</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }
        
        .container {
            background: white;
            border-radius: 20px;
            box-shadow: 0 20px 40px rgba(0,0,0,0.1);
            padding: 40px;
            width: 100%;
            max-width: 800px;
        }
        
        h1 {
            text-align: center;
            color: #333;
            margin-bottom: 30px;
            font-size: 2.5em;
            background: linear-gradient(135deg, #667eea, #764ba2);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }
        
        .form-group {
            margin-bottom: 25px;
        }
        
        label {
            display: block;
            margin-bottom: 8px;
            font-weight: 600;
            color: #555;
        }
        
        .input-type-selector {
            display: flex;
            gap: 15px;
            margin-bottom: 20px;
        }
        
        .radio-group {
            display: flex;
            align-items: center;
            gap: 8px;
        }
        
        input[type="radio"] {
            width: 18px;
            height: 18px;
        }
        
        input[type="url"], textarea {
            width: 100%;
            padding: 15px;
            border: 2px solid #e1e5e9;
            border-radius: 10px;
            font-size: 16px;
            transition: border-color 0.3s ease;
        }
        
        input[type="url"]:focus, textarea:focus {
            outline: none;
            border-color: #667eea;
        }
        
        textarea {
            min-height: 200px;
            resize: vertical;
            font-family: 'Courier New', monospace;
        }
        
        button {
            width: 100%;
            background: linear-gradient(135deg, #667eea, #764ba2);
            color: white;
            border: none;
            padding: 15px 30px;
            border-radius: 10px;
            font-size: 18px;
            font-weight: 600;
            cursor: pointer;
            transition: transform 0.2s ease;
        }
        
        button:hover {
            transform: translateY(-2px);
        }
        
        button:disabled {
            opacity: 0.6;
            cursor: not-allowed;
            transform: none;
        }
        
        .result {
            margin-top: 30px;
            padding: 20px;
            border-radius: 10px;
            display: none;
        }
        
        .result.success {
            background: #d4edda;
            border: 1px solid #c3e6cb;
            color: #155724;
        }
        
        .result.error {
            background: #f8d7da;
            border: 1px solid #f5c6cb;
            color: #721c24;
        }
        
        .subscription-url {
            background: #f8f9fa;
            padding: 15px;
            border-radius: 8px;
            border: 1px solid #dee2e6;
            margin: 15px 0;
            word-break: break-all;
            font-family: 'Courier New', monospace;
        }
        
        .copy-btn {
            background: #28a745;
            color: white;
            border: none;
            padding: 8px 16px;
            border-radius: 5px;
            cursor: pointer;
            font-size: 14px;
            margin-top: 10px;
        }
        
        .copy-btn:hover {
            background: #218838;
        }
        
        .stats {
            display: flex;
            justify-content: space-between;
            margin: 15px 0;
            font-weight: 600;
        }
        
        .loading {
            display: none;
            text-align: center;
            margin: 20px 0;
        }
        
        .spinner {
            border: 4px solid #f3f3f3;
            border-top: 4px solid #667eea;
            border-radius: 50%;
            width: 40px;
            height: 40px;
            animation: spin 1s linear infinite;
            margin: 0 auto 10px;
        }
        
        @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
        }
        
        .config-input {
            display: none;
        }
        
        .config-input.active {
            display: block;
        }
        
        .info-box {
            background: #e3f2fd;
            border: 1px solid #2196f3;
            border-radius: 10px;
            padding: 20px;
            margin-bottom: 30px;
            text-align: center;
        }
        
        .info-box h3 {
            color: #1976d2;
            margin-bottom: 10px;
        }
        
        .info-box p {
            color: #424242;
            line-height: 1.5;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🚀 订阅转换工具 (Go版)</h1>
        
        <div class="info-box">
            <h3>✨ 独立可执行文件版本</h3>
            <p>无需Python环境，直接运行！支持Windows、Linux、macOS跨平台部署</p>
            <br>
            <h3>🔄 新功能特性</h3>
            <p>• 智能去重：相同配置复用订阅链接，避免重复</p>
            <p>• URL配置实时更新（访问时自动检查最新内容）</p>
            <p>• 支持多个不同配置同时管理</p>
            <p>• <a href="/login" style="color: #667eea; text-decoration: none; font-weight: bold;">🔐 管理后台</a> - 查看所有订阅记录</p>
        </div>
        
        <form id="convertForm">
            <div class="form-group">
                <label>配置文件来源：</label>
                <div class="input-type-selector">
                    <div class="radio-group">
                        <input type="radio" id="url_source" name="config_source" value="url" checked>
                        <label for="url_source">URL链接</label>
                    </div>
                    <div class="radio-group">
                        <input type="radio" id="text_source" name="config_source" value="text">
                        <label for="text_source">直接输入</label>
                    </div>
                </div>
            </div>
            
            <div class="config-input active" id="url_input">
                <div class="form-group">
                    <label for="config_url">Clash 配置文件 URL：</label>
                    <input type="url" id="config_url" name="config_url" placeholder="https://example.com/config.yaml">
                </div>
            </div>
            
            <div class="config-input" id="text_input">
                <div class="form-group">
                    <label for="config_text">Clash 配置文件内容：</label>
                    <textarea id="config_text" name="config_text" placeholder="请粘贴完整的 Clash YAML 配置文件内容..."></textarea>
                </div>
            </div>
            
            <button type="submit" id="convertBtn">🎯 生成订阅链接</button>
        </form>
        
        <div class="loading" id="loading">
            <div class="spinner"></div>
            <p>正在转换配置文件，请稍候...</p>
        </div>
        
        <div class="result" id="result">
            <div id="result-content"></div>
        </div>
    </div>
    
    <script>
        // 切换输入方式
        document.querySelectorAll('input[name="config_source"]').forEach(radio => {
            radio.addEventListener('change', function() {
                document.querySelectorAll('.config-input').forEach(input => {
                    input.classList.remove('active');
                });
                document.getElementById(this.value + '_input').classList.add('active');
            });
        });
        
        // 表单提交
        document.getElementById('convertForm').addEventListener('submit', async function(e) {
            e.preventDefault();
            
            const formData = new FormData(this);
            const data = Object.fromEntries(formData.entries());
            
            // 显示加载状态
            document.getElementById('loading').style.display = 'block';
            document.getElementById('result').style.display = 'none';
            document.getElementById('convertBtn').disabled = true;
            
            try {
                const response = await fetch('/api/convert', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify(data)
                });
                
                const result = await response.json();
                
                // 隐藏加载状态
                document.getElementById('loading').style.display = 'none';
                document.getElementById('convertBtn').disabled = false;
                
                // 显示结果
                const resultDiv = document.getElementById('result');
                const resultContent = document.getElementById('result-content');
                
                if (result.success) {
                    resultDiv.className = 'result success';
                    const isAutoUpdate = document.querySelector('input[name="config_source"]:checked').value === 'url';
                    resultContent.innerHTML = ` + "`" + `
                        <h3>✅ ${result.message}</h3>
                        <div class="stats">
                            <span>节点数量: ${result.proxy_count}</span>
                            <span>订阅ID: ${result.subscription_id}</span>
                            <span>生成时间: ${new Date().toLocaleString()}</span>
                        </div>
                        <div class="subscription-url">
                            <strong>订阅链接：</strong><br>
                            <span id="sub-url">${result.subscription_url}</span>
                            <button class="copy-btn" onclick="copyToClipboard('sub-url')">📋 复制链接</button>
                        </div>
                        ${isAutoUpdate ? '<div style="background: #fff3cd; border: 1px solid #ffeaa7; border-radius: 5px; padding: 10px; margin: 10px 0; color: #856404;"><strong>🔄 实时更新：</strong> 此订阅链接每次访问时都会检查并获取最新内容</div>' : ''}
                        <p><strong>使用说明：</strong></p>
                        <ul>
                            <li>将上面的订阅链接复制到你的代理客户端中</li>
                            <li>支持 PassWall、V2rayN、Clash 等客户端</li>
                            <li>每个配置都有独立的订阅链接，不会相互干扰</li>
                            ${isAutoUpdate ? '<li>URL来源的配置会实时更新，每次访问都获取最新节点</li>' : '<li>文本输入的配置不会自动更新</li>'}
                        </ul>
                    ` + "`" + `;
                } else {
                    resultDiv.className = 'result error';
                    resultContent.innerHTML = ` + "`" + `
                        <h3>❌ 转换失败</h3>
                        <p>${result.message}</p>
                    ` + "`" + `;
                }
                
                resultDiv.style.display = 'block';
                
            } catch (error) {
                document.getElementById('loading').style.display = 'none';
                document.getElementById('convertBtn').disabled = false;
                
                const resultDiv = document.getElementById('result');
                const resultContent = document.getElementById('result-content');
                
                resultDiv.className = 'result error';
                resultContent.innerHTML = ` + "`" + `
                    <h3>❌ 网络错误</h3>
                    <p>请检查网络连接后重试</p>
                ` + "`" + `;
                resultDiv.style.display = 'block';
            }
        });
        
        // 复制到剪贴板
        function copyToClipboard(elementId) {
            const element = document.getElementById(elementId);
            const text = element.textContent;
            
            navigator.clipboard.writeText(text).then(function() {
                alert('已复制到剪贴板！');
            }).catch(function() {
                // fallback
                const textArea = document.createElement('textarea');
                textArea.value = text;
                document.body.appendChild(textArea);
                textArea.select();
                document.execCommand('copy');
                document.body.removeChild(textArea);
                alert('已复制到剪贴板！');
            });
        }
    </script>
</body>
</html>`
}

// 首次设置页面模板
const setupTemplate = `<!DOCTYPE html>
<html lang="zh-CN">
    <head>
        <meta charset="UTF-8">
        <meta name="viewport" content="width=device-width, initial-scale=1.0">
        <title>🔧 初始化管理员设置 - 订阅转换服务器</title>
        <style>
            * {
                margin: 0;
                padding: 0;
                box-sizing: border-box;
            }
            
            body {
                font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
                background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
                min-height: 100vh;
                display: flex;
                align-items: center;
                justify-content: center;
                padding: 20px;
            }
            
            .setup-container {
                background: white;
                border-radius: 20px;
                box-shadow: 0 20px 40px rgba(0,0,0,0.1);
                padding: 40px;
                width: 100%;
                max-width: 480px;
                text-align: center;
            }
            
            .setup-header {
                margin-bottom: 30px;
            }
            
            .setup-header h1 {
                color: #333;
                font-size: 2rem;
                margin-bottom: 10px;
            }
            
            .setup-header p {
                color: #666;
                font-size: 1rem;
            }
            
            .form-group {
                margin-bottom: 20px;
                text-align: left;
            }
            
            .form-group label {
                display: block;
                margin-bottom: 8px;
                color: #555;
                font-weight: 500;
            }
            
            .form-group input {
                width: 100%;
                padding: 12px 16px;
                border: 2px solid #e1e5e9;
                border-radius: 10px;
                font-size: 16px;
                transition: border-color 0.3s ease;
            }
            
            .form-group input:focus {
                outline: none;
                border-color: #667eea;
            }
            
            .setup-btn {
                width: 100%;
                padding: 14px;
                background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
                color: white;
                border: none;
                border-radius: 10px;
                font-size: 16px;
                font-weight: 600;
                cursor: pointer;
                transition: transform 0.2s ease;
            }
            
            .setup-btn:hover {
                transform: translateY(-2px);
            }
            
            .warning-box {
                background: #fff3cd;
                border: 1px solid #ffeaa7;
                border-radius: 10px;
                padding: 15px;
                margin-bottom: 20px;
                color: #856404;
            }
        </style>
    </head>
    <body>
        <div class="setup-container">
            <div class="setup-header">
                <h1>🔧 初始化设置</h1>
                <p>首次运行需要设置管理员账号</p>
            </div>
            
            <div class="warning-box">
                <strong>⚠️ 重要提示：</strong><br>
                请务必记住设置的账号密码，用于访问管理后台！
            </div>
            
            <form method="POST" action="/setup">
                <div class="form-group">
                    <label for="username">管理员用户名</label>
                    <input type="text" id="username" name="username" required placeholder="请输入用户名">
                </div>
                
                <div class="form-group">
                    <label for="password">管理员密码</label>
                    <input type="password" id="password" name="password" required placeholder="请输入密码（至少6位）">
                </div>
                
                <div class="form-group">
                    <label for="confirm_password">确认密码</label>
                    <input type="password" id="confirm_password" name="confirm_password" required placeholder="请再次输入密码">
                </div>
                
                <button type="submit" class="setup-btn">完成设置</button>
            </form>
        </div>
    </body>
</html>`

// 登录页面模板
const loginTemplate = `<!DOCTYPE html>
<html lang="zh-CN">
    <head>
        <meta charset="UTF-8">
        <meta name="viewport" content="width=device-width, initial-scale=1.0">
        <title>🔐 管理员登录 - 订阅转换服务器</title>
        <style>
            * {
                margin: 0;
                padding: 0;
                box-sizing: border-box;
            }
            
            body {
                font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
                background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
                min-height: 100vh;
                display: flex;
                align-items: center;
                justify-content: center;
                padding: 20px;
            }
            
            .login-container {
                background: white;
                border-radius: 20px;
                box-shadow: 0 20px 40px rgba(0,0,0,0.1);
                padding: 40px;
                width: 100%;
                max-width: 400px;
                text-align: center;
            }
            
            .login-header {
                margin-bottom: 30px;
            }
            
            .login-header h1 {
                color: #333;
                font-size: 2rem;
                margin-bottom: 10px;
            }
            
            .login-header p {
                color: #666;
                font-size: 1rem;
            }
            
            .form-group {
                margin-bottom: 20px;
                text-align: left;
            }
            
            .form-group label {
                display: block;
                margin-bottom: 8px;
                color: #555;
                font-weight: 500;
            }
            
            .form-group input {
                width: 100%;
                padding: 12px 16px;
                border: 2px solid #e1e5e9;
                border-radius: 10px;
                font-size: 16px;
                transition: border-color 0.3s ease;
            }
            
            .form-group input:focus {
                outline: none;
                border-color: #667eea;
            }
            
            .login-btn {
                width: 100%;
                padding: 14px;
                background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
                color: white;
                border: none;
                border-radius: 10px;
                font-size: 16px;
                font-weight: 600;
                cursor: pointer;
                transition: transform 0.2s ease;
            }
            
            .login-btn:hover {
                transform: translateY(-2px);
            }
            
            .login-btn:disabled {
                opacity: 0.6;
                cursor: not-allowed;
                transform: none;
            }
            
            .error-message {
                background: #ffe6e6;
                border: 1px solid #ffcccc;
                border-radius: 10px;
                padding: 10px;
                margin-top: 15px;
                color: #d63031;
                display: none;
            }
            
            .back-link {
                margin-top: 20px;
                text-align: center;
            }
            
            .back-link a {
                color: #667eea;
                text-decoration: none;
                font-size: 14px;
            }
            
            .back-link a:hover {
                text-decoration: underline;
            }
        </style>
    </head>
    <body>
        <div class="login-container">
            <div class="login-header">
                <h1>🔐 管理员登录</h1>
                <p>访问订阅管理后台</p>
            </div>
            
            <form id="loginForm">
                <div class="form-group">
                    <label for="username">用户名</label>
                    <input type="text" id="username" name="username" required>
                </div>
                
                <div class="form-group">
                    <label for="password">密码</label>
                    <input type="password" id="password" name="password" required>
                </div>
                
                <button type="submit" class="login-btn" id="loginBtn">登录</button>
                
                <div class="error-message" id="errorMessage"></div>
            </form>
            
            <div class="back-link">
                <a href="/">← 返回首页</a>
            </div>
        </div>
        
        <script>
            document.getElementById('loginForm').addEventListener('submit', async function(e) {
                e.preventDefault();
                
                const btn = document.getElementById('loginBtn');
                const errorDiv = document.getElementById('errorMessage');
                const username = document.getElementById('username').value;
                const password = document.getElementById('password').value;
                
                btn.disabled = true;
                btn.textContent = '登录中...';
                errorDiv.style.display = 'none';
                
                try {
                    const response = await fetch('/login', {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json',
                        },
                        body: JSON.stringify({
                            username: username,
                            password: password
                        })
                    });
                    
                    const result = await response.json();
                    
                    if (result.success) {
                        window.location.href = '/admin';
                    } else {
                        errorDiv.textContent = result.message;
                        errorDiv.style.display = 'block';
                    }
                } catch (error) {
                    errorDiv.textContent = '登录请求失败，请重试';
                    errorDiv.style.display = 'block';
                }
                
                btn.disabled = false;
                btn.textContent = '登录';
            });
        </script>
    </body>
</html>`

// 管理后台页面模板
const adminTemplate = `<!DOCTYPE html>
<html lang="zh-CN">
    <head>
        <meta charset="UTF-8">
        <meta name="viewport" content="width=device-width, initial-scale=1.0">
        <title>📊 管理后台 - 订阅转换服务器</title>
        <style>
            * {
                margin: 0;
                padding: 0;
                box-sizing: border-box;
            }
            
            body {
                font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
                background: #f5f7fa;
                min-height: 100vh;
            }
            
            .header {
                background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
                color: white;
                padding: 20px 0;
                box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            }
            
            .header-content {
                max-width: 1200px;
                margin: 0 auto;
                padding: 0 20px;
                display: flex;
                justify-content: space-between;
                align-items: center;
            }
            
            .header h1 {
                font-size: 1.8rem;
                font-weight: 600;
            }
            
            .header-actions {
                display: flex;
                gap: 15px;
            }
            
            .btn {
                padding: 8px 16px;
                border: none;
                border-radius: 8px;
                cursor: pointer;
                font-size: 14px;
                text-decoration: none;
                display: inline-block;
                transition: all 0.2s ease;
            }
            
            .btn-secondary {
                background: rgba(255,255,255,0.2);
                color: white;
            }
            
            .btn-secondary:hover {
                background: rgba(255,255,255,0.3);
            }
            
            .main-content {
                max-width: 1200px;
                margin: 30px auto;
                padding: 0 20px;
            }
            
            .stats-grid {
                display: grid;
                grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
                gap: 20px;
                margin-bottom: 30px;
            }
            
            .stat-card {
                background: white;
                border-radius: 15px;
                padding: 25px;
                box-shadow: 0 5px 15px rgba(0,0,0,0.08);
                text-align: center;
            }
            
            .stat-card .number {
                font-size: 2.5rem;
                font-weight: 700;
                color: #667eea;
                margin-bottom: 5px;
            }
            
            .stat-card .label {
                color: #666;
                font-size: 1rem;
            }
            
            .subscriptions-section {
                background: white;
                border-radius: 15px;
                padding: 25px;
                box-shadow: 0 5px 15px rgba(0,0,0,0.08);
            }
            
            .section-header {
                display: flex;
                justify-content: space-between;
                align-items: center;
                margin-bottom: 20px;
            }
            
            .section-header h2 {
                color: #333;
                font-size: 1.5rem;
            }
            
            .refresh-btn {
                background: #667eea;
                color: white;
                padding: 8px 16px;
                border: none;
                border-radius: 8px;
                cursor: pointer;
                font-size: 14px;
            }
            
            .refresh-btn:hover {
                background: #5a67d8;
            }
            
            .subscriptions-table {
                width: 100%;
                border-collapse: collapse;
                margin-top: 15px;
            }
            
            .subscriptions-table th,
            .subscriptions-table td {
                padding: 12px;
                text-align: left;
                border-bottom: 1px solid #e1e5e9;
            }
            
            .subscriptions-table th {
                background: #f8f9fa;
                font-weight: 600;
                color: #333;
            }
            
            .subscriptions-table td {
                color: #666;
            }
            
            .status-badge {
                padding: 4px 8px;
                border-radius: 12px;
                font-size: 12px;
                font-weight: 500;
            }
            
            .status-auto {
                background: #d4edda;
                color: #155724;
            }
            
            .status-manual {
                background: #fff3cd;
                color: #856404;
            }
            
            .subscription-id {
                font-family: 'Monaco', monospace;
                font-size: 12px;
                background: #f1f3f4;
                padding: 2px 6px;
                border-radius: 4px;
            }
            
            .loading {
                text-align: center;
                padding: 40px;
                color: #666;
            }
            
            .empty-state {
                text-align: center;
                padding: 60px 20px;
                color: #999;
            }
            
            .empty-state .icon {
                font-size: 4rem;
                margin-bottom: 20px;
            }
            
            @media (max-width: 768px) {
                .stats-grid {
                    grid-template-columns: 1fr;
                }
                
                .header-content {
                    flex-direction: column;
                    gap: 15px;
                }
                
                .section-header {
                    flex-direction: column;
                    gap: 15px;
                    align-items: flex-start;
                }
                
                .subscriptions-table {
                    font-size: 14px;
                }
                
                .subscriptions-table th,
                .subscriptions-table td {
                    padding: 8px;
                }
            }
        </style>
    </head>
    <body>
        <div class="header">
            <div class="header-content">
                <h1>📊 订阅管理后台</h1>
                <div class="header-actions">
                    <a href="/" class="btn btn-secondary">返回首页</a>
                    <a href="/logout" class="btn btn-secondary">退出登录</a>
                </div>
            </div>
        </div>
        
        <div class="main-content">
            <div class="stats-grid">
                <div class="stat-card">
                    <div class="number" id="totalSubscriptions">-</div>
                    <div class="label">总订阅数</div>
                </div>
                <div class="stat-card">
                    <div class="number" id="autoUpdateCount">-</div>
                    <div class="label">自动更新</div>
                </div>
                <div class="stat-card">
                    <div class="number" id="totalProxies">-</div>
                    <div class="label">总节点数</div>
                </div>
            </div>
            
            <div class="subscriptions-section">
                <div class="section-header">
                    <h2>订阅列表</h2>
                    <button class="refresh-btn" onclick="loadSubscriptions()">🔄 刷新数据</button>
                </div>
                
                <div id="subscriptionsContent">
                    <div class="loading">正在加载订阅数据...</div>
                </div>
            </div>
        </div>
        
        <script>
            async function loadSubscriptions() {
                const contentDiv = document.getElementById('subscriptionsContent');
                contentDiv.innerHTML = '<div class="loading">正在加载订阅数据...</div>';
                
                try {
                    const response = await fetch('/api/subscriptions');
                    const data = await response.json();
                    
                    if (data.success) {
                        renderSubscriptions(data.subscriptions);
                        updateStats(data.subscriptions);
                    } else {
                        contentDiv.innerHTML = '<div class="empty-state"><div class="icon">❌</div><h3>加载失败</h3><p>无法获取订阅数据</p></div>';
                    }
                } catch (error) {
                    contentDiv.innerHTML = '<div class="empty-state"><div class="icon">❌</div><h3>网络错误</h3><p>请检查网络连接后重试</p></div>';
                }
            }
            
            function renderSubscriptions(subscriptions) {
                const contentDiv = document.getElementById('subscriptionsContent');
                
                if (subscriptions.length === 0) {
                    contentDiv.innerHTML = '<div class="empty-state"><div class="icon">📝</div><h3>暂无订阅</h3><p>还没有生成任何订阅链接</p></div>';
                    return;
                }
                
                let tableHTML = ` + "`" + `
                    <table class="subscriptions-table">
                        <thead>
                            <tr>
                                <th>订阅ID</th>
                                <th>来源</th>
                                <th>节点数</th>
                                <th>更新方式</th>
                                <th>创建时间</th>
                                <th>最后更新</th>
                            </tr>
                        </thead>
                        <tbody>
                ` + "`" + `;
                
                subscriptions.forEach(sub => {
                    const createTime = new Date(sub.create_time).toLocaleString('zh-CN');
                    const updateTime = new Date(sub.last_update).toLocaleString('zh-CN');
                    const sourceType = sub.source_url ? 'URL' : '文本';
                    const source = sub.source_url || '手动输入';
                    const statusClass = sub.is_auto_update ? 'status-auto' : 'status-manual';
                    const statusText = sub.is_auto_update ? '自动更新' : '手动更新';
                    
                    tableHTML += ` + "`" + `
                        <tr>
                            <td><span class="subscription-id">${sub.id}</span></td>
                            <td title="${source}">${sourceType}</td>
                            <td>${sub.proxy_count}</td>
                            <td><span class="status-badge ${statusClass}">${statusText}</span></td>
                            <td>${createTime}</td>
                            <td>${updateTime}</td>
                        </tr>
                    ` + "`" + `;
                });
                
                tableHTML += '</tbody></table>';
                contentDiv.innerHTML = tableHTML;
            }
            
            function updateStats(subscriptions) {
                const total = subscriptions.length;
                const autoUpdate = subscriptions.filter(sub => sub.is_auto_update).length;
                const totalProxies = subscriptions.reduce((sum, sub) => sum + sub.proxy_count, 0);
                
                document.getElementById('totalSubscriptions').textContent = total;
                document.getElementById('autoUpdateCount').textContent = autoUpdate;
                document.getElementById('totalProxies').textContent = totalProxies;
            }
            
            // 页面加载时自动获取数据
            loadSubscriptions();
            
            // 每30秒自动刷新一次
            setInterval(loadSubscriptions, 30000);
        </script>
    </body>
</html>` 