import paramiko

host = "43.156.127.125"
password = "Fiona364"

client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
client.connect(host, username="root", password=password, timeout=10)

stdin, stdout, stderr = client.exec_command("cd /home/work && ls -la && echo '----' && ls -la deploy")
print(stdout.read().decode())
err = stderr.read().decode()
if err: print("Error:", err)
client.close()
