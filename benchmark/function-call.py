import subprocess
import re
import sys

from common.single import run
from parsers.response import Response
from common.invoker import newRequest
from common.invoker import request

def invoke(times, args = None):
    entries = request(times, args, reuse = False)
    return reduce(lambda total, time: total + time, Response(entries).series(Response.FIELD_RESPONSE)) / len(entries)


port = 8080
precision = 2
confidence = 0.98
if len(sys.argv) > 1:
    port = int(sys.argv[1])

req = newRequest(
    "GET",
    "http://localhost:{0}/".format(port),
    headers = {
        "X-FUNCTION": "hello"
    }
)

ret = run(precision, confidence, invoke, n = 10, args = req)
print("RRT to {3} of {0}% for a confidence level of {1}% would be {2}".format(precision, confidence * 100, ret, port))
