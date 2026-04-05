import paramiko

ssh = paramiko.SSHClient()
ssh.set_missing_host_key_policy(paramiko.AutoAddPolicy())
ssh.connect("43.156.127.125", username="root", password="Fiona364", timeout=10)

# Check openclaw gateway pair help
cmds = [
    # Check what pairing commands are available
    "export PATH=$PATH:/root/.nvm/versions/node/v22.22.2/bin && openclaw gateway --help 2>&1",
    # List pending pairing requests
    "export PATH=$PATH:/root/.nvm/versions/node/v22.22.2/bin && openclaw gateway pair --help 2>&1",
    # Check current config for auth/pairing settings
    "cat /root/.openclaw/openclaw.json | python3 -c 'import sys,json; c=json.load(sys.stdin); print(json.dumps(c.get(\"gateway\",{}),indent=2))'",
]

for cmd in cmds:
    print(f"\nCMD: {cmd.split('&&')[-1].strip()}")
    _, stdout, stderr = ssh.exec_command(cmd, timeout=15)
    out = stdout.read().decode().rstrip()
    err = stderr.read().decode().rstrip()
    if out: print(out)
    if err: print(f"ERR: {err}")

ssh.close()
