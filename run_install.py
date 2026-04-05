import paramiko

host = "43.156.127.125"
password = "Fiona364"

client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
client.connect(host, username="root", password=password, timeout=10)

print("[OK] Connected to Server\n")
print("Executing install.sh...")
# 确保先拉取最新代码再打包
cmd = "cd /home/work/deploy/sub2api && git fetch --all && sh /home/work/deploy/install.sh"
stdin, stdout, stderr = client.exec_command(cmd, get_pty=True)

for line in iter(stdout.readline, ""):
    print(line, end="")

status = stdout.channel.recv_exit_status()
print(f"\n[DONE] install.sh exited with status: {status}")

client.close()
