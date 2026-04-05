import paramiko

host = "43.156.127.125"
password = "Fiona364"

client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
client.connect(host, username="root", password=password, timeout=10)

stdin, stdout, stderr = client.exec_command("cat /home/work/deploy/install.sh | grep -A 5 -B 5 git")
print(stdout.read().decode())
client.close()
