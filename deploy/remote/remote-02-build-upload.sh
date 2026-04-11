#!/bin/bash

###remot_build_upload.sh

### build-upload.sh

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

# ==================== 配置变量 ====================
DATE_DIR=$(date +%Y%m%d)
TIME_DIR=$(date +%H%M%S)

# 项目相关路径
WORK_DIR="/home/work/deploy/sub2api"          # 源码编译目录
DEPLOY_ROOT_DIR="/home/work/deploy/deploy-sub2api"              # 部署目标目录
DEPLOY_DIR="${DEPLOY_ROOT_DIR}/${DATE_DIR}"
BAK_DEPLOY_DIR="${DEPLOY_ROOT_DIR}/${DATE_DIR}${TIME_DIR}"

# 二进制文件名称
BIN_NAME="sub2api"

# 前端构建产物路径（相对于 backend 目录）
FRONTEND_DIST_PATH="internal/web/dist"

# 启动脚本名称
START_SCRIPT="nohup_start.sh"

# Git 相关配置
GIT_BRANCH="prod"                       # 要拉取的sub2adpi项目prod分支名称
BACKUP_BRANCH_PREFIX="tag"             # 备份分支前缀
# ==================================================

# 错误处理函数
error_exit() {
    echo -e "${RED}错误: $1${NC}"
    echo -e "${RED}脚本在第 $2 步失败，终止执行${NC}"
    exit 1
}

# 设置脚本在遇到任何错误时立即退出
set -e  # 任何命令失败（返回非0值）时退出脚本
set -u  # 使用未定义的变量时退出
set -o pipefail  # 管道命令中任何一个失败都会导致整体失败

# 记录开始时间
echo "========================================="
echo "开始构建 sub2api 项目"
echo "源码目录: ${WORK_DIR}"
echo "部署根目录: ${DEPLOY_ROOT_DIR}"
echo "备份目录: ${BAK_DEPLOY_DIR}"
echo "sub2api Git 分支: ${GIT_BRANCH}"
echo "时间: $(date)"
echo "========================================="

# 1. 进入项目目录并拉取最新代码
echo "步骤 1: 进入 ${WORK_DIR} 目录并拉取最新代码"
cd ${WORK_DIR} || error_exit "无法进入 ${WORK_DIR} 目录，请确认目录存在" 1

echo "成功进入: $(pwd)"
echo "开始拉取最新代码"
git reset --hard origin/${GIT_BRANCH} || error_exit "git reset 失败" 1
echo -e "${GREEN}拉取代码已完成${NC}"

echo -e "${GREEN}✅ 步骤 1: 进入目录并拉取最新代码完成${NC}"
echo "========================================="

# 2. 安装 pnpm（如果还没有安装）
echo ""
echo "步骤 2: 安装 pnpm"
if command -v pnpm &> /dev/null; then
    echo "pnpm 已安装，版本: $(pnpm --version)"
else
    echo "正在安装 pnpm..."
    npm install -g pnpm || error_exit "pnpm 安装失败" 2
    echo -e "${GREEN}pnpm 安装成功${NC}"
fi
echo -e "${GREEN}✅ 步骤 2: pnpm 安装检查完成${NC}"
echo "========================================="

# 3. 编译前端
echo ""
echo "步骤 3: 编译前端"
echo "进入 frontend 目录..."
cd frontend || error_exit "无法进入 frontend 目录" 3

echo "执行 pnpm install..."
pnpm install || error_exit "pnpm install 失败" 3

echo "执行 pnpm run build..."
pnpm run build || error_exit "前端构建失败" 3

echo -e "${GREEN}前端构建成功，产物已输出到 ../backend/${FRONTEND_DIST_PATH}/${NC}"
echo -e "${GREEN}✅ 步骤 3: 前端编译完成${NC}"
echo "========================================="

# 4. 编译后端（嵌入前端）
echo ""
echo "步骤 4: 编译后端"
cd ../backend || error_exit "无法进入 backend 目录" 4

echo "执行 go build..."
go build -tags embed -o ${BIN_NAME} ./cmd/server || error_exit "后端编译失败" 4

echo -e "${GREEN}后端编译成功${NC}"
echo -e "${GREEN}✅ 步骤 4: 后端编译完成${NC}"
echo "========================================="



# 5. 复制编译后的执行文件到目标位置
echo ""
echo "步骤 5: 复制编译后的执行文件到目标位置"

if [ -d "${DEPLOY_DIR}" ]; then
    # rm -rf ${DEPLOY_DIR} 2>/dev/null  # 删除旧的 frontend 软链接或目录
     cd ${DEPLOY_ROOT_DIR}
     mv ${DATE_DIR} ${DATE_DIR}${TIME_DIR} || error_exit "重命名前端目录失败" 5
    echo "备份老文件成功"
    
    cd ${DEPLOY_ROOT_DIR} 
    # 推送（如果远程仓库有README等文件，先拉取）
    git pull

    # 添加所有文件
    git add .

    # 提交
    git commit -m "删除提交,时间:${DATE_DIR}${TIME_DIR}"
    git push -u origin main
    echo "删除提交git成功"
fi

# 确保目标目录存在
mkdir -p ${DEPLOY_DIR}/backend || error_exit "创建后端目标目录失败" 5

# 复制后端文件
cp ${WORK_DIR}/backend/${BIN_NAME} ${DEPLOY_DIR}/backend/ || error_exit "复制后端文件失败" 5
echo "后端文件复制完成"

# 复制前端文件
cp -r ${WORK_DIR}/backend/${FRONTEND_DIST_PATH} ${DEPLOY_DIR}/ || error_exit "复制前端文件失败" 5
echo "前端文件复制完成"

# 重命名前端目录
cd ${DEPLOY_DIR} || error_exit "无法进入 ${DEPLOY_DIR} 目录" 5
if [ -d "dist" ]; then
    rm -rf frontend 2>/dev/null  # 删除旧的 frontend 软链接或目录
    mv dist frontend || error_exit "重命名前端目录失败" 5
    echo "前端目录重命名为 frontend"
fi

echo -e "${GREEN}✅ 步骤 5: 复制编译后的执行文件到目标位置完成${NC}"
echo "========================================="

# 6. 提交到制品包的git项目
cd ${DEPLOY_ROOT_DIR} || error_exit "无法进入 ${DEPLOY_ROOT_DIR} 目录" 6

# 配置用户信息
git config --global user.name "yufenfei2026"
git config --global user.email "yufenfei2026@126.com"
git config --global credential.helper store

# 推送（如果远程仓库有README等文件，先拉取）
#git pull origin main --rebase
git pull
git add .
git commit -m "打包提交,时间:${DATE_DIR}${TIME_DIR}"
git push -u origin main

echo -e "${GREEN}✅ 步骤 6: 提交git完成${NC}"
echo "========================================="
