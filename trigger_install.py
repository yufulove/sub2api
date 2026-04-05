import paramiko
import time

host = "43.156.127.125"
password = "Fiona364"

client = paramiko.SSHClient()
client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
client.connect(host, username="root", password=password, timeout=30)

print("Killing active install and re-running with git fetch...")
client.exec_command("pkill -f install.sh ; pkill -f vite ; pkill -f go")
time.sleep(2)

cmd = "source /etc/profile && source ~/.bashrc && export PATH=$PATH:/usr/local/go/bin && cd /home/work/deploy/sub2api && git fetch --all && git reset --hard origin/main && nohup sh /home/work/deploy/install.sh > /tmp/install4.log 2>&1 &"
client.exec_command(cmd)

print("Started install.sh in background!")
client.close()
