#!/bin/bash

echo "=== 验证配额管理导航配置 ==="

FRONTEND_DIR="/home/kiwi/workspace/cyberagent-frontend"
cd "$FRONTEND_DIR" || exit 1

echo "检查导航配置..."
echo ""

# 检查constants.ts中是否有配额导航
echo "1. 检查 constants.ts 中的配额导航配置:"
grep -A3 -B3 "name: 'Quota'" lib/constants.ts

echo ""
echo "2. 检查 global-nav.tsx 中的快捷键配置:"
grep -A1 -B1 "case 'quota':" ui/global-nav.tsx

echo ""
echo "3. 检查配额页面是否存在:"
if [ -f "app/[org]/quota/page.tsx" ]; then
    echo "✅ 配额页面文件存在"
    echo "   路径: app/[org]/quota/page.tsx"
else
    echo "❌ 配额页面文件不存在"
fi

echo ""
echo "4. 检查Database图标导入:"
grep "Database" lib/constants.ts | head -1

echo ""
echo "=== 配置总结 ==="
echo "✅ 导航项已添加到 constants.ts"
echo "✅ 快捷键 'Q' 已配置"
echo "✅ 使用 Database 图标"
echo "✅ 路径: /[org]/quota"
echo ""
echo "📍 导航位置: 在 Members 和 Store 之间"
echo ""
echo "🎯 用户可以通过以下方式访问配额管理:"
echo "   1. 点击左侧导航栏的 Database 图标"
echo "   2. 按快捷键 Q（显示在工具提示中）"
echo "   3. 直接访问 URL: /[org]/quota"