import paramiko
import time

ssh = paramiko.SSHClient()
ssh.set_missing_host_key_policy(paramiko.AutoAddPolicy())
ssh.connect("43.156.127.125", username="root", password="Fiona364", timeout=30, banner_timeout=30)

# 1. Find the actual openclaw binary
print("=== Finding binary ===")
_, o, _ = ssh.exec_command("find /root/.npm/_npx -name 'openclaw-gateway' -type f 2>/dev/null; find /root/.npm/_npx -name 'openclaw' -not -path '*/node_modules/openclaw/*' -type f 2>/dev/null | head -5", timeout=10)
bins = o.read().decode(errors="replace").strip()
print(bins)

# 2. Try using the node directly with the known path
_, o, _ = ssh.exec_command("ls /root/.npm/_npx/b5349d1f8847e5f1/node_modules/.bin/openclaw-gateway 2>/dev/null", timeout=10)
known_bin = o.read().decode(errors="replace").strip()
print(f"Known bin: {known_bin}")

# 3. Get node path
_, o, _ = ssh.exec_command("which node; source /root/.nvm/nvm.sh 2>/dev/null && which node", timeout=10)
node_path = o.read().decode(errors="replace").strip().split('\n')[-1]
print(f"Node: {node_path}")

# 4. Start using the full path
print("\n=== Starting gateway ===")
start_cmd = f"export PATH=/root/.nvm/versions/node/v22.22.2/bin:/root/.npm/_npx/b5349d1f8847e5f1/node_modules/.bin:$PATH; nohup openclaw-gateway --port 18789 > /tmp/openclaw-start.log 2>&1 &"
_, o, e = ssh.exec_command(start_cmd, timeout=10)
o.read(); e.read()

time.sleep(10)

# 5. If that didn't work, try npx with PATH set
_, o, _ = ssh.exec_command("ps aux | grep openclaw-gateway | grep -v grep", timeout=10)
proc = o.read().decode(errors="replace").strip()
if not proc:
    print("Direct start failed, trying npx...")
    start_cmd2 = "export PATH=/root/.nvm/versions/node/v22.22.2/bin:$PATH; nohup npx -y openclaw@latest gateway --port 18789 > /tmp/openclaw-start.log 2>&1 &"
    _, o, e = ssh.exec_command(start_cmd2, timeout=10)
    o.read(); e.read()
    time.sleep(12)
    _, o, _ = ssh.exec_command("ps aux | grep openclaw | grep -v grep", timeout=10)
    proc = o.read().decode(errors="replace").strip()

print(f"Process: {proc}")

# 6. Health
_, o, _ = ssh.exec_command("curl -s http://127.0.0.1:18789/health", timeout=10)
print(f"Health: {o.read().decode().strip()}")

# 7. Check startup log for errors
_, o, _ = ssh.exec_command("cat /tmp/openclaw-start.log 2>/dev/null | tail -10", timeout=10)
print(f"Log: {o.read().decode(errors='replace').strip()[:500]}")

ssh.close()
