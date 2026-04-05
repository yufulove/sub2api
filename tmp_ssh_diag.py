import paramiko
import json
import time

host = "43.156.127.125"
password = "Fiona364"

client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
client.connect(host, username="root", password=password, timeout=10)
print("[OK] Connected\n")

def run(cmd, timeout=30):
    save_cmd = f"({cmd}) > /tmp/_fix.txt 2>&1"
    stdin, stdout, stderr = client.exec_command(save_cmd, timeout=timeout)
    stdout.channel.recv_exit_status()
    stdin2, stdout2, stderr2 = client.exec_command("cat /tmp/_fix.txt", timeout=15)
    out = stdout2.read().decode('utf-8', errors='replace').strip()
    print(f"$ {cmd}")
    print(out if out else "(empty)")
    print("-" * 50)
    return out

# 1. Read current config
stdin, stdout, stderr = client.exec_command("cat /root/.openclaw/openclaw.json", timeout=15)
config_raw = stdout.read().decode('utf-8', errors='replace')
config = json.loads(config_raw)
print("[OK] Read current config\n")

# 2. Backup
run("cp /root/.openclaw/openclaw.json /root/.openclaw/openclaw.json.bak.timeout")

# 3. Add timeout settings
# Add llm.idleTimeoutSeconds to agents.defaults
if "llm" not in config["agents"]["defaults"]:
    config["agents"]["defaults"]["llm"] = {}
config["agents"]["defaults"]["llm"]["idleTimeoutSeconds"] = 300
print("[FIX] Added agents.defaults.llm.idleTimeoutSeconds = 300")

# Add timeoutSeconds to agents.defaults
config["agents"]["defaults"]["timeoutSeconds"] = 600
print("[FIX] Added agents.defaults.timeoutSeconds = 600")

# 4. Write the fixed config
fixed_json = json.dumps(config, indent=2, ensure_ascii=False)
write_cmd = f"cat > /root/.openclaw/openclaw.json << 'EOFCONFIG'\n{fixed_json}\nEOFCONFIG"
stdin2, stdout2, stderr2 = client.exec_command(write_cmd, timeout=15)
stdout2.channel.recv_exit_status()
print("[OK] Written updated config\n")

# 5. Verify JSON validity
run("python3 -c \"import json; c=json.load(open('/root/.openclaw/openclaw.json')); print('JSON valid'); print('idleTimeout:', c['agents']['defaults'].get('llm',{}).get('idleTimeoutSeconds')); print('timeoutSeconds:', c['agents']['defaults'].get('timeoutSeconds'))\"")

# 6. Kill existing openclaw processes
run("pkill -9 -f 'openclaw' 2>/dev/null; sleep 2; echo 'killed all openclaw'")

# 7. Start OpenClaw with nohup
run("source /root/.nvm/nvm.sh && nohup npx -y openclaw@latest gateway --port 18789 >> /tmp/openclaw_restart.log 2>&1 & echo 'Started PID:' $!", timeout=15)

# 8. Wait for startup
print("\nWaiting 20 seconds for OpenClaw to start...")
time.sleep(20)

# 9. Check if listening
run("ss -tlnp | grep 18789")

# 10. Check process
run("ps aux | grep 'openclaw-gateway' | grep -v grep | head -3")

# 11. Test health
run("curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:18789/")

# 12. Restart auto_approve loop
run("nohup bash /root/.openclaw/auto_approve_loop.sh >> /root/.openclaw/auto_approve.log2 2>&1 & echo 'auto_approve restarted'")

# Cleanup
run("rm -f /tmp/_fix.txt /tmp/openclaw_restart.log")

client.close()
print("\n[DONE] Config updated and service restarted")
