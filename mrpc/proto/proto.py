import sys
import os
import subprocess

os.environ["path"] = ";".join([os.environ["path"], "C:\\p\\protoc-24.1-win64\\bin"])

proto_dir = os.path.dirname(os.path.abspath(__file__))

result = subprocess.run(f"protoc.exe --proto_path={proto_dir} --go_out={proto_dir} --go-grpc_out={proto_dir} msg.proto", shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, encoding='utf-8')

print(result.stdout)
print(result.stderr)