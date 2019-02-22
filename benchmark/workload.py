import sys
import os

from common.invoker import newRequest
from common.invoker import request
from common.invoker import threadize

port = 8080
if len(sys.argv) > 1:
    port = int(sys.argv[1])

workload = 10000

req = newRequest(
    "GET",
    "http://localhost:{0}/".format(port),
    headers = {
        "X-FUNCTION": "hello"
    }
)

base = os.path.dirname(__file__)

entries = threadize(workload, req, num = 12, reuse = False)

fileObject = open(base + '/data/response.txt', 'w')
for i in entries:
    fileObject.write(",".join(map(lambda field: str(field), i)))
    fileObject.write('\n')
fileObject.close()
