#!/bin/bash

### remote_deploy.sh

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

# ==================== 配置变量 ====================
# 项目相关路径
WORK_DIR="/home/work/deploy/sub2api"          # 源码编译目录
DEPLOY_DIR="/opt/sub2api"              # 部署目标目录
BACKUP_BASE="/opt/backup"              # 备份基础目录

DATE_DIR=$(date +%Y%m%d)
TIME_DIR=$(date +%H%M%S)
BACKUP_DIR="${BACKUP_BASE}/${DATE_DIR}/${TIME_DIR}"
PKG_ROOT_DIR="/home/work/deploy/deploy-sub2api"
PKG_DIR="${PKG_ROOT_DIR}/${DATE_DIR}"

# 二进制文件名称
BIN_NAME="sub2api"

# 前端构建产物路径（相对于 backend 目录）
FRONTEND_DIST_PATH="internal/web/dist"

# 启动脚本名称
START_SCRIPT="nohup_start.sh"

# Git 相关配置
GIT_BRANCH="main"                       # 要拉取的分支名称
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


# 1. 进入项目目录并拉取最新代码
echo "步骤 1: 进入 ${PKG_ROOT_DIR} 目录并拉取最新代码"
cd ${PKG_ROOT_DIR} || error_exit "无法进入 ${PKG_ROOT_DIR} 目录，请确认目录存在" 1

echo "成功进入: $(pwd)"

echo "开始拉取最新代码"
#git reset --hard origin/${GIT_BRANCH} || error_exit "git reset 失败" 1
git pull || error_exit "git pull 失败" 1
echo -e "${GREEN}拉取代码已完成${NC}"

echo -e "${GREEN}✅ 步骤 1: 进入目录并拉取最新代码完成${NC}"
echo "========================================="


# 2. kill当前进程
echo ""
echo "步骤 2: kill 当前进程"
cd ${DEPLOY_DIR} || error_exit "无法进入部署目录 ${DEPLOY_DIR}" 2

pid=$(ps aux | grep "./backend/${BIN_NAME}" | grep -v grep | awk '{print $2}' || true)

if [ -n "$pid" ]; then
    echo "找到进程 PID: $pid"
    kill $pid || error_exit "kill 进程失败" 2
    
    # 等待进程完全退出
    sleep 2
    if ps -p $pid > /dev/null 2>&1; then
        echo "进程未完全退出，强制 kill..."
        kill -9 $pid || error_exit "强制 kill 进程失败" 2
    fi
    echo -e "${GREEN}已 kill 进程 $pid${NC}"
else
    echo -e "${YELLOW}未找到匹配的进程，无需 kill${NC}"
fi
echo -e "${GREEN}✅ 步骤 2: 进程清理完成${NC}"
echo "========================================="


# 3. 开始备份、清理文件
echo ""
echo "步骤 3: 开始备份、清理文件"

# 创建备份目录
if [ ! -d "$BACKUP_DIR" ]; then
    mkdir -p "$BACKUP_DIR" || error_exit "创建备份目录失败" 3
    echo "创建备份目录: $BACKUP_DIR"
else
    echo "备份目录已存在: $BACKUP_DIR"
fi

# 备份后端文件
if [ -f "${DEPLOY_DIR}/backend/${BIN_NAME}" ]; then
    cp ${DEPLOY_DIR}/backend/${BIN_NAME} ${BACKUP_DIR}/${BIN_NAME} || error_exit "备份后端文件失败" 3
    echo "备份[${DEPLOY_DIR}/backend/${BIN_NAME}]完成: ${BACKUP_DIR}/${BIN_NAME}"
    rm -rf ${DEPLOY_DIR}/backend/${BIN_NAME} || error_exit "删除旧后端文件失败" 3
    echo "已删除旧的后端文件"
fi

# 备份前端文件
if [ -d "${DEPLOY_DIR}/frontend" ]; then
    cp -R ${DEPLOY_DIR}/frontend ${BACKUP_DIR}/frontend || error_exit "备份前端文件失败" 3
    echo "备份[${DEPLOY_DIR}/frontend]完成: ${BACKUP_DIR}/frontend"
    rm -rf ${DEPLOY_DIR}/frontend || error_exit "删除旧前端文件失败" 3
    echo "已删除旧的前端目录"
fi

echo -e "${GREEN}✅ 步骤 3: 备份清理完成${NC}"
echo "========================================="

# 4. 复制编译后的执行文件到目标位置
echo ""
echo "步骤 4: 复制编译后的执行文件到目标位置"

# 确保目标目录存在
mkdir -p ${DEPLOY_DIR}/backend || error_exit "创建后端目标目录失败" 4

# 复制后端文件
cp ${PKG_DIR}/backend/${BIN_NAME} ${DEPLOY_DIR}/backend/ || error_exit "复制后端文件失败" 4
echo "后端文件复制完成"

# 复制前端文件
cp -r ${PKG_DIR}/frontend ${DEPLOY_DIR}/ || error_exit "复制前端文件失败" 4
echo "前端文件复制完成"

echo -e "${GREEN}✅ 步骤 4: 复制编译后的执行文件到目标位置完成${NC}"
echo "========================================="

# 5. 开始启动应用
echo ""
echo "步骤 5: 开始启动应用"

cd ${DEPLOY_DIR} || error_exit "无法进入 ${DEPLOY_DIR} 目录" 5

# 检查启动脚本是否存在
if [ ! -f "${START_SCRIPT}" ]; then
    error_exit "启动脚本 ${START_SCRIPT} 不存在" 5
fi

# 给脚本添加执行权限
chmod +x ${START_SCRIPT} || error_exit "修改启动脚本权限失败" 5

# 执行启动脚本
./${START_SCRIPT} || error_exit "启动应用失败" 5

echo -e "${GREEN}✅ 步骤 5: 启动命令已执行${NC}"
echo "========================================="

# 6. 检查启动是否成功
echo ""
echo "步骤 6: 检查启动是否成功"

sleep 5

pid=$(ps aux | grep "./backend/${BIN_NAME}" | grep -v grep | awk '{print $2}' || true)

if [ -n "$pid" ]; then
    echo -e "${GREEN}✅ 找到进程 PID: $pid，启动成功！${NC}"
else
    # 检查日志文件是否存在
    if [ -f "${DEPLOY_DIR}/nohup.out" ]; then
        echo -e "${RED}启动失败，最后几行日志：${NC}"
        tail -20 ${DEPLOY_DIR}/nohup.out
    fi
    error_exit "未找到匹配的进程，启动失败，请检查日志" 6
fi

echo -e "${GREEN}✅ 步骤 6: 启动结果检查完成${NC}"
echo "========================================="


# 完成
echo ""
echo "========================================="
echo -e "${GREEN}✅ 部署全部完成！${NC}"
echo "源码目录: ${WORK_DIR}"
echo "部署目录: ${DEPLOY_DIR}"
echo "时间: $(date)"
echo "========================================="
