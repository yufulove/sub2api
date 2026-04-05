import paramiko
import time

host = "43.156.127.125"
password = "Fiona364"

client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
# Add keepalive
client.connect(host, username="root", password=password, timeout=30)

print("[OK] Connected to Server\n")
print("Triggering install.sh via nohup...")
# Run install.sh via nohup
cmd = "source /etc/profile && source ~/.bashrc && export PATH=$PATH:/usr/local/go/bin && cd /home/work/deploy/sub2api && nohup sh /home/work/deploy/install.sh > /tmp/install.log 2>&1 &"
client.exec_command(cmd)

print("Started install.sh in background. Tailing logs...")

# Tail log
stdin, stdout, stderr = client.exec_command("tail -f /tmp/install.log")
try:
    for line in iter(stdout.readline, ""):
        print(line, end="")
        if "恭喜，所有步骤已顺利完成" in line or "部署成功" in line or "- 失败 -" in line or "已重新启动服务" in line:
            break
except Exception as e:
    print(f"Error reading output: {e}")

client.close()
