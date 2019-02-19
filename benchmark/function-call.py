import subprocess
import re
import sys

from common.single import run
from parsers.response import Response
from common.invoker import newRequest
from common.invoker import request
from common.invoker import threadize

def invoke(times, args = None):
    entries = threadize(times, args, num = 8, reuse = False)
    filtered = Response(entries).filter(lambda entry: entry[Response.FIELD_STATUSCODE] == 200)
    print("{0} Throttled".format(len(entries) - filtered.length()))
    if filtered.length() < 2:
        return 0
        
    return reduce(
        lambda total, time: total + time,
        filtered.series(Response.FIELD_RESPONSE)
        ) / len(entries)


port = 8080
precision = 1
confidence = 0.99
if len(sys.argv) > 1:
    port = int(sys.argv[1])

req = newRequest(
    "GET",
    "http://localhost:{0}/".format(port),
    headers = {
        "X-FUNCTION": "hello"
    }
)

ret = run(precision, confidence, invoke, n = 500, args = req)
print("RRT to {3} of {0}% for a confidence level of {1}% would be {2}".format(precision, confidence * 100, ret, port))
