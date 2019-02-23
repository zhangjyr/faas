import subprocess
import re
import sys
import os
import traceback

from common.single import run
from graphs.parsers.response import Response
from common.invoker import newRequest, request, threadize
from common.recorder import Recorder

def invoke(times, args = None):
    entries = threadize(times, args["req"], num = args["thread"], reuse = True)
    recorder.save(entries)

    filtered = Response(entries).filter(lambda entry: entry[Response.FIELD_STATUSCODE] == 200)
    throttled = len(entries) - filtered.length()
    if throttled > 0:
        print("{0} Throttled".format(len(entries) - filtered.length()))

    if filtered.length() < 2:
        return 0
    return reduce(
        lambda total, time: total + time,
        filtered.series(Response.FIELD_RESPONSE)
        ) / len(entries)

port = 8080
thread = 1
function = "hello"
precision = 1
confidence = 0.99
if len(sys.argv) > 1:
    for i in range(1, len(sys.argv)):
        try:
            val = int(sys.argv[i])
            if val <= 1024:
                thread = val
            else:
                port = val
        except ValueError:
            function = sys.argv[i]

req = newRequest(
    "GET",
    "http://server:{0}/".format(port),
    headers = {
        "X-FUNCTION": "hello"
    }
)

base = os.path.dirname(__file__)
if base == '':
    base = '.'

recorder = Recorder(base + '/data/')
recorder.open(function + '_' + str(thread) + '.csv')

ret = None
try:
    ret = run(precision, confidence, invoke, n = 500, args = {
        "req": req,
        "thread": thread
    })
except BaseException:
    traceback.print_exc()

print("Wait for recorder")
recorder.close()

if ret:
    print("RRT to {3} of {0}% for a confidence level of {1}% would be {2}".format(precision, confidence * 100, ret, port))
